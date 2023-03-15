package payments

import (
	"context"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/google/uuid"
)

type ServiceLogger struct {
	PaymentService
	log Logger
}

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func NewServiceLogger(svc PaymentService, log Logger) *ServiceLogger {
	return &ServiceLogger{
		PaymentService: svc,
		log:            log,
	}
}

// CreatePayment creates a new payment.
func (s *ServiceLogger) CreatePayment(ctx context.Context, payment *Payment) (*Payment, error) {
	s.log.Debugf("creating payment: %s", utils.AnyToString(payment))

	result, err := s.PaymentService.CreatePayment(ctx, payment)
	if err != nil {
		s.log.Errorf("failed to create payment: %s", err.Error())
		return nil, err
	}

	s.log.Infof("payment created: %s", result.ID.String())

	return result, nil
}

// GetPayment returns the payment with the given ID.
func (s *ServiceLogger) GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error) {
	s.log.Debugf("getting payment: %s", id.String())

	result, err := s.PaymentService.GetPayment(ctx, id)
	if err != nil {
		s.log.Errorf("failed to get payment: %s", err.Error())
		return nil, err
	}

	return result, nil
}

// GetPaymentByExternalID returns the payment with the given external ID.
func (s *ServiceLogger) GetPaymentByExternalID(ctx context.Context, externalID string) (*Payment, error) {
	s.log.Debugf("getting payment by external id: %s", externalID)

	result, err := s.PaymentService.GetPaymentByExternalID(ctx, externalID)
	if err != nil {
		s.log.Errorf("failed to get payment by external id %s: %s", externalID, err.Error())
		return nil, err
	}

	return result, nil
}

// GeneratePaymentLink generates a new payment link for the given payment.
func (s *ServiceLogger) GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID, mint string, applyBonus bool) (string, error) {
	s.log.Debugf("generating payment link: id=%s, mint=%s, apply_bonus=%t", paymentID.String(), mint, applyBonus)

	result, err := s.PaymentService.GeneratePaymentLink(ctx, paymentID, mint, applyBonus)
	if err != nil {
		s.log.Errorf("failed to generate payment link: %s", err.Error())
		return "", err
	}

	s.log.Debugf("payment link generated: %s", result)

	return result, nil
}

// UpdatePaymentStatus updates the status of the payment with the given ID.
func (s *ServiceLogger) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status PaymentStatus) error {
	s.log.Debugf("updating payment status: id=%s, status=%s", id.String(), status)

	if err := s.PaymentService.UpdatePaymentStatus(ctx, id, status); err != nil {
		s.log.Errorf("failed to update payment status: %s", err.Error())
		return err
	}

	s.log.Infof("payment status updated: id=%s, status=%s", id.String(), status)

	return nil
}

// CancelPayment cancels the payment with the given ID.
func (s *ServiceLogger) CancelPayment(ctx context.Context, id uuid.UUID) error {
	s.log.Debugf("cancelling payment by id=%s", id.String())

	if err := s.PaymentService.CancelPayment(ctx, id); err != nil {
		s.log.Errorf("failed to cancel payment with id=%s: %s", id.String(), err.Error())
		return err
	}

	s.log.Infof("payment cancelled: id=%s", id.String())

	return nil
}

// CancelPaymentByExternalID cancels the payment with the given external ID.
func (s *ServiceLogger) CancelPaymentByExternalID(ctx context.Context, externalID string) error {
	s.log.Debugf("cancelling payment by external_id=%s", externalID)

	if err := s.PaymentService.CancelPaymentByExternalID(ctx, externalID); err != nil {
		s.log.Errorf("failed to cancel payment by external_id=%s: %s", externalID, err.Error())
		return err
	}

	s.log.Infof("payment cancelled: external_id=%s", externalID)

	return nil
}

// BuildTransaction builds a new transaction for the given payment.
func (s *ServiceLogger) BuildTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	s.log.Debugf("building transaction: %s", utils.AnyToString(tx))

	result, err := s.PaymentService.BuildTransaction(ctx, tx)
	if err != nil {
		s.log.Errorf("failed to build transaction: %s", err.Error())
		return nil, err
	}

	s.log.Infof("transaction built: %s", result.ID.String())

	return result, nil
}

// GetTransactionByReference returns the transaction with the given reference.
func (s *ServiceLogger) GetTransactionByReference(ctx context.Context, reference string) (*Transaction, error) {
	s.log.Debugf("getting transaction by reference: %s", reference)

	result, err := s.PaymentService.GetTransactionByReference(ctx, reference)
	if err != nil {
		s.log.Errorf("failed to get transaction by reference %s: %s", reference, err.Error())
		return nil, err
	}

	return result, nil
}

// MarkPaymentsAsExpired marks all payments that are expired as expired.
func (s *ServiceLogger) MarkPaymentsAsExpired(ctx context.Context) error {
	s.log.Debugf("marking payments as expired")

	if err := s.PaymentService.MarkPaymentsAsExpired(ctx); err != nil {
		s.log.Errorf("failed to mark payments as expired: %s", err.Error())
		return err
	}

	s.log.Infof("payments marked as expired")

	return nil
}

// UpdateTransaction updates the status and signature of the transaction with the given reference.
func (s *ServiceLogger) UpdateTransaction(ctx context.Context, reference string, status TransactionStatus, signature string) error {
	s.log.Debugf("updating transaction: reference=%s, status=%s, signature=%s", reference, status, signature)

	if err := s.PaymentService.UpdateTransaction(ctx, reference, status, signature); err != nil {
		s.log.Errorf("failed to update transaction: %s", err.Error())
		return err
	}

	s.log.Infof("transaction updated: reference=%s, status=%s, signature=%s", reference, status, signature)

	return nil
}

// GetPendingTransactions returns all pending transactions.
func (s *ServiceLogger) GetPendingTransactions(ctx context.Context) ([]*Transaction, error) {
	s.log.Debugf("getting pending transactions")

	result, err := s.PaymentService.GetPendingTransactions(ctx)
	if err != nil {
		s.log.Errorf("failed to get pending transactions: %s", err.Error())
		return nil, err
	}

	return result, nil
}

// MarkTransactionsAsExpired marks all transactions that are expired as expired.
func (s *ServiceLogger) MarkTransactionsAsExpired(ctx context.Context) error {
	s.log.Debugf("marking transactions as expired")

	if err := s.PaymentService.MarkTransactionsAsExpired(ctx); err != nil {
		s.log.Errorf("failed to mark transactions as expired: %s", err.Error())
		return err
	}

	s.log.Infof("transactions marked as expired")

	return nil
}
