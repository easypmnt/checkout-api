package payments

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// Task names.
const (
	TastMarkPaymentsAsExpired   = "mark_payments_as_expired"
	TaskCheckPaymentByReference = "check_payment_by_reference"
)

// Reference payload to check payment by reference task.
type ReferencePayload struct {
	Reference string `json:"reference"`
}

type (
	// Worker is a task handler for email delivery.
	Worker struct {
		svc paymentService
		sol workerSolanaClient
	}

	paymentService interface {
		MarkPaymentsAsExpired(ctx context.Context) error
		GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error)
		UpdateTransaction(ctx context.Context, reference string, status TransactionStatus, signature string) error
	}

	workerSolanaClient interface {
		ValidateTransactionByReference(ctx context.Context, reference, destination string, amount uint64, mint string) (string, error)
	}
)

// NewWorker creates a new payments task handler.
func NewWorker(svc paymentService, sol workerSolanaClient) *Worker {
	return &Worker{svc: svc}
}

// Register registers task handlers for email delivery.
func (w *Worker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TastMarkPaymentsAsExpired, w.MarkPaymentsAsExpired)
}

// FireEvent sends a webhook event to the specified URL.
func (w *Worker) MarkPaymentsAsExpired(ctx context.Context, t *asynq.Task) error {
	if err := w.svc.MarkPaymentsAsExpired(ctx); err != nil {
		return fmt.Errorf("worker: %w", err)
	}

	return nil
}

// CheckPaymentByReference checks payment status by reference and unsubscribes from account notifications.
func (w *Worker) CheckPaymentByReference(ctx context.Context, t *asynq.Task) error {
	var p ReferencePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	tx, err := w.svc.GetTransactionByReference(ctx, p.Reference)
	if err != nil {
		return fmt.Errorf("failed to get transaction by reference: %w", err)
	}

	if tx.Status != TransactionStatusPending {
		return nil
	}

	txSign, err := w.sol.ValidateTransactionByReference(
		ctx,
		p.Reference,
		tx.DestinationWallet,
		tx.TotalAmount,
		tx.DestinationMint,
	)
	if err != nil {
		return fmt.Errorf("failed to validate transaction by reference: %w", err)
	}

	if err := w.svc.UpdateTransaction(ctx, p.Reference, tx.Status, txSign); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}
