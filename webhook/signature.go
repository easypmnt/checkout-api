package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/easypmnt/checkout-api/internal/utils"
)

// SignPayload signs a payload using a secret key and returns the signature as a base64 encoded string
func SignPayload(payload []byte, secretKey []byte) (string, error) {
	hash := hmac.New(sha256.New, secretKey)
	if _, err := hash.Write(payload); err != nil {
		return "", fmt.Errorf("failed to write payload to hash: %w", err)
	}

	return utils.BytesToBase64(hash.Sum(nil)), nil
}

// VerifySignature verifies a signature against a payload using a secret key
func VerifySignature(payload []byte, signature string, secretKey []byte) error {
	expectedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	hash := hmac.New(sha256.New, secretKey)
	if _, err := hash.Write(payload); err != nil {
		return fmt.Errorf("failed to write payload to hash: %w", err)
	}

	actualSignature := hash.Sum(nil)
	if !hmac.Equal(expectedSignature, actualSignature) {
		return errors.New("signature verification failed")
	}

	return nil
}
