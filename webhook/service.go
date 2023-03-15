package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type (
	// Service is the webhook service implementation.
	Service struct {
		client          *http.Client
		signatureHeader string
		signatureSecret []byte
		webhookURI      string
	}

	// ServiceOption is a function that configures the webhook service.
	ServiceOption func(*Service)
)

// NewService creates a new webhook service.
func NewService(opts ...ServiceOption) *Service {
	s := &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		signatureHeader: DefaultSignatureHeader,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithHTTPClient configures the webhook service with a custom HTTP client.
func WithHTTPClient(client *http.Client) ServiceOption {
	return func(s *Service) {
		s.client = client
	}
}

// WithSignatureHeader configures the webhook service with a custom signature header.
func WithSignatureHeader(header string) ServiceOption {
	return func(s *Service) {
		s.signatureHeader = strings.TrimSpace(header)
	}
}

// WithSignatureSecret configures the webhook service with a custom signature secret.
func WithSignatureSecret(secret []byte) ServiceOption {
	return func(s *Service) {
		s.signatureSecret = secret
	}
}

// WithWebhookURI configures the webhook service with a custom webhook URI.
func WithWebhookURI(uri string) ServiceOption {
	return func(s *Service) {
		s.webhookURI = uri
	}
}

// Send post request to webhook url with payload.
func (s *Service) Send(url string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	signature, err := SignPayload(body, s.signatureSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", ContentTypeJSON)
	req.Header.Set(s.signatureHeader, signature)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make POST request: %w", err)
	}

	return resp, nil
}

// FireEvent sends a webhook event to the webhook url.
func (s *Service) FireEvent(event string, payload interface{}) error {
	if s.webhookURI == "" {
		return fmt.Errorf("webhook uri is not set")
	}

	return s.fireEvent(event, s.webhookURI, payload)
}

// fireEvent sends a webhook event to the webhook url.
func (s *Service) fireEvent(event, url string, payload interface{}) error {
	reqData := WebhookRequestPayload{
		Event: event,
		Data:  payload,
	}
	resp, err := s.Send(url, reqData)
	if err != nil {
		return fmt.Errorf("failed to send webhook event: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send webhook event: %s", resp.Status)
	}

	return nil
}
