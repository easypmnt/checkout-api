package server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/easypmnt/checkout-api/internal/validator"
	"github.com/easypmnt/checkout-api/payment"
	"github.com/go-kit/kit/endpoint"
	"github.com/google/uuid"
)

type (
	// Endpoints is a collection of all the endpoints that comprise a server.
	Endpoints struct {
		GetAppInfo                 endpoint.Endpoint
		CreatePayment              endpoint.Endpoint
		CancelPayment              endpoint.Endpoint
		GetPaymentInfo             endpoint.Endpoint
		GetPaymentInfoByExternalID endpoint.Endpoint
		GeneratePaymentLink        endpoint.Endpoint
		GeneratePaymentTransaction endpoint.Endpoint
	}

	Config struct {
		AppName    string // AppName is the name of the application to be displayed in the payment page and wallet.
		AppIconURI string // AppIconURI is the URI of the application icon to be displayed in the payment page and wallet.
	}

	paymentService interface {
		CreatePayment(ctx context.Context, arg payment.CreatePaymentParams) (uuid.UUID, error)
		CancelPayment(ctx context.Context, paymentID uuid.UUID) error
		GetPaymentInfo(ctx context.Context, paymentID uuid.UUID) (*payment.Payment, error)
		GetPaymentInfoByExternalID(ctx context.Context, externalID string) (*payment.Payment, error)
		GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID, currency string, applyBonus bool) (string, error)
		GeneratePaymentTransaction(ctx context.Context, arg payment.GeneratePaymentTransactionParams) (string, error)
	}
)

// MakeEndpoints returns an Endpoints struct where each field is an endpoint
// that comprises the server.
func MakeEndpoints(ps paymentService, cfg Config) Endpoints {
	return Endpoints{
		GetAppInfo:                 makeGetAppInfoEndpoint(cfg),
		CreatePayment:              makeCreatePaymentEndpoint(ps),
		CancelPayment:              makeCancelPaymentEndpoint(ps),
		GetPaymentInfo:             makeGetPaymentInfoEndpoint(ps),
		GetPaymentInfoByExternalID: makeGetPaymentInfoByExternalIDEndpoint(ps),
		GeneratePaymentLink:        makeGeneratePaymentLinkEndpoint(ps),
		GeneratePaymentTransaction: makeGeneratePaymentTransactionEndpoint(ps),
	}
}

// GetAppInfoResponse is the response type for the GetAppInfo method.
type GetAppInfoResponse struct {
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// makeGetAppInfoEndpoint returns an endpoint function for the GetAppInfo method.
func makeGetAppInfoEndpoint(cfg Config) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return GetAppInfoResponse{
			Label: cfg.AppName,
			Icon:  cfg.AppIconURI,
		}, nil
	}
}

// CreatePaymentRequest is the request type for the CreatePayment method.
// For more information about the fields, see the struct definition in payment/payment.go.CreatePaymentParams
type CreatePaymentRequest struct {
	ExternalID   string `json:"external_id,omitempty"`
	Currency     string `json:"currency,omitempty"`
	Amount       int64  `json:"amount,omitempty"`
	Message      string `json:"message,omitempty"`
	Memo         string `json:"memo,omitempty"`
	TTL          int64  `json:"ttl,omitempty"`
	Destinations []struct {
		Amount          int64  `json:"amount,omitempty"`
		Percentage      int16  `json:"percentage,omitempty"`
		WalletAddress   string `json:"wallet_address,omitempty"`
		ApplyBonus      bool   `json:"apply_bonus,omitempty"`
		MaxBonusAmount  int64  `json:"max_bonus_amount,omitempty"`
		MaxBonusPercent int16  `json:"max_bonus_percent,omitempty"`
	} `json:"destinations,omitempty"`
}

// CreatePaymentResponse is the response type for the CreatePayment method.
type CreatePaymentResponse struct {
	PaymentID uuid.UUID `json:"payment_id"`
}

// makeCreatePaymentEndpoint returns an endpoint function for the CreatePayment method.
func makeCreatePaymentEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(CreatePaymentRequest)
		if !ok {
			return nil, ErrInvalidRequest
		}
		if v := validator.ValidateStruct(req); len(v) > 0 {
			return nil, validator.NewValidationError(v)
		}

		paymentID, err := ps.CreatePayment(ctx, payment.CreatePaymentParams{})
		if err != nil {
			return nil, err
		}

		return CreatePaymentResponse{PaymentID: paymentID}, nil
	}
}

// makeCancelPaymentEndpoint returns an endpoint function for the CancelPayment method.
func makeCancelPaymentEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		paymentID, ok := request.(uuid.UUID)
		if !ok {
			return nil, ErrInvalidRequest
		}

		if err := ps.CancelPayment(ctx, paymentID); err != nil {
			return nil, err
		}

		return nil, nil
	}
}

// GetPaymentInfoResponse is the response type for the GetPaymentInfo method.
type GetPaymentInfoResponse struct {
	Payment payment.Payment `json:"payment"`
}

// makeGetPaymentInfoEndpoint returns an endpoint function for the GetPaymentInfo method.
func makeGetPaymentInfoEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		paymentID, ok := request.(uuid.UUID)
		if !ok {
			return nil, ErrInvalidRequest
		}

		payment, err := ps.GetPaymentInfo(ctx, paymentID)
		if err != nil {
			return nil, err
		}

		return GetPaymentInfoResponse{Payment: *payment}, nil
	}
}

// makeGetPaymentInfoByExternalIDEndpoint returns an endpoint function for the GetPaymentInfoByExternalID method.
func makeGetPaymentInfoByExternalIDEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		externalID, ok := request.(string)
		if !ok {
			return nil, ErrInvalidRequest
		}

		payment, err := ps.GetPaymentInfoByExternalID(ctx, externalID)
		if err != nil {
			return nil, err
		}

		return GetPaymentInfoResponse{Payment: *payment}, nil
	}
}

// GeneratePaymentLinkRequest is the request type for the GeneratePaymentLink method.
type GeneratePaymentLinkRequest struct {
	PaymentID  uuid.UUID `json:"-" validate:"-" label:"Payment ID"`
	Currency   string    `json:"currency,omitempty" validate:"-" label:"Currency"`
	ApplyBonus string    `json:"apply_bonus,omitempty" validate:"omitempty|bool" label:"Apply Bonus"`
}

// GeneratePaymentLinkResponse is the response type for the GeneratePaymentLink method.
type GeneratePaymentLinkResponse struct {
	Link string `json:"link"`
}

// makeGeneratePaymentLinkEndpoint returns an endpoint function for the GeneratePaymentLink method.
func makeGeneratePaymentLinkEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GeneratePaymentLinkRequest)
		if !ok {
			return nil, ErrInvalidRequest
		}
		if v := validator.ValidateStruct(req); len(v) > 0 {
			return nil, validator.NewValidationError(v)
		}

		applyBonus, _ := strconv.ParseBool(req.ApplyBonus)
		link, err := ps.GeneratePaymentLink(ctx, req.PaymentID, req.Currency, applyBonus)
		if err != nil {
			return nil, err
		}

		return GeneratePaymentLinkResponse{Link: link}, nil
	}
}

// GeneratePaymentTransactionRequest is the request type for the GeneratePaymentTransaction method.
type GeneratePaymentTransactionRequest struct {
	PaymentID  string `json:"-" validate:"required|uuid" label:"Payment ID"`
	Base58Addr string `json:"account" validate:"required" label:"Account public key"`
	Currency   string `json:"-" validate:"-"`
	ApplyBonus string `json:"-" validate:"omitempty|bool"`
}

// GeneratePaymentTransactionResponse is the response type for the GeneratePaymentTransaction method.
type GeneratePaymentTransactionResponse struct {
	Transaction string `json:"transaction"`
}

// makeGeneratePaymentTransactionEndpoint returns an endpoint function for the GeneratePaymentTransaction method.
func makeGeneratePaymentTransactionEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GeneratePaymentTransactionRequest)
		if !ok {
			return nil, ErrInvalidRequest
		}
		if v := validator.ValidateStruct(req); len(v) > 0 {
			return nil, validator.NewValidationError(v)
		}

		paymentID, err := uuid.Parse(req.PaymentID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid payment ID: %v", ErrInvalidParameter, err)
		}

		applyBonus, _ := strconv.ParseBool(req.ApplyBonus)

		base64Tx, err := ps.GeneratePaymentTransaction(ctx, payment.GeneratePaymentTransactionParams{
			PaymentID:  paymentID,
			Base58Addr: req.Base58Addr,
			Currency:   req.Currency,
			ApplyBonus: applyBonus,
		})
		if err != nil {
			return nil, err
		}

		return GeneratePaymentTransactionResponse{Transaction: base64Tx}, nil
	}
}
