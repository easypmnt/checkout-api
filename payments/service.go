package payments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/easypmnt/checkout-api/repository"
	"github.com/google/uuid"
)

type (
	Service struct {
		repo paymentRepository
		sol  solanaClient
		jup  jupiterClient
		conf Config
	}
)

// NewService creates a new payment service instance.
func NewService(repo paymentRepository, sol solanaClient, jup jupiterClient, conf Config) *Service {
	return &Service{
		repo: repo,
		sol:  sol,
		jup:  jup,
		conf: conf,
	}
}

// CreatePayment creates a new payment.
func (s *Service) CreatePayment(ctx context.Context, payment *Payment) (*Payment, error) {
	payment = s.mergePaymentWithDefaultConfig(payment)
	if payment.Amount == 0 {
		return nil, fmt.Errorf("payment amount must be greater than 0")
	}
	payment.DestinationMint = MintAddress(payment.DestinationMint, s.conf.DestinationMint)

	result, err := s.repo.CreatePayment(ctx, repository.CreatePaymentParams{
		ExternalID:        sql.NullString{String: payment.ExternalID, Valid: payment.ExternalID != ""},
		DestinationWallet: payment.DestinationWallet,
		DestinationMint:   payment.DestinationMint,
		Amount:            int64(payment.Amount),
		Status:            repository.PaymentStatusNew,
		Message:           sql.NullString{String: payment.Message, Valid: payment.Message != ""},
		ExpiresAt:         sql.NullTime{Time: *payment.ExpiresAt, Valid: payment.ExpiresAt != nil},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return castFromRepositoryPayment(result), nil
}

// GetPayment returns the payment with the given ID.
func (s *Service) GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error) {
	result, err := s.repo.GetPayment(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return castFromRepositoryPayment(result), nil
}

// GetPaymentByExternalID returns the payment with the given external ID.
func (s *Service) GetPaymentByExternalID(ctx context.Context, externalID string) (*Payment, error) {
	result, err := s.repo.GetPaymentByExternalID(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	return castFromRepositoryPayment(result), nil
}

// GeneratePaymentLink generates a new payment link for the given payment.
func (s *Service) GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID, mint string, applyBonus bool) (string, error) {
	payment, err := s.GetPayment(ctx, paymentID)
	if err != nil {
		return "", fmt.Errorf("failed to get payment: %w", err)
	}
	if payment.Status != PaymentStatusNew && payment.Status != PaymentStatusPending {
		return "", fmt.Errorf("payment already %s", payment.Status)
	}

	mint = MintAddress(mint, payment.DestinationMint)

	uri := strings.Join([]string{
		strings.TrimRight(s.conf.SolPayBaseURL, "/"),
		strings.Trim(paymentID.String(), "/"),
		strings.Trim(mint, "/"),
		strconv.FormatBool(applyBonus),
	}, "/")

	return fmt.Sprintf("solana:%s", uri), nil
}

// UpdatePaymentStatus updates the status of the payment with the given ID.
func (s *Service) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status PaymentStatus) error {
	if _, err := s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
		ID:     id,
		Status: castToRepositoryPaymentStatus(status),
	}); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// CancelPayment cancels the payment with the given ID.
func (s *Service) CancelPayment(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
		ID:     id,
		Status: repository.PaymentStatusCanceled,
	}); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// CancelPaymentByExternalID cancels the payment with the given external ID.
func (s *Service) CancelPaymentByExternalID(ctx context.Context, externalID string) error {
	payment, err := s.GetPaymentByExternalID(ctx, externalID)
	if err != nil {
		return err
	}

	if _, err := s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
		ID:     payment.ID,
		Status: repository.PaymentStatusCanceled,
	}); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// BuildTransaction builds a new transaction for the given payment.
func (s *Service) BuildTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	if tx.PaymentID == uuid.Nil {
		return nil, fmt.Errorf("payment ID is required")
	}
	if tx.SourceWallet == "" {
		return nil, fmt.Errorf("sender wallet address is required")
	}
	payment, err := s.GetPayment(ctx, tx.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	if payment.Status != PaymentStatusNew && payment.Status != PaymentStatusPending {
		return nil, fmt.Errorf("payment already %s", payment.Status)
	}
	payment.DestinationMint = MintAddress(payment.DestinationMint, s.conf.DestinationMint)
	tx.SourceMint = MintAddress(tx.SourceMint, payment.DestinationMint)

	base64Tx, tx, err := NewPaymentTransactionBuilder(s.sol, s.jup, s.conf).
		SetTransaction(tx, payment).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	repoTx, err := s.repo.CreateTransaction(ctx, repository.CreateTransactionParams{
		PaymentID:          tx.PaymentID,
		Reference:          tx.Reference,
		SourceWallet:       tx.SourceWallet,
		SourceMint:         tx.SourceMint,
		DestinationWallet:  tx.DestinationWallet,
		DestinationMint:    tx.DestinationMint,
		Amount:             int64(tx.Amount),
		DiscountAmount:     int64(tx.DiscountAmount),
		TotalAmount:        int64(tx.TotalAmount),
		Message:            sql.NullString{String: tx.Message, Valid: tx.Message != ""},
		Memo:               sql.NullString{String: tx.Memo, Valid: tx.Memo != ""},
		ApplyBonus:         sql.NullBool{Bool: tx.ApplyBonus, Valid: true},
		AccruedBonusAmount: int64(tx.AccruedBonusAmount),
		Status:             repository.TransactionStatusPending,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	result := castFromRepositoryTransaction(repoTx, s.conf)
	result.Transaction = base64Tx

	return result, nil
}

// GetTransactionByReference returns the transaction with the given reference.
func (s *Service) GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error) {
	result, err := s.repo.GetTransactionByReference(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction by reference=%s: %w", reference, err)
	}

	return castFromRepositoryTransaction(result, s.conf), nil
}

// MarkPaymentsAsExpired marks all payments that are expired as expired.
func (s *Service) MarkPaymentsAsExpired(ctx context.Context) error {
	if err := s.repo.MarkPaymentsExpired(ctx); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to mark payments as expired: %w", err)
		}
	}

	return nil
}

// UpdateTransaction updates the status and signature of the transaction with the given reference.
func (s *Service) UpdateTransaction(ctx context.Context, reference string, status TransactionStatus, signature string) error {
	if _, err := s.repo.UpdateTransactionByReference(ctx, repository.UpdateTransactionByReferenceParams{
		Reference:   reference,
		Status:      castToRepositoryTransactionStatus(status),
		TxSignature: sql.NullString{String: signature, Valid: signature != ""},
	}); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

// GetPendingTransactions returns all pending transactions.
func (s *Service) GetPendingTransactions(ctx context.Context) ([]*Transaction, error) {
	pendingTxs, err := s.repo.GetPendingTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}

	result := make([]*Transaction, 0, len(pendingTxs))
	for _, tx := range pendingTxs {
		result = append(result, castFromRepositoryTransaction(tx, s.conf))
	}

	return result, nil
}

// MarkTransactionsAsExpired marks all transactions that are expired as expired.
func (s *Service) MarkTransactionsAsExpired(ctx context.Context) error {
	if err := s.repo.MarkTransactionsAsExpired(ctx); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to mark transactions as expired: %w", err)
		}
	}

	return nil
}

func (s *Service) mergePaymentWithDefaultConfig(payment *Payment) *Payment {
	if payment.DestinationWallet == "" {
		payment.DestinationWallet = s.conf.DestinationWallet
	}
	if payment.DestinationMint == "" {
		payment.DestinationMint = s.conf.DestinationMint
	}
	if payment.ExpiresAt == nil {
		payment.ExpiresAt = utils.Pointer(time.Now().Add(s.conf.PaymentTTL))
	}
	return payment
}
