package webhook

const (
	// ContentTypeJSON is the content type for JSON.
	ContentTypeJSON = "application/json"
	// Default signature header
	DefaultSignatureHeader = "X-Webhook-Signature"
)

// Event types
const (
	EventPaymentCreated   = "payment.created"
	EventPaymentPending   = "payment.pending"
	EventPaymentCompleted = "payment.completed"
	EventPaymentFailed    = "payment.failed"
)

type (
	// Webhook request payload
	WebhookRequestPayload struct {
		Event     string      `json:"event"`                // The name of the event that triggered the webhook
		EventID   string      `json:"event_id,omitempty"`   // The ID of the event that triggered the webhook
		WebhookID string      `json:"webhook_id,omitempty"` // The ID of the webhook that triggered the webhook
		Data      interface{} `json:"data"`                 // The data associated with the event that triggered the webhook
	}

	// Payment data payload
	PaymentData struct {
		PaymentID  string        `json:"payment_id"`      // The ID of the payment
		ExternalID string        `json:"external_id"`     // The ID of the payment in your system. E.g. the order ID, etc.
		Amount     uint64        `json:"amount"`          // The amount of the payment in base units (e.g. lamports, etc.)
		Currency   string        `json:"currency"`        // The currency of the payment: SOL, USDC, or any token mint address.
		Status     string        `json:"status"`          // The status of the payment: new, pending, completed, or failed.
		CreatedAt  string        `json:"created_at"`      // The time the payment was created.
		TxID       string        `json:"tx_id,omitempty"` // The transaction ID of the payment.
		Err        *PaymentError `json:"error,omitempty"` // The error details if the payment failed.
	}

	// Payment error payload
	PaymentError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details"`
	}
)

// Worker task types
const (
	TaskFireEvent = "webhook:fire_event"
)

// FireEventPayload is the payload for the webhook:fire_event task.
type FireEventPayload struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}
