package events

// Predefined
const (
	PaymentCreated                   EventName = "payment.created"
	PaymentProcessing                EventName = "payment.processing"
	PaymentCancelled                 EventName = "payment.cancelled"
	PaymentFailed                    EventName = "payment.failed"
	PaymentExpired                   EventName = "payment.expired"
	PaymentSucceeded                 EventName = "payment.succeeded"
	PaymentLinkGenerated             EventName = "payment.link.generated"
	TransactionCreated               EventName = "transaction.created"
	TransactionUpdated               EventName = "transaction.updated"
	TransactionReferenceNotification EventName = "transaction.reference.notification"
)

var AllEvents = []EventName{
	PaymentCreated,
	PaymentProcessing,
	PaymentCancelled,
	PaymentFailed,
	PaymentExpired,
	PaymentSucceeded,
	PaymentLinkGenerated,
	TransactionCreated,
	TransactionUpdated,
}

// Event payloads.
type (
	// PaymentID is an interface for all events that have payment_id field.
	PaymentIDGetter interface {
		GetPaymentID() string
	}

	// PaymentIDGetter struct for getting payment_id from event payload.
	PaymentID struct {
		PaymentID string `json:"payment_id"`
	}

	PaymentCreatedPayload struct {
		PaymentID
	}

	PaymentStatusUpdatedPayload struct {
		PaymentID
		Status string `json:"status"`
	}

	PaymentLinkGeneratedPayload struct {
		PaymentID
		Link string `json:"link"`
	}

	TransactionCreatedPayload struct {
		PaymentID
		TransactionID string `json:"transaction_id"`
		Reference     string `json:"reference"`
	}

	TransactionUpdatedPayload struct {
		PaymentID
		Reference   string      `json:"reference"`
		Status      string      `json:"status"`
		Signature   string      `json:"signature"`
		Transaction interface{} `json:"transaction,omitempty"`
	}

	ReferencePayload struct {
		Reference string `json:"reference"`
	}
)

// GetPaymentID returns payment_id from event payload.
// This method is required for PaymentIDGetter interface.
func (p PaymentID) GetPaymentID() string {
	return p.PaymentID
}
