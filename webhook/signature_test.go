package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignature(t *testing.T) {
	secretKey := []byte("secret")
	payload := []byte("payload")

	signature, err := SignPayload(payload, secretKey)
	require.NoError(t, err)
	require.NotEmpty(t, signature)

	err = VerifySignature(payload, signature, secretKey)
	require.NoError(t, err)
}
