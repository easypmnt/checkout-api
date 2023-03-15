package sse

import (
	"errors"
	"fmt"

	"github.com/easypmnt/checkout-api/events"
)

type sseService interface {
	PubEvent(channelID string, data EventData, ttl int64) error
}

// TranslateEventsToSSEChannel translates the events from the events package to the webhook events.
func TranslateEventsToSSEChannel(sse sseService) events.Listener {
	return func(event events.EventName, payload interface{}) error {
		if payload == nil {
			return nil
		}

		pid, err := getPaymentIDFromPayload(payload)
		if err != nil {
			return fmt.Errorf("translate events to webhook events: %w", err)
		}

		return sse.PubEvent(pid, EventData{
			Name:    string(event),
			Payload: payload,
		}, 300)
	}
}

// getPaymentIDFromPayload returns the payment ID from the payload.
func getPaymentIDFromPayload(payload interface{}) (string, error) {
	v, ok := payload.(events.PaymentIDGetter)
	if !ok {
		return "", errors.New("payload does not implement PaymentIDGetter")
	}

	return v.GetPaymentID(), nil
}
