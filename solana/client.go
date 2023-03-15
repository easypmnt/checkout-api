package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/easypmnt/checkout-api/solana/metadata"
	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/program/metaplex/token_metadata"
	"github.com/portto/solana-go-sdk/rpc"
)

type (
	// Client struct is a wrapper for the solana-go-sdk client.
	// It implements the SolanaClient interface.
	Client struct {
		rpcClient     *client.Client
		wsClient      *client.Client
		tokenListPath string
	}

	// ClientOption is a function that configures the Client.
	ClientOption func(*Client)
)

// NewClient creates a new Client instance.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		tokenListPath: "https://raw.githubusercontent.com/solana-labs/token-list/main/src/tokens/solana.tokenlist.json",
	}
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

// WithTokenListPath sets the token list path.
func WithTokenListPath(path string) ClientOption {
	return func(c *Client) {
		c.tokenListPath = path
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

// GetMinimumBalanceForRentExemption gets the minimum balance for rent exemption.
// Returns the minimum balance in lamports or an error.
func (c *Client) GetMinimumBalanceForRentExemption(ctx context.Context, size uint64) (uint64, error) {
	mintAccountRent, err := c.rpcClient.GetMinimumBalanceForRentExemption(ctx, size)
	if err != nil {
		return 0, fmt.Errorf("failed to get minimum balance for rent exemption: %w", err)
	}

	return mintAccountRent, nil
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
) (string, *client.GetTransactionResponse, error) {
	limit := 1000
	result, err := c.rpcClient.GetSignaturesForAddressWithConfig(ctx, base58Addr, rpc.GetSignaturesForAddressConfig{
		Limit:      limit,
		Before:     offsetTxSignature,
		Commitment: rpc.CommitmentFinalized,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to get signatures for address: %s: %w", base58Addr, err)
	}

	if l := len(result); l == 0 {
		return "", nil, ErrNoTransactionsFound
	} else if l < limit {
		tx := result[l-1]
		if tx.Err != nil {
			return "", nil, fmt.Errorf("transaction failed: %v", tx.Err)
		}
		if tx.Signature == "" {
			return "", nil, ErrNoTransactionsFound
		}
		if tx.BlockTime == nil || *tx.BlockTime == 0 || *tx.BlockTime > time.Now().Unix() {
			return "", nil, ErrTransactionNotConfirmed
		}

		resp, err := c.GetTransaction(ctx, tx.Signature)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get oldest transaction for wallet: %s: %w", base58Addr, err)
		}

		return tx.Signature, resp, nil
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

// GetTokenSupply returns the token supply for a given mint address.
// This is a wrapper around the GetTokenSupply function from the solana-go-sdk.
// base58MintAddr is the base58 encoded address of the token mint.
// The function returns the token supply and decimals or an error.
func (c *Client) GetTokenSupply(ctx context.Context, base58MintAddr string) (Balance, error) {
	amount, decimals, err := c.rpcClient.GetTokenSupply(ctx, base58MintAddr)
	if err != nil {
		return Balance{}, fmt.Errorf("failed to get token supply: %w", err)
	}

	return NewBalance(amount, decimals), nil
}

// GetFungibleTokenMetadata returns the on-chain SPL token metadata by the given base58 encoded SPL token mint address.
// Returns the token metadata or an error.
func (c *Client) GetFungibleTokenMetadata(ctx context.Context, base58MintAddr string) (result *FungibleTokenMetadata, err error) {
	// fallback to the deprecated metadata account if the given mint address has no on-chain metadata
	defer func() {
		if result == nil || err != nil || result.Name == "" || result.Symbol == "" || result.LogoURI == "" {
			if depr, err := c.getDeprecatedTokenMetadata(ctx, base58MintAddr); err == nil {
				if result == nil {
					result = depr
				} else {
					if result.Name == "" {
						result.Name = depr.Name
					}
					if result.Symbol == "" {
						result.Symbol = depr.Symbol
					}
					if result.LogoURI == "" {
						result.LogoURI = depr.LogoURI
					}
					if result.ExternalURL == "" {
						result.ExternalURL = depr.ExternalURL
					}
					if result.Description == "" {
						result.Description = depr.Description
					}
				}
			}
		}
	}()

	metadataAccount, err := token_metadata.GetTokenMetaPubkey(common.PublicKeyFromString(base58MintAddr))
	if err != nil {
		return result, fmt.Errorf("failed to get token metadata account: %w", err)
	}

	accountInfo, err := c.rpcClient.GetAccountInfo(ctx, metadataAccount.ToBase58())
	if err != nil {
		return result, fmt.Errorf("failed to get account info: %w", err)
	}

	md, err := token_metadata.MetadataDeserialize(accountInfo.Data)
	if err != nil {
		return result, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	result = &FungibleTokenMetadata{
		Mint:   base58MintAddr,
		Name:   md.Data.Name,
		Symbol: md.Data.Symbol,
	}

	if sup, err := c.GetTokenSupply(ctx, base58MintAddr); err == nil {
		result.Decimals = sup.Decimals
	}

	if md.Data.Uri != "" && strings.HasPrefix(md.Data.Uri, "http") {
		mde, err := metadata.MetadataFromURI(md.Data.Uri)
		if err != nil {
			return result, fmt.Errorf("failed to get additional metadata from uri: %w", err)
		}

		result.Description = mde.Description
		result.LogoURI = mde.Image
		result.ExternalURL = mde.ExternalURL
	}

	return result, nil
}

// @deprecated
// getDeprecatedTokenMetadata returns the deprecated SPL token metadata by the given base58 encoded SPL token mint address.
// This is a temporary solution to support the deprecated metadata format.
// Returns the token metadata or an error.
// Works only with mainnet.
func (c *Client) getDeprecatedTokenMetadata(_ context.Context, base58MintAddr string) (*FungibleTokenMetadata, error) {
	if c.tokenListPath == "" || base58MintAddr == "" {
		return nil, fmt.Errorf("failed to get token metadata: token list path or mint address is empty")
	}

	resp, err := http.Get(c.tokenListPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download token list from uri: %w", err)
	}
	defer resp.Body.Close()

	var tokenList TokenList
	if err := json.NewDecoder(resp.Body).Decode(&tokenList); err != nil {
		return nil, fmt.Errorf("failed to decode token list from uri: %w", err)
	}

	// Find token metadata.
	var tokenMeta TokenListToken
	for _, token := range tokenList.Tokens {
		if token.Address == base58MintAddr && token.ChainID == ChainIdMainnet {
			tokenMeta = token
			break
		}
	}

	result := FungibleTokenMetadata{
		Mint:     base58MintAddr,
		Name:     tokenMeta.Name,
		Symbol:   tokenMeta.Symbol,
		Decimals: uint8(tokenMeta.Decimals),
		LogoURI:  tokenMeta.LogoURI,
	}

	if tokenMeta.Extensions != nil {
		if tokenMeta.Extensions["description"] != nil {
			result.Description = tokenMeta.Extensions["description"].(string)
		}
		if tokenMeta.Extensions["website"] != nil {
			result.ExternalURL = tokenMeta.Extensions["website"].(string)
		} else if tokenMeta.Extensions["twitter"] != nil {
			result.ExternalURL = tokenMeta.Extensions["twitter"].(string)
		} else if tokenMeta.Extensions["discord"] != nil {
			result.ExternalURL = tokenMeta.Extensions["discord"].(string)
		}
	}

	return &result, nil
}

// ValidateTransactionByReference returns the transaction by the given reference.
// Returns transaction signature or an error if the transaction is not found or the transaction failed.
func (c *Client) ValidateTransactionByReference(ctx context.Context, reference, destination string, amount uint64, mint string) (string, error) {
	txSign, tx, err := c.GetOldestTransactionForWallet(ctx, reference, "")
	if err != nil {
		return "", fmt.Errorf("failed to validate transaction for reference %s: %w", reference, err)
	}

	if mint == "" || mint == "SOL" || mint == "So11111111111111111111111111111111111111112" {
		if err := CheckSolTransferTransaction(tx.Meta, tx.Transaction, destination, amount); err != nil {
			return "", fmt.Errorf("failed to validate transaction for reference %s: %w", reference, err)
		}
		return txSign, nil
	}

	if err := CheckTokenTransferTransaction(tx.Meta, tx.Transaction, mint, destination, amount); err != nil {
		return "", fmt.Errorf("failed to validate transaction for reference %s: %w", reference, err)
	}

	return txSign, nil
}
