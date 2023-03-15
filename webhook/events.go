package webhook

import (
	"context"

	"github.com/easypmnt/checkout-api/events"
)

type webhookEnqueuer interface {
	FireEvent(ctx context.Context, event string, payload interface{}) error
}

// TranslateEventsToWebhookEvents translates the events from the events package to the webhook events.
func TranslateEventsToWebhookEvents(enq webhookEnqueuer) events.Listener {
	return func(event events.EventName, payload interface{}) error {
		if payload == nil {
			return nil
		}

		return enq.FireEvent(context.Background(), string(event), payload)
	}
}
