package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/easypmnt/checkout-api/repository"
	"github.com/hibiken/asynq"
)

type (
	// Worker is a task handler for email delivery.
	Worker struct {
		svc   service
		event workerEventClient
	}

	service interface {
		CheckPaymentStatus(ctx context.Context, reference string) (string, error)
	}

	workerEventClient interface {
		UnsubscribeByAddress(base58Addr string) error
	}
)

// NewWorker creates a new email task handler.
func NewWorker(svc service, event workerEventClient) *Worker {
	return &Worker{svc: svc, event: event}
}

// Register registers task handlers for email delivery.
func (w *Worker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskCheckPaymentByReference, w.CheckPaymentByReference)
}

// FireEvent sends a webhook event to the specified URL.
func (w *Worker) CheckPaymentByReference(ctx context.Context, t *asynq.Task) error {
	var p ReferencePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	status, err := w.svc.CheckPaymentStatus(ctx, p.Reference)
	if err != nil {
		return fmt.Errorf("failed to fire webhook event: %w", err)
	}

	if status != string(repository.TransactionStatusPending) {
		if err := w.event.UnsubscribeByAddress(p.Reference); err != nil {
			return fmt.Errorf("failed to unsubscribe from account notifications: %w", err)
		}
	}

	return nil
}
