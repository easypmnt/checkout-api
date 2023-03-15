package payments

import (
	"context"
	"fmt"

	"github.com/easypmnt/checkout-api/events"
	"github.com/google/uuid"
)

type (
	ServiceEvents struct {
		svc       PaymentService
		fireEvent fireEventFunc
	}

	fireEventFunc func(events.EventName, ...interface{})
)

func NewServiceEvents(svc PaymentService, eventFn fireEventFunc) *ServiceEvents {
	return &ServiceEvents{svc: svc, fireEvent: eventFn}
}

// CreatePayment creates a new payment.
func (s *ServiceEvents) CreatePayment(ctx context.Context, payment *Payment) (*Payment, error) {
	result, err := s.svc.CreatePayment(ctx, payment)
	if err != nil {
		return nil, err
	}

	s.fireEvent(events.PaymentCreated, events.PaymentCreatedPayload{
		PaymentID: result.ID.String(),
	})

	return result, nil
}

// GetPayment returns the payment with the given ID.
func (s *ServiceEvents) GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error) {
	return s.svc.GetPayment(ctx, id)
}

// GetPaymentByExternalID returns the payment with the given external ID.
func (s *ServiceEvents) GetPaymentByExternalID(ctx context.Context, externalID string) (*Payment, error) {
	return s.svc.GetPaymentByExternalID(ctx, externalID)
}

// GeneratePaymentLink generates a new payment link for the given payment.
func (s *ServiceEvents) GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID, mint string, applyBonus bool) (string, error) {
	result, err := s.svc.GeneratePaymentLink(ctx, paymentID, mint, applyBonus)
	if err != nil {
		return "", err
	}

	s.fireEvent(events.PaymentLinkGenerated, events.PaymentLinkGeneratedPayload{
		PaymentID: paymentID.String(),
		Link:      result,
	})

	return result, nil
}

// CancelPayment cancels the payment with the given ID.
func (s *ServiceEvents) CancelPayment(ctx context.Context, id uuid.UUID) error {
	if err := s.svc.CancelPayment(ctx, id); err != nil {
		return err
	}

	s.fireEvent(events.PaymentCancelled, events.PaymentStatusUpdatedPayload{
		PaymentID: id.String(),
		Status:    string(PaymentStatusCanceled),
	})

	return nil
}

// CancelPaymentByExternalID cancels the payment with the given external ID.
func (s *ServiceEvents) CancelPaymentByExternalID(ctx context.Context, externalID string) error {
	payment, err := s.svc.GetPaymentByExternalID(ctx, externalID)
	if err != nil {
		return err
	}

	if err := s.svc.CancelPaymentByExternalID(ctx, externalID); err != nil {
		return err
	}

	s.fireEvent(events.PaymentCancelled, events.PaymentStatusUpdatedPayload{
		PaymentID: payment.ID.String(),
		Status:    string(PaymentStatusCanceled),
	})

	return nil
}

// UpdatePaymentStatus updates the status of the payment with the given ID.
func (s *ServiceEvents) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status PaymentStatus) error {
	prev, err := s.GetPayment(ctx, id)
	if err != nil {
		return err
	}

	if err := s.svc.UpdatePaymentStatus(ctx, id, status); err != nil {
		return err
	}

	if prev.Status != status {
		eventName := getEventName(status)
		if eventName == "" {
			return fmt.Errorf("unknown payment status %s", status)
		}
		s.fireEvent(eventName, events.PaymentStatusUpdatedPayload{
			PaymentID: id.String(),
			Status:    string(status),
		})
	}

	return nil
}

// BuildTransaction builds a new transaction for the given payment.
func (s *ServiceEvents) BuildTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	result, err := s.svc.BuildTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	s.fireEvent(events.TransactionCreated, events.TransactionCreatedPayload{
		TransactionID: result.ID.String(),
		PaymentID:     result.PaymentID.String(),
		Reference:     result.Reference,
	})

	return result, nil
}

// GetTransactionByReference returns the transaction with the given reference.
func (s *ServiceEvents) GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error) {
	return s.svc.GetTransactionByReference(ctx, reference)
}

// MarkPaymentsAsExpired marks all payments that are expired as expired.
func (s *ServiceEvents) MarkPaymentsAsExpired(ctx context.Context) error {
	return s.svc.MarkPaymentsAsExpired(ctx)
}

// UpdateTransaction updates the status and signature of the transaction with the given reference.
func (s *ServiceEvents) UpdateTransaction(ctx context.Context, reference string, status TransactionStatus, signature string) error {
	if err := s.svc.UpdateTransaction(ctx, reference, status, signature); err != nil {
		return err
	}

	tx, err := s.GetTransactionByReference(ctx, reference)
	if err != nil {
		return err
	}

	s.fireEvent(events.TransactionUpdated, events.TransactionUpdatedPayload{
		PaymentID: tx.PaymentID.String(),
		Reference: tx.Reference,
		Status:    string(tx.Status),
		Signature: tx.Signature,
	})

	return nil
}
