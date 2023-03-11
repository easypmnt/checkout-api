package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type QueriesTx struct {
	*Queries
	dbConn *sql.DB
}

// NewWithConnection creates a new QueriesTx with a database connection to use for transactions.
// It is the caller's responsibility to close the connection.
func NewWithConnection(ctx context.Context, dbConn *sql.DB) (*QueriesTx, error) {
	q, err := Prepare(ctx, dbConn)
	if err != nil {
		return nil, err
	}

	return &QueriesTx{
		Queries: q,
		dbConn:  dbConn,
	}, nil
}

// Represents a complete payment info, including payment, destinations and transactions.
type PaymentInfo struct {
	Payment      Payment
	Destinations []PaymentDestination
	Transactions []Transaction
}

// CreatePaymentWithDestinationsParams is a struct for CreatePaymentWithDestinations method
type CreatePaymentWithDestinationsParams struct {
	Payment      CreatePaymentParams
	Destinations []CreatePaymentDestinationParams
}

// CreatePaymentWithDestinations creates a new payment with destinations
func (q *QueriesTx) CreatePaymentWithDestinations(ctx context.Context, arg CreatePaymentWithDestinationsParams) (PaymentInfo, error) {
	tx, err := q.dbConn.Begin()
	if err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repo := q.WithTx(tx)

	if arg.Payment.ExternalID.Valid {
		if _, err := repo.GetPaymentByExternalID(ctx, arg.Payment.ExternalID.String); !errors.Is(err, sql.ErrNoRows) {
			return PaymentInfo{}, fmt.Errorf("payment with external ID %s already exists", arg.Payment.ExternalID.String)
		}
	}

	payment, err := repo.CreatePayment(ctx, arg.Payment)
	if err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to create payment: %w", err)
	}

	destinations := make([]PaymentDestination, 0, len(arg.Destinations))
	for _, dest := range arg.Destinations {
		dest.PaymentID = payment.ID
		destination, err := repo.CreatePaymentDestination(ctx, dest)
		if err != nil {
			return PaymentInfo{}, fmt.Errorf("failed to create payment destination: %w", err)
		}
		destinations = append(destinations, destination)
	}

	if err := tx.Commit(); err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return PaymentInfo{
		Payment:      payment,
		Destinations: destinations,
	}, nil
}

// CreateTransactionWithCallbackParams is a struct for CreateTransactionWithCallback method
type CreateTransactionWithCallbackParams struct {
	Transaction  CreateTransactionParams
	Destinations []CreatePaymentDestinationParams
	Callback     func() error
}

// CreateTransactionWithCallback creates a new transaction with callback
func (q *QueriesTx) CreateTransactionWithCallback(ctx context.Context, arg CreateTransactionWithCallbackParams) (Transaction, error) {
	tx, err := q.dbConn.Begin()
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repo := q.WithTx(tx)

	transaction, err := repo.CreateTransaction(ctx, arg.Transaction)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	if len(arg.Destinations) > 0 {
		if err := repo.DeletePaymentDestinations(ctx, transaction.PaymentID); err != nil {
			return Transaction{}, fmt.Errorf("failed to delete payment destinations: %w", err)
		}

		for _, createParams := range arg.Destinations {
			if _, err := repo.CreatePaymentDestination(ctx, createParams); err != nil {
				return Transaction{}, fmt.Errorf("failed to create payment destination: %w", err)
			}
		}
	}

	if _, err := repo.UpdatePaymentStatus(ctx, UpdatePaymentStatusParams{
		ID:     transaction.PaymentID,
		Status: PaymentStatusPending,
	}); err != nil {
		return Transaction{}, fmt.Errorf("failed to update payment status: %w", err)
	}

	if err := arg.Callback(); err != nil {
		return Transaction{}, fmt.Errorf("failed to execute callback: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Transaction{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return transaction, nil
}

// UpdateTransactionParams is a struct for UpdateTransaction method
type UpdateTransactionParams struct {
	Status      TransactionStatus `json:"status"`
	Reference   string            `json:"reference"`
	TxSignature string            `json:"tx_signature"`
}

// UpdateTransaction updates a transaction
func (q *QueriesTx) UpdateTransaction(ctx context.Context, arg UpdateTransactionParams) (Transaction, error) {
	tx, err := q.dbConn.Begin()
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repo := q.WithTx(tx)

	transaction, err := repo.GetTransactionByReference(ctx, arg.Reference)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to get transaction: %w", err)
	}

	transaction, err = repo.UpdateTransactionByReference(ctx, UpdateTransactionByReferenceParams{
		TxSignature: sql.NullString{String: arg.TxSignature, Valid: arg.TxSignature != ""},
		Status:      arg.Status,
		Reference:   arg.Reference,
	})
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to update transaction: %w", err)
	}

	payment, err := repo.GetPayment(ctx, transaction.PaymentID)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.Status == PaymentStatusCompleted {
		return Transaction{}, errors.New("payment is already completed")
	}

	paymentStatus := payment.Status
	switch arg.Status {
	case TransactionStatusPending:
		paymentStatus = PaymentStatusPending
	case TransactionStatusCompleted:
		paymentStatus = PaymentStatusCompleted
	case TransactionStatusFailed:
		paymentStatus = PaymentStatusFailed
	}

	if _, err := repo.UpdatePaymentStatus(ctx, UpdatePaymentStatusParams{
		ID:     transaction.PaymentID,
		Status: paymentStatus,
	}); err != nil {
		return Transaction{}, fmt.Errorf("failed to update payment status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Transaction{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return transaction, nil
}

// GetPaymentInfo returns completed payment info
func (q *QueriesTx) GetPaymentInfo(ctx context.Context, paymentID uuid.UUID) (PaymentInfo, error) {
	payment, err := q.GetPayment(ctx, paymentID)
	if err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to get payment: %w", err)
	}

	destinations, err := q.GetPaymentDestinations(ctx, paymentID)
	if err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to get payment destinations: %w", err)
	}

	transactions, err := q.GetTransactionsByPaymentID(ctx, paymentID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return PaymentInfo{}, fmt.Errorf("failed to get transactions: %w", err)
		}
	}

	return PaymentInfo{
		Payment:      payment,
		Destinations: destinations,
		Transactions: transactions,
	}, nil
}

// GetPaymentInfoByExternalID returns completed payment info by external id
func (q *QueriesTx) GetPaymentInfoByExternalID(ctx context.Context, externalID string) (PaymentInfo, error) {
	payment, err := q.GetPaymentByExternalID(ctx, externalID)
	if err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to get payment: %w", err)
	}

	destinations, err := q.GetPaymentDestinations(ctx, payment.ID)
	if err != nil {
		return PaymentInfo{}, fmt.Errorf("failed to get payment destinations: %w", err)
	}

	transactions, err := q.GetTransactionsByPaymentID(ctx, payment.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return PaymentInfo{}, fmt.Errorf("failed to get transactions: %w", err)
		}
	}

	return PaymentInfo{
		Payment:      payment,
		Destinations: destinations,
		Transactions: transactions,
	}, nil
}

// UpdatePaymentDestinationsParams is a struct for UpdatePaymentDestinations method
type UpdatePaymentDestinationsParams struct {
	PaymentID    uuid.UUID
	Destinations []CreatePaymentDestinationParams
}

// UpdatePaymentDestinations updates payment destinations
func (q *QueriesTx) UpdatePaymentDestinations(ctx context.Context, arg UpdatePaymentDestinationsParams) error {
	tx, err := q.dbConn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repo := q.WithTx(tx)

	if err := repo.DeletePaymentDestinations(ctx, arg.PaymentID); err != nil {
		return fmt.Errorf("failed to delete payment destinations: %w", err)
	}

	for _, createParams := range arg.Destinations {
		if _, err := repo.CreatePaymentDestination(ctx, createParams); err != nil {
			return fmt.Errorf("failed to create payment destination: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
