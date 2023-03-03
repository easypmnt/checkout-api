package solana

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/rpc"
)

type (
	// Client struct is a wrapper for the solana-go-sdk client.
	// It implements the SolanaClient interface.
	Client struct {
		rpcClient *client.Client
		wsClient  *client.Client
	}

	// ClientOption is a function that configures the Client.
	ClientOption func(*Client)
)

// NewClient creates a new Client instance.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	if c.rpcClient == nil {
		panic("rpc client is nil")
	}
	return c
}

// WithRPCClient sets the rpc client.
func WithRPCClient(rpcClient *client.Client) ClientOption {
	return func(c *Client) {
		c.rpcClient = rpcClient
	}
}

// WithRPCEndpoint sets the rpc endpoint.
func WithRPCEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.rpcClient = client.NewClient(endpoint)
	}
}

// WithWSClient sets the ws client.
func WithWSClient(wsClient *client.Client) ClientOption {
	return func(c *Client) {
		c.wsClient = wsClient
	}
}

// WithWSEndpoint sets the ws endpoint.
func WithWSEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.wsClient = client.NewClient(endpoint)
	}
}

// GetLatestBlockhash returns the latest blockhash.
func (c *Client) GetLatestBlockhash(ctx context.Context) (string, error) {
	blockhash, err := c.rpcClient.GetLatestBlockhash(ctx)
	if err != nil {
		return "", ErrGetLatestBlockhash
	}

	return blockhash.Blockhash, nil
}

// DoesTokenAccountExist returns true if the token account exists.
// Otherwise, it returns false.
func (c *Client) DoesTokenAccountExist(ctx context.Context, base58AtaAddr string) (bool, error) {
	ata, err := c.rpcClient.GetTokenAccount(ctx, base58AtaAddr)
	if err != nil {
		return false, ErrTokenAccountDoesNotExist
	}

	return ata.Mint.Bytes() != nil, nil
}

// RequestAirdrop sends a request to the solana network to airdrop SOL to the given account.
// Returns the transaction signature or an error.
func (c *Client) RequestAirdrop(ctx context.Context, base58Addr string, amount uint64) (string, error) {
	txSig, err := c.rpcClient.RequestAirdrop(ctx, base58Addr, amount)
	if err != nil {
		return "", errors.Wrap(err, "failed to request airdrop")
	}

	return txSig, nil
}

// GetSOLBalance returns the SOL balance in lamports of the given base58 encoded account address.
// Returns the balance or an error.
func (c *Client) GetSOLBalance(ctx context.Context, base58Addr string) (Balance, error) {
	balance, err := c.rpcClient.GetBalance(ctx, base58Addr)
	if err != nil {
		return Balance{}, errors.Wrap(err, "failed to get balance")
	}

	return NewBalance(balance, 9), nil
}

// GetAtaBalance returns the SPL token balance of the given base58 encoded associated token account address.
// base58Addr is the base58 encoded associated token account address.
// Returns the balance in lamports and token decimals, or an error.
func (c *Client) GetAtaBalance(ctx context.Context, base58Addr string) (Balance, error) {
	balance, decimals, err := c.rpcClient.GetTokenAccountBalance(ctx, base58Addr)
	if err != nil {
		return Balance{}, errors.Wrap(err, "failed to get token account balance")
	}

	return NewBalance(balance, decimals), nil
}

// GetTokenBalance returns the SPL token balance of the given base58 encoded account address and SPL token mint address.
// base58Addr is the base58 encoded account address.
// base58MintAddr is the base58 encoded SPL token mint address.
// Returns the Balance object, or an error.
func (c *Client) GetTokenBalance(ctx context.Context, base58Addr, base58MintAddr string) (Balance, error) {
	ata, _, err := common.FindAssociatedTokenAddress(
		common.PublicKeyFromString(base58Addr),
		common.PublicKeyFromString(base58MintAddr),
	)
	if err != nil {
		return Balance{}, errors.Wrap(err, "failed to find associated token address")
	}

	return c.GetAtaBalance(ctx, ata.String())
}

// GetTransactionStatus gets the transaction status.
// Returns the transaction status or an error.
func (c *Client) GetTransactionStatus(ctx context.Context, txhash string) (TransactionStatus, error) {
	status, err := c.rpcClient.GetSignatureStatus(ctx, txhash)
	if err != nil {
		return TransactionStatusUnknown, fmt.Errorf("failed to get transaction status: %v", err)
	}
	if status == nil {
		return TransactionStatusUnknown, nil
	}
	if status.Err != nil {
		return TransactionStatusFailure, fmt.Errorf("transaction failed: %v", status.Err)
	}

	result := TransactionStatusUnknown
	if status.Confirmations != nil && *status.Confirmations > 0 {
		result = TransactionStatusInProgress
	}
	if status.ConfirmationStatus != nil {
		result = ParseTransactionStatus(*status.ConfirmationStatus)
	}

	return result, nil
}

// SendTransaction sends a transaction to the network.
// Returns the transaction signature or an error.
func (c *Client) SendTransaction(ctx context.Context, txSource string) (string, error) {
	tx, err := DecodeTransaction(txSource)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: base64 to bytes: %w", err)
	}

	txSig, err := c.rpcClient.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return txSig, nil
}

// WaitForTransactionConfirmed waits for a transaction to be confirmed.
// Returns the transaction status or an error.
// If maxDuration is 0, it will wait for 5 minutes.
// Can be useful for testing, but not recommended for production because it may block requests for a long time.
func (c *Client) WaitForTransactionConfirmed(ctx context.Context, txhash string, maxDuration time.Duration) (TransactionStatus, error) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	if maxDuration == 0 {
		maxDuration = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return TransactionStatusUnknown, fmt.Errorf(
				"transaction %s is not confirmed after %s",
				txhash, maxDuration.String(),
			)
		case <-tick.C:
			status, err := c.GetTransactionStatus(ctx, txhash)
			if err != nil {
				return TransactionStatusUnknown, fmt.Errorf("failed to get transaction status: %w", err)
			}
			if status == TransactionStatusInProgress || status == TransactionStatusUnknown {
				continue
			}
			if status == TransactionStatusFailure || status == TransactionStatusSuccess {
				return status, nil
			}
		}
	}
}

// GetOldestTransactionForWallet returns the oldest transaction by the given base58 encoded public key.
// Returns the transaction or an error.
func (c *Client) GetOldestTransactionForWallet(
	ctx context.Context,
	base58Addr string,
	offsetTxSignature string,
) (*client.GetTransactionResponse, error) {
	limit := 1000
	result, err := c.rpcClient.GetSignaturesForAddressWithConfig(ctx, base58Addr, rpc.GetSignaturesForAddressConfig{
		Limit:      limit,
		Before:     offsetTxSignature,
		Commitment: rpc.CommitmentFinalized,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get signatures for address: %s: %w", base58Addr, err)
	}

	if l := len(result); l == 0 {
		return nil, ErrNoTransactionsFound
	} else if l < limit {
		tx := result[l-1]
		if tx.Err != nil {
			return nil, fmt.Errorf("transaction failed: %v", tx.Err)
		}
		if tx.Signature == "" {
			return nil, ErrNoTransactionsFound
		}
		if tx.BlockTime == nil || *tx.BlockTime == 0 || *tx.BlockTime > time.Now().Unix() {
			return nil, ErrTransactionNotConfirmed
		}

		resp, err := c.GetTransaction(ctx, tx.Signature)
		if err != nil {
			return nil, fmt.Errorf("failed to get oldest transaction for wallet: %s: %w", base58Addr, err)
		}

		return resp, nil
	}

	return c.GetOldestTransactionForWallet(ctx, base58Addr, result[limit-1].Signature)
}

// GetTransaction returns the transaction by the given base58 encoded transaction signature.
// Returns the transaction or an error.
func (c *Client) GetTransaction(ctx context.Context, txSignature string) (*client.GetTransactionResponse, error) {
	tx, err := c.rpcClient.GetTransaction(ctx, txSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if tx == nil || tx.Meta == nil {
		return nil, ErrTransactionNotFound
	}
	if tx.Meta.Err != nil {
		return nil, fmt.Errorf("transaction failed: %v", tx.Meta.Err)
	}

	return tx, nil
}
