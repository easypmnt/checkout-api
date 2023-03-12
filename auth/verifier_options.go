package auth

import "time"

// WithAccessTokenTTL sets the TTL for access tokens.
func WithAccessTokenTTL(ttl time.Duration) VarifierOption {
	return func(v *Verifier) {
		if ttl > 0 {
			v.accessTokenTTL = ttl
		}
	}
}

// WithRefreshTokenTTL sets the TTL for refresh tokens.
func WithRefreshTokenTTL(ttl time.Duration) VarifierOption {
	return func(v *Verifier) {
		if ttl > 0 {
			v.refreshTokenTTL = ttl
		}
	}
}
