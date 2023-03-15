package events

// Predefined
const (
	PaymentCreated       EventName = "payment.created"
	PaymentProcessing    EventName = "payment.processing"
	PaymentCancelled     EventName = "payment.cancelled"
	PaymentFailed        EventName = "payment.failed"
	PaymentExpired       EventName = "payment.expired"
	PaymentSucceeded     EventName = "payment.succeeded"
	PaymentLinkGenerated EventName = "payment.link.generated"
	TransactionCreated   EventName = "transaction.created"
	TransactionUpdated   EventName = "transaction.updated"
)

// Event payloads.
type (
	PaymentCreatedPayload struct {
		PaymentID string `json:"payment_id"`
	}

	PaymentStatusUpdatedPayload struct {
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
	}

	PaymentLinkGeneratedPayload struct {
		PaymentID string `json:"payment_id"`
		Link      string `json:"link"`
	}

	TransactionCreatedPayload struct {
		PaymentID     string `json:"payment_id"`
		TransactionID string `json:"transaction_id"`
		Reference     string `json:"reference"`
	}

	TransactionUpdatedPayload struct {
		PaymentID string `json:"payment_id"`
		Reference string `json:"reference"`
		Status    string `json:"status"`
		Signature string `json:"signature"`
	}
)
