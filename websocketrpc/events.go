package websocketrpc

// EventName represents a JSON-RPC event name.
type EventName string

// Predefined event names.
const (
	EventAccountNotification   EventName = "accountNotification"
	EventSignatureNotification EventName = "signatureNotification"
)

// RequestType represents a JSON-RPC request type.
type RequestType string

// Predefined subscribe/unsubscribe request methods.
const (
	SubscribeAccountNotification     RequestType = "accountSubscribe"
	SubscribeSignatureNotification   RequestType = "signatureSubscribe"
	UnsubscribeAccountNotification   RequestType = "accountUnsubscribe"
	UnsubscribeSignatureNotification RequestType = "signatureUnsubscribe"
)

// Predefined encoding types.
const (
	EncodingJSONParsed = "jsonParsed"
	EncodingBase58     = "base58"
	EncodingBase64     = "base64"
	EncodingBase64Zstd = "base64+zstd"
)

// Predefined commitment levels.
const (
	CommitmentFinalized = "finalized"
	CommitmentConfirmed = "confirmed"
	CommitmentProcessed = "processed"
)

// Account subscribe request payload.
func GetAccountSubscribeRequestPayload(base58Addr string, commitment string) []interface{} {
	if commitment == "" {
		commitment = CommitmentFinalized
	}
	return []interface{}{
		base58Addr,
		map[string]interface{}{
			"encoding":   EncodingJSONParsed,
			"commitment": commitment,
		},
	}
}

// GetAccountUnsubscribeRequestPayload returns an account unsubscribe request payload.
func GetAccountUnsubscribeRequestPayload(subscriptionID int) []interface{} {
	return []interface{}{
		subscriptionID,
	}
}

// Signature subscribe request payload.
func GetSignatureSubscribeRequestPayload(signature string, commitment string) []interface{} {
	if commitment == "" {
		commitment = CommitmentFinalized
	}
	return []interface{}{
		signature,
		map[string]interface{}{
			"commitment": CommitmentFinalized,
		},
	}
}

// GetSignatureUnsubscribeRequestPayload returns a signature unsubscribe request payload.
func GetSignatureUnsubscribeRequestPayload(subscriptionID int) []interface{} {
	return []interface{}{
		subscriptionID,
	}
}
