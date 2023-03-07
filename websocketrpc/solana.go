package websocketrpc

import "encoding/json"

// Predefined event names.
const (
	EventAccountNotification = "accountNotification"
)

// Predefined subscribe/unsubscribe request methods.
const (
	SubscribeAccountRequest   = "accountSubscribe"
	UnsubscribeAccountRequest = "accountUnsubscribe"
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
// It's a helper function to create a request payload for the account subscribe request,
// but you can also create the payload manually.
func AccountSubscribeRequestPayload(base58Addr string) []interface{} {
	return []interface{}{
		base58Addr,
		map[string]interface{}{
			"encoding":   EncodingJSONParsed,
			"commitment": CommitmentFinalized,
		},
	}
}

// AccountUnsubscribeRequestPayload returns an account unsubscribe request payload.
func AccountUnsubscribeRequestPayload(subscriptionID interface{}) []interface{} {
	return []interface{}{subscriptionID}
}

// NotificationPayload represents an notification payload from the websocket server.
// See https://docs.solana.com/api/websocket
type NotificationPayload struct {
	Result struct {
		Context struct {
			Slot int `json:"slot"`
		} `json:"context"`
		Value json.RawMessage `json:"value"`
	} `json:"result"`
	Subscription int `json:"subscription"`
}
