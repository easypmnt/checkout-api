package solana

import (
	"fmt"

	"github.com/easypmnt/checkout-api/utils"
	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/types"
)

// EncodeTransaction returns a base64 encoded transaction.
func EncodeTransaction(tx types.Transaction) (string, error) {
	txb, err := tx.Serialize()
	if err != nil {
		return "", errors.Wrap(err, "failed to build transaction: serialize")
	}

	return utils.BytesToBase64(txb), nil
}

// DecodeTransaction returns a transaction from a base64 encoded transaction.
func DecodeTransaction(base64Tx string) (types.Transaction, error) {
	txb, err := utils.Base64ToBytes(base64Tx)
	if err != nil {
		return types.Transaction{}, errors.Wrap(err, "failed to deserialize transaction: base64 to bytes")
	}

	tx, err := types.TransactionDeserialize(txb)
	if err != nil {
		return types.Transaction{}, errors.Wrap(err, "failed to deserialize transaction: deserialize")
	}

	return tx, nil
}

// SignTransaction signs a transaction and returns a base64 encoded transaction.
func SignTransaction(txSource string, signer types.Account) (string, error) {
	tx, err := DecodeTransaction(txSource)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: base64 to bytes: %w", err)
	}

	msg, err := tx.Message.Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: serialize message: %w", err)
	}

	if err := tx.AddSignature(signer.Sign(msg)); err != nil {
		return "", fmt.Errorf("failed to sign transaction: add signature: %w", err)
	}

	result, err := EncodeTransaction(tx)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: encode transaction: %w", err)
	}

	return result, nil
}
