package solana

import (
	"fmt"
	"strconv"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/client"
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

// CheckSolTransferTransaction checks if a transaction is a SOL transfer transaction.
// Verifies that destination account has been credited with the correct amount.
func CheckSolTransferTransaction(meta *client.TransactionMeta, tx types.Transaction, destination string, amount uint64) error {
	var destIdx int
	for i, acc := range tx.Message.Accounts {
		if acc.ToBase58() == destination {
			destIdx = i
			break
		}
	}

	txAmount := meta.PostBalances[destIdx] - meta.PreBalances[destIdx]
	if txAmount != int64(amount) {
		return fmt.Errorf("amount is not equal to the amount in the transaction: %d != %d", amount, txAmount)
	}

	return nil
}

// CheckTokenTransferTransaction checks if a transaction is a token transfer transaction.
// Verifies that destination account has been credited with the correct amount of the token.
func CheckTokenTransferTransaction(meta *client.TransactionMeta, tx types.Transaction, mint, destination string, amount uint64) error {
	var preBalance uint64
	var postBalance uint64

	for _, balance := range meta.PreTokenBalances {
		if balance.Mint == mint && balance.Owner == destination {
			amount, err := strconv.ParseUint(balance.UITokenAmount.Amount, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse pre balance: %w", err)
			}
			preBalance = amount
			break
		}
	}

	for _, balance := range meta.PostTokenBalances {
		if balance.Mint == mint && balance.Owner == destination {
			amount, err := strconv.ParseUint(balance.UITokenAmount.Amount, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse post balance: %w", err)
			}
			postBalance = amount
			break
		}
	}

	if postBalance-preBalance != amount {
		return fmt.Errorf("amount is not equal to the amount in the transaction: %d != %d", amount, postBalance-preBalance)
	}

	return nil
}
