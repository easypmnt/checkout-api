package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/easypmnt/checkout-api/repository"
	"github.com/go-chi/oauth"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	// Verifier is the service that validates the client credentials.
	// Implements the interface gihub.com/go-chi/oauth/server.go.CredentialsVerifier
	Verifier struct {
		repo verifierRepository

		clientID         string
		clientSecretHash string // bcrypt hash of the client secret, used for comparison.
		accessTokenTTL   time.Duration
		refreshTokenTTL  time.Duration
	}

	// VerifierOption is a function that configures the Verifier.
	VarifierOption func(*Verifier)

	verifierRepository interface {
		GetToken(ctx context.Context, arg repository.GetTokenParams) (repository.Token, error)
		StoreToken(ctx context.Context, arg repository.StoreTokenParams) (repository.Token, error)
	}
)

// NewVerifier creates a new Verifier.
func NewVerifier(repo verifierRepository, clientID, clientSecretHash string, opts ...VarifierOption) *Verifier {
	if clientID == "" || clientSecretHash == "" {
		panic("Client id and secret hash are required")
	}

	v := &Verifier{
		repo:             repo,
		clientID:         clientID,
		clientSecretHash: clientSecretHash,
		accessTokenTTL:   time.Hour,
		refreshTokenTTL:  time.Hour * 24 * 30,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// Validate username and password returning an error if the user credentials are wrong
func (v *Verifier) ValidateUser(username, password, scope string, r *http.Request) error {
	return ErrPasswordNotSupported
}

// Validate clientID and secret returning an error if the client credentials are wrong
func (v *Verifier) ValidateClient(clientID, clientSecret, _ string, r *http.Request) error {
	if clientID != v.clientID {
		return ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(v.clientSecretHash), []byte(clientSecret)) != nil {
		return ErrInvalidCredentials
	}
	return nil
}

// Provide additional claims to the token
func (v *Verifier) AddClaims(tokenType oauth.TokenType, credential, tokenID, scope string, r *http.Request) (map[string]string, error) {
	return nil, nil
}

// Provide additional information to the authorization server response
func (v *Verifier) AddProperties(tokenType oauth.TokenType, credential, tokenID, scope string, r *http.Request) (map[string]string, error) {
	return nil, nil
}

// Optionally validate previously stored tokenID during refresh request
func (v *Verifier) ValidateTokenID(tokenType oauth.TokenType, credential, tokenID, refreshTokenID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	accessID, err := uuid.Parse(tokenID)
	if err != nil {
		return ErrInvalidToken
	}
	refreshID, err := uuid.Parse(refreshTokenID)
	if err != nil {
		return ErrInvalidToken
	}

	token, err := v.repo.GetToken(ctx, repository.GetTokenParams{
		TokenType:      string(tokenType),
		Credential:     credential,
		AccessTokenID:  accessID,
		RefreshTokenID: refreshID,
	})
	if err != nil {
		return ErrInvalidToken
	}

	if token.AccessExpiresAt.Before(time.Now()) ||
		token.RefreshExpiresAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	return nil
}

// Optionally store the tokenID generated for the user
func (v *Verifier) StoreTokenID(tokenType oauth.TokenType, credential, tokenID, refreshTokenID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	accessID, err := uuid.Parse(tokenID)
	if err != nil {
		return ErrInvalidToken
	}
	refreshID, err := uuid.Parse(refreshTokenID)
	if err != nil {
		return ErrInvalidToken
	}

	if _, err := v.repo.StoreToken(ctx, repository.StoreTokenParams{
		TokenType:        string(tokenType),
		Credential:       credential,
		AccessTokenID:    accessID,
		RefreshTokenID:   refreshID,
		AccessExpiresAt:  time.Now().Add(v.accessTokenTTL),
		RefreshExpiresAt: time.Now().Add(v.refreshTokenTTL),
	}); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}
