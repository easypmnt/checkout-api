package solana

import "errors"

// Predefined package errors.
var (
	ErrClientNotSet              = errors.New("solana rpc client not set")
	ErrFeePayerNotSet            = errors.New("missing or invalid fee payer public key")
	ErrNoInstruction             = errors.New("no instructions added, require at least one instruction")
	ErrSenderAndRecipientAreSame = errors.New("sender and recipient are the same account")
	ErrMustBeGreaterThanZero     = errors.New("amount must be greater than 0")
	ErrSenderIsRequired          = errors.New("sender wallet address is required")
	ErrRecipientIsRequired       = errors.New("recipient wallet address is required")
	ErrMintIsRequired            = errors.New("mint address is required")
	ErrMemoCannotBeEmpty         = errors.New("memo cannot be empty")
	ErrGetLatestBlockhash        = errors.New("failed to get latest blockhash")
	ErrTokenAccountDoesNotExist  = errors.New("token account does not exist")
	ErrNoTransactionsFound       = errors.New("no transactions found")
	ErrTransactionNotConfirmed   = errors.New("transaction not confirmed")
	ErrTransactionNotFound       = errors.New("transaction not found")
)
