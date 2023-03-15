package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

type (
	// Worker is a task handler for email delivery.
	Worker struct {
		svc service
	}

	service interface {
		FireEvent(event string, payload interface{}) error
	}
)

// NewWorker creates a new email task handler.
func NewWorker(svc service) *Worker {
	return &Worker{svc: svc}
}

// Register registers task handlers for email delivery.
func (w *Worker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskFireEvent, w.FireEvent)
}

// FireEvent sends a webhook event to the specified URL.
func (w *Worker) FireEvent(ctx context.Context, t *asynq.Task) error {
	var p FireEventPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if err := w.svc.FireEvent(p.Event, p.Payload); err != nil {
		return fmt.Errorf("failed to fire webhook event: %w", err)
	}

	return nil
}
