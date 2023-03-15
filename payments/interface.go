package payments

import (
	"context"

	"github.com/google/uuid"
)

// PaymentService is the interface that wraps the basic payment operations.
type PaymentService interface {
	// CreatePayment creates a new payment.
	CreatePayment(ctx context.Context, payment *Payment) (*Payment, error)
	// GetPayment returns the payment with the given ID.
	GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error)
	// GetPaymentByExternalID returns the payment with the given external ID.
	GetPaymentByExternalID(ctx context.Context, externalID string) (*Payment, error)
	// GeneratePaymentLink generates a new payment link for the given payment.
	GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID, mint string, applyBonus bool) (string, error)
	// UpdatePaymentStatus updates the status of the payment with the given ID.
	UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status PaymentStatus) error
	// CancelPayment cancels the payment with the given ID.
	CancelPayment(ctx context.Context, id uuid.UUID) error
	// CancelPaymentByExternalID cancels the payment with the given external ID.
	CancelPaymentByExternalID(ctx context.Context, externalID string) error
	// MarkPaymentsAsExpired marks all payments that are expired as expired.
	MarkPaymentsAsExpired(ctx context.Context) error
	// BuildTransaction builds a new transaction for the given payment.
	BuildTransaction(ctx context.Context, tx *Transaction) (*Transaction, error)
	// GetTransactionByReference returns the transaction with the given reference.
	GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error)
	// UpdateTransaction updates the status and signature of the transaction with the given reference.
	UpdateTransaction(ctx context.Context, reference string, status TransactionStatus, signature string) error
	// GetPendingTransactions returns all pending transactions.
	GetPendingTransactions(ctx context.Context) ([]*Transaction, error)
	// MarkTransactionsAsExpired marks all transactions that are expired as expired.
	MarkTransactionsAsExpired(ctx context.Context) error
}
