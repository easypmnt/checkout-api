package server

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/easypmnt/checkout-api/internal/validator"
	"github.com/easypmnt/checkout-api/jupiter"
	"github.com/easypmnt/checkout-api/payments"
	"github.com/go-kit/kit/endpoint"
	"github.com/google/uuid"
)

type (
	// Endpoints is a collection of all the endpoints that comprise a server.
	Endpoints struct {
		GetAppInfo                 endpoint.Endpoint
		CreatePayment              endpoint.Endpoint
		CancelPayment              endpoint.Endpoint
		GetPayment                 endpoint.Endpoint
		GetPaymentByExternalID     endpoint.Endpoint
		GeneratePaymentLink        endpoint.Endpoint
		GeneratePaymentTransaction endpoint.Endpoint
		GetExchangeRate            endpoint.Endpoint
	}

	Config struct {
		AppName    string // AppName is the name of the application to be displayed in the payment page and wallet.
		AppIconURI string // AppIconURI is the URI of the application icon to be displayed in the payment page and wallet.
	}

	paymentService interface {
		// CreatePayment creates a new payment.
		CreatePayment(ctx context.Context, payment *payments.Payment) (*payments.Payment, error)
		// GetPayment returns the payment with the given ID.
		GetPayment(ctx context.Context, id uuid.UUID) (*payments.Payment, error)
		// GetPaymentByExternalID returns the payment with the given external ID.
		GetPaymentByExternalID(ctx context.Context, externalID string) (*payments.Payment, error)
		// GeneratePaymentLink generates a new payment link for the given payment.
		GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID, mint string, applyBonus bool) (string, error)
		// CancelPayment cancels the payment with the given ID.
		CancelPayment(ctx context.Context, id uuid.UUID) error
		// CancelPaymentByExternalID cancels the payment with the given external ID.
		CancelPaymentByExternalID(ctx context.Context, externalID string) error
		// BuildTransaction builds a new transaction for the given payment.
		BuildTransaction(ctx context.Context, tx *payments.Transaction) (*payments.Transaction, error)
		// GetTransactionByReference returns the transaction with the given reference.
		GetTransactionByReference(ctx context.Context, reference string) (*payments.Transaction, error)
	}

	jupiterClient interface {
		ExchangeRate(params jupiter.ExchangeRateParams) (jupiter.Rate, error)
	}
)

// MakeEndpoints returns an Endpoints struct where each field is an endpoint
// that comprises the server.
func MakeEndpoints(ps paymentService, jup jupiterClient, cfg Config) Endpoints {
	return Endpoints{
		GetAppInfo:                 makeGetAppInfoEndpoint(cfg),
		CreatePayment:              makeCreatePaymentEndpoint(ps),
		CancelPayment:              makeCancelPaymentEndpoint(ps),
		GetPayment:                 makeGetPaymentEndpoint(ps),
		GetPaymentByExternalID:     makeGetPaymentByExternalIDEndpoint(ps),
		GeneratePaymentLink:        makeGeneratePaymentLinkEndpoint(ps),
		GeneratePaymentTransaction: makeGeneratePaymentTransactionEndpoint(ps),
		GetExchangeRate:            makeGetExchangeRateEndpoint(jup),
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
	ExternalID string `json:"external_id,omitempty" validate:"min_len:1|max_len:50"`
	Amount     uint64 `json:"amount,omitempty" validate:"required|gt:0"`
	Message    string `json:"message,omitempty" validate:"min_len:2|max_len:100"`
	TTL        int64  `json:"ttl,omitempty" validate:"min:0|max:86400"`
}

// CreatePaymentResponse is the response type for the CreatePayment method.
type CreatePaymentResponse struct {
	Payment *payments.Payment `json:"payment"`
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

		payment := &payments.Payment{
			ExternalID: req.ExternalID,
			Amount:     req.Amount,
			Message:    req.Message,
		}
		if req.TTL > 0 {
			payment.ExpiresAt = utils.Pointer(time.Now().Add(time.Duration(req.TTL) * time.Second))
		}
		payment, err := ps.CreatePayment(ctx, payment)
		if err != nil {
			return nil, err
		}

		return CreatePaymentResponse{Payment: payment}, nil
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

// GetPaymentResponse is the response type for the GetPayment method.
type GetPaymentResponse struct {
	Payment     *payments.Payment     `json:"payment"`
	Transaction *payments.Transaction `json:"transaction,omitempty"`
}

// makeGetPaymentEndpoint returns an endpoint function for the GetPayment method.
func makeGetPaymentEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		paymentID, ok := request.(uuid.UUID)
		if !ok {
			return nil, ErrInvalidRequest
		}

		payment, err := ps.GetPayment(ctx, paymentID)
		if err != nil {
			return nil, err
		}

		return GetPaymentResponse{Payment: payment}, nil
	}
}

// makeGetPaymentByExternalIDEndpoint returns an endpoint function for the GetPaymentByExternalID method.
func makeGetPaymentByExternalIDEndpoint(ps paymentService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		externalID, ok := request.(string)
		if !ok {
			return nil, ErrInvalidRequest
		}

		payment, err := ps.GetPaymentByExternalID(ctx, externalID)
		if err != nil {
			return nil, err
		}

		return GetPaymentResponse{Payment: payment}, nil
	}
}

// GeneratePaymentLinkRequest is the request type for the GeneratePaymentLink method.
type GeneratePaymentLinkRequest struct {
	PaymentID  uuid.UUID `json:"-" validate:"-" label:"Payment ID"`
	Mint       string    `json:"mint,omitempty" validate:"-" label:"Selected Mint"`
	ApplyBonus bool      `json:"apply_bonus,omitempty" validate:"bool" label:"Apply Bonus"`
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

		link, err := ps.GeneratePaymentLink(ctx, req.PaymentID, req.Mint, req.ApplyBonus)
		if err != nil {
			return nil, err
		}

		return GeneratePaymentLinkResponse{Link: link}, nil
	}
}

// GeneratePaymentTransactionRequest is the request type for the GeneratePaymentTransaction method.
type GeneratePaymentTransactionRequest struct {
	PaymentID    string `json:"-" validate:"required|uuid" label:"Payment ID"`
	SourceWallet string `json:"account" validate:"required" label:"Account public key"`
	Mint         string `json:"-" validate:"-"`
	ApplyBonus   string `json:"-" validate:"bool"`
}

// GeneratePaymentTransactionResponse is the response type for the GeneratePaymentTransaction method.
type GeneratePaymentTransactionResponse struct {
	Transaction string `json:"transaction"`
	Message     string `json:"message,omitempty"`
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
		tx := &payments.Transaction{
			PaymentID:    paymentID,
			SourceWallet: req.SourceWallet,
			SourceMint:   req.Mint,
			ApplyBonus:   applyBonus,
		}

		result, err := ps.BuildTransaction(ctx, tx)
		if err != nil {
			return nil, err
		}

		return GeneratePaymentTransactionResponse{
			Transaction: result.Transaction,
			Message:     result.Message,
		}, nil
	}
}

// GetExchangeRateRequest is the request type for the GetExchangeRate method.
type GetExchangeRateRequest struct {
	InCurrency  string `json:"in_currency" validate:"required" label:"In Currency"`
	OutCurrency string `json:"out_currency" validate:"required" label:"Out Currency"`
	Amount      uint64 `json:"amount" validate:"required|gte:0" label:"Amount"`
	SwapMode    string `json:"swap_mode" validate:"required|in:ExactIn,ExactOut" label:"Swap Mode"`
}

// GetExchangeRateResponse is the response type for the GetExchangeRate method.
type GetExchangeRateResponse struct {
	ExchangeRate jupiter.Rate `json:"exchange_rate"`
}

// makeGetExchangeRateEndpoint returns an endpoint function for the GetExchangeRate method.
func makeGetExchangeRateEndpoint(jup jupiterClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		currency, ok := request.(GetExchangeRateRequest)
		if !ok {
			return nil, ErrInvalidRequest
		}
		if v := validator.ValidateStruct(currency); len(v) > 0 {
			return nil, validator.NewValidationError(v)
		}

		rate, err := jup.ExchangeRate(jupiter.ExchangeRateParams{
			InputMint:  currency.InCurrency,
			OutputMint: currency.OutCurrency,
			Amount:     currency.Amount,
			SwapMode:   currency.SwapMode,
		})
		if err != nil {
			return nil, err
		}

		return GetExchangeRateResponse{
			ExchangeRate: rate,
		}, nil
	}
}
