package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/oauth"
)

// Set up limited oauth2 server for client_credentials, and refresh_token flows.
// Does not support password, authorization code flow.
func NewOAuth2Server(singingKey string, ttl time.Duration, verifier oauth.CredentialsVerifier) *oauth.BearerServer {
	if ttl == 0 {
		ttl = time.Hour
	}
	if verifier == nil {
		panic("Credentials verifier is not set")
	}

	return oauth.NewBearerServer(singingKey, ttl, verifier, nil)
}

// MakeHTTPHandler returns an http.Handler that can be used to serve the OAuth2 API.
func MakeHTTPHandler(oauthSvc interface {
	ClientCredentials(w http.ResponseWriter, r *http.Request)
},
) http.Handler {
	r := chi.NewRouter()
	r.Post("/token", oauthSvc.ClientCredentials)
	return r
}
