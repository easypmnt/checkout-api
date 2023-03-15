package payments

import (
	"context"
	"fmt"

	"github.com/easypmnt/checkout-api/events"
	"github.com/google/uuid"
)

// getEventName returns the name of the event for the given payment status.
func getEventName(status PaymentStatus) events.EventName {
	switch status {
	case PaymentStatusNew:
		return events.PaymentCreated
	case PaymentStatusPending:
		return events.PaymentProcessing
	case PaymentStatusCompleted:
		return events.PaymentSucceeded
	case PaymentStatusFailed:
		return events.PaymentFailed
	case PaymentStatusCanceled:
		return events.PaymentCancelled
	case PaymentStatusExpired:
		return events.PaymentExpired
	default:
		return ""
	}
}

// UpdateTransactionStatusListener is a listener for the transaction.updated event.
func UpdateTransactionStatusListener(service PaymentService) events.Listener {
	return func(payload ...interface{}) error {
		if len(payload) == 0 {
			return nil
		}

		p, ok := payload[0].(events.TransactionUpdatedPayload)
		if !ok {
			return nil
		}

		if p.Status != "success" {
			return nil
		}

		pid, err := uuid.Parse(p.PaymentID)
		if err != nil {
			return fmt.Errorf("failed to parse payment id: %s", err.Error())
		}

		status := PaymentStatusPending
		switch TransactionStatus(p.Status) {
		case TransactionStatusCompleted:
			status = PaymentStatusCompleted
		case TransactionStatusFailed:
			status = PaymentStatusFailed
		case TransactionStatusPending:
			status = PaymentStatusPending
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		return service.UpdatePaymentStatus(ctx, pid, status)
	}
}
