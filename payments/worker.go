package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Task names.
const (
	TastMarkPaymentsAsExpired     = "mark_payments_as_expired"
	TaskCheckPaymentByReference   = "check_payment_by_reference"
	TaskMarkTransactionsAsExpired = "mark_transactions_as_expired"
	TaskCheckPendingTransactions  = "check_pending_transactions"
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
		enq paymentEnqueuer
	}

	paymentService interface {
		MarkPaymentsAsExpired(ctx context.Context) error
		GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error)
		UpdateTransaction(ctx context.Context, reference string, status TransactionStatus, signature string) error
		MarkTransactionsAsExpired(ctx context.Context) error
		GetPendingTransactions(ctx context.Context) ([]*Transaction, error)
	}

	workerSolanaClient interface {
		ValidateTransactionByReference(ctx context.Context, reference, destination string, amount uint64, mint string) (string, error)
	}

	paymentEnqueuer interface {
		CheckPaymentByReference(ctx context.Context, reference string) error
	}
)

// NewWorker creates a new payments task handler.
func NewWorker(svc paymentService, sol workerSolanaClient, enq paymentEnqueuer) *Worker {
	return &Worker{svc: svc, sol: sol, enq: enq}
}

// Register registers task handlers for email delivery.
func (w *Worker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TastMarkPaymentsAsExpired, w.MarkPaymentsAsExpired)
	mux.HandleFunc(TaskCheckPaymentByReference, w.CheckPaymentByReference)
	mux.HandleFunc(TaskMarkTransactionsAsExpired, w.MarkTransactionsAsExpired)
	mux.HandleFunc(TaskCheckPendingTransactions, w.CheckPendingTransactions)
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

	ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			tx, err := w.svc.GetTransactionByReference(ctx, p.Reference)
			if err != nil {
				continue
				// return fmt.Errorf("failed to get transaction by reference: %w", err)
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
				continue
				// return fmt.Errorf("failed to validate transaction by reference: %w", err)
			}

			if err := w.svc.UpdateTransaction(ctx, p.Reference, TransactionStatusCompleted, txSign); err != nil {
				continue
				// return fmt.Errorf("failed to update transaction status: %w", err)
			}

			return nil
		}
	}
}

// MarkTransactionsAsExpired marks transactions as expired.
func (w *Worker) MarkTransactionsAsExpired(ctx context.Context, t *asynq.Task) error {
	if err := w.svc.MarkTransactionsAsExpired(ctx); err != nil {
		return fmt.Errorf("worker: %w", err)
	}

	return nil
}

// CheckPendingTransactions checks pending transactions.
func (w *Worker) CheckPendingTransactions(ctx context.Context, t *asynq.Task) error {
	txs, err := w.svc.GetPendingTransactions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending transactions: %w", err)
	}

	for _, tx := range txs {
		if err := w.enq.CheckPaymentByReference(ctx, tx.Reference); err != nil {
			return fmt.Errorf("failed to enqueue check payment by reference task: %w", err)
		}
	}

	return nil
}
