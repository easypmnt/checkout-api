package payments

import "github.com/hibiken/asynq"

// Scheduler is a task scheduler for iam service.
type Scheduler struct{}

// NewScheduler creates a new task scheduler for iam service.
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// Schedule tasks for auth service.
func (s *Scheduler) Schedule(scheduler *asynq.Scheduler) {
	scheduler.Register("@every 5m", asynq.NewTask(TastMarkPaymentsAsExpired, nil))
	scheduler.Register("@every 5m", asynq.NewTask(TaskMarkTransactionsAsExpired, nil))
	scheduler.Register("@every 5m", asynq.NewTask(TaskCheckPendingTransactions, nil))
}
