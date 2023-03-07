package solana

import (
	"context"

	"github.com/easypmnt/checkout-api/utils"
	"github.com/portto/solana-go-sdk/rpc"
	"github.com/portto/solana-go-sdk/types"
)

type (
	// SolanaClient is an RPC client for Solana.
	SolanaClient interface {
		GetLatestBlockhash(ctx context.Context) (string, error)
		DoesTokenAccountExist(ctx context.Context, base58AtaAddr string) (bool, error)
	}

	// InstructionFunc is a function that returns a list of prepared instructions.
	InstructionFunc func(ctx context.Context, c SolanaClient) ([]types.Instruction, error)

	// Balance represents the balance of a token account or a wallet.
	Balance struct {
		Amount         uint64  `json:"amount"`           // Balance in minimal units. E.g. 1000000000 (1 SOL) or 1000000 (1 USDC).
		Decimals       uint8   `json:"decimals"`         // Number of decimals. E.g. 9 for SOL, 6 for USDC.
		UIAmount       float64 `json:"ui_amount"`        // Balance in UI units. E.g. 1 (1 SOL) or 1.000001 (1.000001 USDC).
		UIAmountString string  `json:"ui_amount_string"` // Balance in UI units as a string. E.g. "1" (1 SOL) or "1.000001" (1.000001 USDC).
	}
)

// NewBalance returns a new Balance instance.
func NewBalance(amount uint64, decimals uint8) Balance {
	return Balance{
		Amount:         amount,
		Decimals:       decimals,
		UIAmount:       utils.AmountToFloat64(amount, decimals),
		UIAmountString: utils.AmountToString(amount, decimals),
	}
}

// TransactionStatus represents the status of a transaction.
type TransactionStatus uint8

// TransactionStatus enum.
const (
	TransactionStatusUnknown TransactionStatus = iota
	TransactionStatusSuccess
	TransactionStatusInProgress
	TransactionStatusFailure
)

// TransactionStatusStrings is a map of TransactionStatus to string.
var transactionStatusStrings = map[TransactionStatus]string{
	TransactionStatusUnknown:    "unknown",
	TransactionStatusSuccess:    "success",
	TransactionStatusInProgress: "in_progress",
	TransactionStatusFailure:    "failure",
}

// String returns the string representation of the transaction status.
func (s TransactionStatus) String() string {
	return transactionStatusStrings[s]
}

// ParseTransactionStatus parses the transaction status from the given string.
func ParseTransactionStatus(s rpc.Commitment) TransactionStatus {
	switch s {
	case rpc.CommitmentFinalized:
		return TransactionStatusSuccess
	case rpc.CommitmentConfirmed, rpc.CommitmentProcessed:
		return TransactionStatusInProgress
	default:
		return TransactionStatusUnknown
	}
}
