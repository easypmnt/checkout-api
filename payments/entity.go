package payments

import (
	"time"

	"github.com/easypmnt/checkout-api/repository"
	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment.
type PaymentStatus string

// Predefined payment statuses.
const (
	PaymentStatusNew       PaymentStatus = "new"
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusExpired   PaymentStatus = "expired"
)

// TransactionStatus represents the status of a transaction.
type TransactionStatus string

// Predefined transaction statuses.
const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

// Payment represents an initial payment request.
type Payment struct {
	ID                uuid.UUID     `json:"id,omitempty"`
	ExternalID        string        `json:"external_id,omitempty"`
	DestinationWallet string        `json:"destination_wallet,omitempty"`
	DestinationMint   string        `json:"destination_mint,omitempty"`
	Amount            uint64        `json:"amount,omitempty"`
	Status            PaymentStatus `json:"status,omitempty"`
	Message           string        `json:"message,omitempty"`
	ExpiresAt         *time.Time    `json:"expires_at,omitempty"`
}

type Transaction struct {
	ID                 uuid.UUID         `json:"id,omitempty"`
	PaymentID          uuid.UUID         `json:"payment_id,omitempty"`
	Reference          string            `json:"reference,omitempty"`
	SourceWallet       string            `json:"source_wallet,omitempty"`
	SourceMint         string            `json:"source_mint,omitempty"`
	DestinationWallet  string            `json:"destination_wallet,omitempty"`
	DestinationMint    string            `json:"destination_mint,omitempty"`
	Amount             uint64            `json:"amount,omitempty"`
	DiscountAmount     uint64            `json:"discount_amount,omitempty"`
	TotalAmount        uint64            `json:"total_amount,omitempty"`
	AccruedBonusAmount uint64            `json:"accrued_bonus_amount,omitempty"`
	Message            string            `json:"message,omitempty"`
	Memo               string            `json:"memo,omitempty"`
	ApplyBonus         bool              `json:"apply_bonus,omitempty"`
	Transaction        string            `json:"transaction,omitempty"`
	Status             TransactionStatus `json:"status,omitempty"`
	Signature          string            `json:"signature,omitempty"`
}

// cast repository.Payment to payments.Payment
func castFromRepositoryPayment(p repository.Payment) *Payment {
	result := &Payment{
		ID:                p.ID,
		ExternalID:        p.ExternalID.String,
		DestinationWallet: p.DestinationWallet,
		DestinationMint:   p.DestinationMint,
		Amount:            uint64(p.Amount),
		Status:            castFromRepositoryPaymentStatus(p.Status),
		Message:           p.Message.String,
	}

	if p.ExpiresAt.Valid {
		result.ExpiresAt = &p.ExpiresAt.Time
	}

	return result
}

// cast repository payment status to payments.PaymentStatus
func castFromRepositoryPaymentStatus(status repository.PaymentStatus) PaymentStatus {
	switch status {
	case repository.PaymentStatusNew:
		return PaymentStatusNew
	case repository.PaymentStatusPending:
		return PaymentStatusPending
	case repository.PaymentStatusCompleted:
		return PaymentStatusCompleted
	case repository.PaymentStatusFailed:
		return PaymentStatusFailed
	case repository.PaymentStatusCanceled:
		return PaymentStatusCanceled
	case repository.PaymentStatusExpired:
		return PaymentStatusExpired
	default:
		return PaymentStatusNew
	}
}

// castToRepositoryPaymentStatus casts payments.PaymentStatus to repository.PaymentStatus
func castToRepositoryPaymentStatus(status PaymentStatus) repository.PaymentStatus {
	switch status {
	case PaymentStatusNew:
		return repository.PaymentStatusNew
	case PaymentStatusPending:
		return repository.PaymentStatusPending
	case PaymentStatusCompleted:
		return repository.PaymentStatusCompleted
	case PaymentStatusFailed:
		return repository.PaymentStatusFailed
	case PaymentStatusCanceled:
		return repository.PaymentStatusCanceled
	case PaymentStatusExpired:
		return repository.PaymentStatusExpired
	}

	return repository.PaymentStatusNew
}

// cast repository.Transaction to payments.Transaction
func castFromRepositoryTransaction(t repository.Transaction, conf Config) *Transaction {
	result := &Transaction{
		ID:                 t.ID,
		PaymentID:          t.PaymentID,
		Reference:          t.Reference,
		SourceWallet:       t.SourceWallet,
		SourceMint:         t.SourceMint,
		DestinationWallet:  t.DestinationWallet,
		DestinationMint:    t.DestinationMint,
		Amount:             uint64(t.Amount),
		DiscountAmount:     uint64(t.DiscountAmount),
		TotalAmount:        uint64(t.TotalAmount),
		AccruedBonusAmount: uint64(t.AccruedBonusAmount),
		Message:            t.Message.String,
		Memo:               t.Memo.String,
		Status:             castFromRepositoryTransactionStatus(t.Status),
		Signature:          t.TxSignature.String,
	}

	if t.ApplyBonus.Valid {
		result.ApplyBonus = t.ApplyBonus.Bool
	} else {
		result.ApplyBonus = conf.ApplyBonus
	}

	if t.DestinationWallet == "" {
		result.DestinationWallet = conf.DestinationWallet
	}
	if t.DestinationMint == "" {
		result.DestinationMint = conf.DestinationMint
	}

	if t.TotalAmount == 0 && result.Amount > 0 {
		result.TotalAmount = result.Amount - result.DiscountAmount
	} else if result.Amount == 0 && result.TotalAmount > 0 {
		result.Amount = result.TotalAmount + result.DiscountAmount
	} else if result.Amount == 0 && result.TotalAmount == 0 {
		result.Amount = result.DiscountAmount
	}

	return result
}

func castToRepositoryTransactionStatus(status TransactionStatus) repository.TransactionStatus {
	switch status {
	case TransactionStatusPending:
		return repository.TransactionStatusPending
	case TransactionStatusCompleted:
		return repository.TransactionStatusCompleted
	case TransactionStatusFailed:
		return repository.TransactionStatusFailed
	}

	return repository.TransactionStatusPending
}

// cast from repository.TransactionStatus to payments.TransactionStatus
func castFromRepositoryTransactionStatus(status repository.TransactionStatus) TransactionStatus {
	switch status {
	case repository.TransactionStatusPending:
		return TransactionStatusPending
	case repository.TransactionStatusCompleted:
		return TransactionStatusCompleted
	case repository.TransactionStatusFailed:
		return TransactionStatusFailed
	}

	return TransactionStatusPending
}
