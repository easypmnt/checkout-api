package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/easypmnt/checkout-api/internal/httpencoder"
	"github.com/easypmnt/checkout-api/internal/validator"
	"github.com/go-chi/chi/v5"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/google/uuid"
)

type (
	logger interface {
		Log(keyvals ...interface{}) error
	}

	middlewareFunc func(http.Handler) http.Handler
)

// MakeHTTPHandler returns an http.Handler that can be used to serve the API.
func MakeHTTPHandler(e Endpoints, log logger, authMdw middlewareFunc) http.Handler {
	r := chi.NewRouter()

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(log)),
		httptransport.ServerErrorEncoder(httpencoder.EncodeError(log, codeAndMessageFrom)),
	}

	// Without auth
	r.Group(func(r chi.Router) {
		r.Get("/checkout/{payment_id}", httptransport.NewServer(
			e.GetAppInfo,
			decodeGetAppInfoRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)

		r.Post("/checkout/{payment_id}", httptransport.NewServer(
			e.GeneratePaymentTransaction,
			decodeGeneratePaymentTransactionRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)
	})

	// With auth
	r.Group(func(r chi.Router) {
		r.Use(authMdw)

		r.Post("/", httptransport.NewServer(
			e.CreatePayment,
			decodeCreatePaymentRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)

		r.Get("/pid/{payment_id}", httptransport.NewServer(
			e.GetPaymentInfo,
			decodeGetPaymentInfoRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)

		r.Get("/ext/{external_id}", httptransport.NewServer(
			e.GetPaymentInfoByExternalID,
			decodeGetPaymentInfoByExternalIDRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)

		r.Post("/pid/{payment_id}/cancel", httptransport.NewServer(
			e.CancelPayment,
			decodeCancelPaymentRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)

		r.Post("/pid/{payment_id}/link", httptransport.NewServer(
			e.GeneratePaymentLink,
			decodeGeneratePaymentLinkRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)

		r.Post("/pid/{payment_id}/transaction", httptransport.NewServer(
			e.GeneratePaymentTransaction,
			decodeGeneratePaymentTransactionRequest,
			httpencoder.EncodeResponse,
			options...,
		).ServeHTTP)
	})

	return r
}

// returns http error code by error type
func codeAndMessageFrom(err error) (int, interface{}) {
	if errors.Is(err, validator.ErrValidation) {
		return http.StatusPreconditionFailed, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, err
	}
	if resp := NewError(err); resp != nil {
		return resp.Code, resp
	}

	return httpencoder.CodeAndMessageFrom(err)
}

// DecodeGetAppInfoRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeGetAppInfoRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

// decodeGeneratePaymentTransactionRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeGeneratePaymentTransactionRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req GeneratePaymentTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	req.PaymentID = chi.URLParam(r, "payment_id")
	req.Currency = r.URL.Query().Get("currency")
	req.ApplyBonus = r.URL.Query().Get("apply_bonus")

	return req, nil
}

// decodeCreatePaymentRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeCreatePaymentRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// decodeGetPaymentInfoRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeGetPaymentInfoRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	pid, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil {
		return nil, ErrInvalidRequest
	}

	return pid, nil
}

// decodeGetPaymentInfoByExternalIDRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeGetPaymentInfoByExternalIDRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return chi.URLParam(r, "external_id"), nil
}

// decodeCancelPaymentRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeCancelPaymentRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	pid, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil {
		return nil, ErrInvalidRequest
	}

	return pid, nil
}

// decodeGeneratePaymentLinkRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded request from the HTTP request body.
func decodeGeneratePaymentLinkRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req GeneratePaymentLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	pid, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil {
		return nil, ErrInvalidRequest
	}
	req.PaymentID = pid

	return req, nil
}
