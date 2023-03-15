package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

type (
	// Enqueuer is a helper struct for enqueuing email tasks.
	Enqueuer struct {
		client       *asynq.Client
		queueName    string
		taskDeadline time.Duration
		maxRetry     int
	}

	// EnqueuerOption is a function that configures an enqueuer.
	EnqueuerOption func(*Enqueuer)
)

// NewEnqueuer creates a new email enqueuer.
// This function accepts EnqueuerOption to configure the enqueuer.
// Default values are used if no option is provided.
// Default values are:
//   - queue name: "default"
//   - task deadline: 1 minute
//   - max retry: 3
func NewEnqueuer(client *asynq.Client, opt ...EnqueuerOption) *Enqueuer {
	if client == nil {
		panic("client is nil")
	}

	e := &Enqueuer{
		client:       client,
		queueName:    "default",
		taskDeadline: time.Minute,
		maxRetry:     3,
	}

	for _, o := range opt {
		o(e)
	}

	return e
}

// WithQueueName configures the queue name.
func WithQueueName(name string) EnqueuerOption {
	return func(e *Enqueuer) {
		e.queueName = name
	}
}

// WithTaskDeadline configures the task deadline.
func WithTaskDeadline(d time.Duration) EnqueuerOption {
	return func(e *Enqueuer) {
		e.taskDeadline = d
	}
}

// WithMaxRetry configures the max retry.
func WithMaxRetry(n int) EnqueuerOption {
	return func(e *Enqueuer) {
		e.maxRetry = n
	}
}

// enqueueTask enqueues a task to the queue.
func (e *Enqueuer) enqueueTask(ctx context.Context, task *asynq.Task, opts ...asynq.Option) error {
	if len(opts) == 0 {
		opts = append(opts,
			asynq.Queue(e.queueName),
			asynq.Deadline(time.Now().Add(e.taskDeadline)),
			asynq.MaxRetry(e.maxRetry),
			asynq.Unique(e.taskDeadline),
		)
	}

	if _, err := e.client.Enqueue(task, opts...); err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	return nil
}

// FireEvent enqueues a task to fire an event.
// This function returns an error if the task could not be enqueued.
func (e *Enqueuer) CheckPaymentByReference(ctx context.Context, reference string) error {
	task, err := json.Marshal(ReferencePayload{Reference: reference})
	if err != nil {
		return fmt.Errorf("CheckPaymentByReference: failed to marshal task payload: %w", err)
	}

	if err := e.enqueueTask(ctx, asynq.NewTask(TaskCheckPaymentByReference, task)); err != nil {
		return fmt.Errorf("CheckPaymentByReference: %w", err)
	}

	return nil
}
