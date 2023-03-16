package solana

import (
	"context"
	"fmt"
	"strings"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/easypmnt/checkout-api/solana/metadata"
	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/program/associated_token_account"
	"github.com/portto/solana-go-sdk/program/memo"
	"github.com/portto/solana-go-sdk/program/metaplex/token_metadata"
	"github.com/portto/solana-go-sdk/program/system"
	"github.com/portto/solana-go-sdk/program/token"
	"github.com/portto/solana-go-sdk/types"
)

// CreateAssociatedTokenAccountParam defines the parameters for creating an associated token account.
type CreateAssociatedTokenAccountParam struct {
	Funder string // base58 encoded public key of the account that will fund the associated token account. Must be a signer.
	Owner  string // base58 encoded public key of the owner of the associated token account. Must be a signer.
	Mint   string // base58 encoded public key of the mint of the associated token account.
}

// CreateAssociatedTokenAccountIfNotExists creates an associated token account for
// the given owner and mint if it does not exist.
func CreateAssociatedTokenAccountIfNotExists(params CreateAssociatedTokenAccountParam) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		var (
			funderPubKey = common.PublicKeyFromString(params.Funder)
			ownerPubKey  = common.PublicKeyFromString(params.Owner)
			mintPubKey   = common.PublicKeyFromString(params.Mint)
		)

		ata, _, err := common.FindAssociatedTokenAddress(ownerPubKey, mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to find associated token address: %w", err)
		}
		if exists, err := c.DoesTokenAccountExist(ctx, ata.ToBase58()); err == nil && exists {
			return nil, nil
		}

		return []types.Instruction{
			associated_token_account.CreateAssociatedTokenAccount(
				associated_token_account.CreateAssociatedTokenAccountParam{
					Funder:                 funderPubKey,
					Owner:                  ownerPubKey,
					Mint:                   mintPubKey,
					AssociatedTokenAccount: ata,
				},
			),
		}, nil
	}
}

// Memo returns a list of instructions that can be used to add a memo to transaction.
func Memo(str string, signers ...string) InstructionFunc {
	return func(ctx context.Context, _ SolanaClient) ([]types.Instruction, error) {
		if str == "" {
			return nil, ErrMemoCannotBeEmpty
		}

		signersPubKeys := make([]common.PublicKey, 0, len(signers))
		for _, signer := range signers {
			if signer == "" {
				continue
			}
			signersPubKeys = append(signersPubKeys, common.PublicKeyFromString(signer))
		}

		return []types.Instruction{
			memo.BuildMemo(memo.BuildMemoParam{
				SignerPubkeys: signersPubKeys,
				Memo:          []byte(str),
			}),
		}, nil
	}
}

// TransferSOLParams defines the parameters for transferring SOL.
type TransferSOLParams struct {
	Sender    string // required; base58 encoded public key of the sender. Must be a signer.
	Recipient string // required; base58 encoded public key of the recipient.
	Reference string // optional; base58 encoded public key to use as a reference for the transaction.
	Amount    uint64 // required; the amount of SOL to send (in lamports). Must be greater than minimum account rent exemption (~0.0025 SOL).
}

// Validate validates the parameters.
func (p TransferSOLParams) Validate() error {
	if p.Sender == "" {
		return ErrSenderIsRequired
	}
	if p.Recipient == "" {
		return ErrRecipientIsRequired
	}
	if p.Sender == p.Recipient {
		return ErrSenderAndRecipientAreSame
	}
	if p.Amount <= 0 {
		return ErrMustBeGreaterThanZero
	}
	return nil
}

// TransferSOL transfers SOL from one wallet to another.
// Note: This function does not check if the sender has enough SOL to send. It is the responsibility
// of the caller to check this.
// Amount must be greater than minimum account rent exemption (~0.0025 SOL).
func TransferSOL(params TransferSOLParams) InstructionFunc {
	return func(ctx context.Context, _ SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, errors.Wrap(err, "invalid parameters for TransferSOL instruction")
		}

		var (
			senderPubKey    = common.PublicKeyFromString(params.Sender)
			recipientPubKey = common.PublicKeyFromString(params.Recipient)
		)

		instruction := system.Transfer(system.TransferParam{
			From:   senderPubKey,
			To:     recipientPubKey,
			Amount: params.Amount,
		})

		if params.Reference != "" {
			instruction.Accounts = append(instruction.Accounts, types.AccountMeta{
				PubKey:     common.PublicKeyFromString(params.Reference),
				IsSigner:   false,
				IsWritable: false,
			})
		}

		return []types.Instruction{instruction}, nil
	}
}

// TransferTokenParam defines the parameters for transferring tokens.
type TransferTokenParam struct {
	Sender    string // required; base58 encoded public key of the sender. Must be a signer.
	Recipient string // required; base58 encoded public key of the recipient.
	Mint      string // required; base58 encoded public key of the mint of the token to send.
	Reference string // optional; base58 encoded public key to use as a reference for the transaction.
	Amount    uint64 // required; the amount of tokens to send (in token minimal units), e.g. 1 USDT = 1000000 (10^6) lamports.
}

// Validate validates the parameters.
func (p TransferTokenParam) Validate() error {
	if p.Sender == "" {
		return ErrSenderIsRequired
	}
	if p.Recipient == "" {
		return ErrRecipientIsRequired
	}
	if p.Sender == p.Recipient {
		return ErrSenderAndRecipientAreSame
	}
	if p.Mint == "" {
		return ErrMintIsRequired
	}
	if p.Amount <= 0 {
		return ErrMustBeGreaterThanZero
	}
	return nil
}

// TransferToken transfers tokens from one wallet to another.
// Note: This function does not check if the sender has enough tokens to send. It is the responsibility
// of the caller to check this.
// FeePayer must be provided if Sender is not set.
func TransferToken(params TransferTokenParam) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, errors.Wrap(err, "invalid parameters for TransferToken instruction")
		}

		var (
			senderPubKey    = common.PublicKeyFromString(params.Sender)
			recipientPubKey = common.PublicKeyFromString(params.Recipient)
			mintPubKey      = common.PublicKeyFromString(params.Mint)
		)
		senderAta, _, err := common.FindAssociatedTokenAddress(senderPubKey, mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to find associated token address for sender wallet: %w", err)
		}
		recipientAta, _, err := common.FindAssociatedTokenAddress(recipientPubKey, mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to find associated token address for recipient wallet: %w", err)
		}

		instructions := make([]types.Instruction, 0, 2)

		if exists, _ := c.DoesTokenAccountExist(ctx, recipientAta.ToBase58()); !exists {
			instructions = append(instructions,
				associated_token_account.CreateAssociatedTokenAccount(
					associated_token_account.CreateAssociatedTokenAccountParam{
						Funder:                 senderPubKey,
						Owner:                  recipientPubKey,
						Mint:                   mintPubKey,
						AssociatedTokenAccount: recipientAta,
					},
				),
			)
		}

		instruction := token.Transfer(token.TransferParam{
			From:   senderAta,
			To:     recipientAta,
			Auth:   senderPubKey,
			Amount: params.Amount,
		})

		if params.Reference != "" {
			instruction.Accounts = append(instruction.Accounts, types.AccountMeta{
				PubKey:     common.PublicKeyFromString(params.Reference),
				IsSigner:   false,
				IsWritable: false,
			})
		}

		instructions = append(instructions, instruction)

		return instructions, nil
	}
}

// CreateFungibleTokenParam defines the parameters for the CreateFungibleToken instruction.
type CreateFungibleTokenParam struct {
	Mint     string // required; The token mint public key.
	Owner    string // required; The owner of the token.
	FeePayer string // required; The wallet to pay the fees from.

	Decimals    uint8  // optional; The number of decimals the token has. Default is 0.
	MetadataURI string // optional; URI of the token metadata; can be set later
	TokenName   string // optional; Name of the token; used for the token metadata if MetadataURI is not set.
	TokenSymbol string // optional; Symbol of the token; used for the token metadata if MetadataURI is not set.
}

// Validate checks that the required fields of the params are set.
func (p CreateFungibleTokenParam) Validate() error {
	if p.Mint == "" {
		return fmt.Errorf("mint address is required")
	}
	if p.Owner == "" {
		return fmt.Errorf("owner public key is required")
	}
	if p.FeePayer == "" {
		return fmt.Errorf("invalid fee payer public key")
	}
	if p.MetadataURI != "" && !strings.HasPrefix(p.MetadataURI, "http") {
		return fmt.Errorf("field MetadataURI must be a valid URI")
	}
	if p.MetadataURI == "" && (p.TokenName == "" || p.TokenSymbol == "") {
		return fmt.Errorf("field TokenName and TokenSymbol are required if MetadataURI is not set")
	}
	if p.TokenName != "" && (len(p.TokenName) < 2 || len(p.TokenName) > 32) {
		return fmt.Errorf("token name must be between 2 and 32 characters")
	}
	if p.TokenSymbol != "" && (len(p.TokenSymbol) < 3 || len(p.TokenSymbol) > 10) {
		return fmt.Errorf("token symbol must be between 3 and 10 characters")
	}
	return nil
}

// CreateFungibleToken creates instructions for minting fungible tokens or assets.
// The token mint account must be created before calling this function.
// To mint common fungible tokens, decimals must be greater than 0.
// If decimals is 0, the token is fungible asset.
func CreateFungibleToken(params CreateFungibleTokenParam) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}

		var (
			mintPubKey  = common.PublicKeyFromString(params.Mint)
			ownerPubKey = common.PublicKeyFromString(params.Owner)
			feePayer    = common.PublicKeyFromString(params.FeePayer)
		)

		metaPubkey, err := token_metadata.GetTokenMetaPubkey(mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get token metadata pubkey: %w", err)
		}

		var metadataV2 token_metadata.DataV2
		if params.MetadataURI != "" {
			md, err := metadata.MetadataFromURI(params.MetadataURI)
			if err != nil {
				return nil, fmt.Errorf("failed to get metadata from URI: %w", err)
			}

			if md.Name == "" || len(md.Name) < 2 || len(md.Name) > 32 {
				return nil, fmt.Errorf("metadata name must be between 2 and 32 characters")
			}
			if md.Symbol == "" || len(md.Symbol) < 2 || len(md.Symbol) > 10 {
				return nil, fmt.Errorf("metadata symbol must be between 2 and 10 characters")
			}

			metadataV2 = token_metadata.DataV2{
				Name:   md.Name,
				Symbol: md.Symbol,
				Uri:    params.MetadataURI,
			}
		} else {
			metadataV2 = token_metadata.DataV2{
				Name:   params.TokenName,
				Symbol: params.TokenSymbol,
			}
		}

		rentExemption, err := c.GetMinimumBalanceForRentExemption(ctx, token.MintAccountSize)
		if err != nil {
			return nil, fmt.Errorf("failed to get minimum balance for rent exemption: %w", err)
		}

		instructions := []types.Instruction{
			system.CreateAccount(system.CreateAccountParam{
				From:     feePayer,
				New:      mintPubKey,
				Owner:    common.TokenProgramID,
				Lamports: rentExemption,
				Space:    token.MintAccountSize,
			}),
			token.InitializeMint2(token.InitializeMint2Param{
				Decimals:   params.Decimals,
				Mint:       mintPubKey,
				MintAuth:   ownerPubKey,
				FreezeAuth: utils.Pointer(ownerPubKey),
			}),
			token_metadata.CreateMetadataAccountV2(token_metadata.CreateMetadataAccountV2Param{
				Metadata:                metaPubkey,
				Mint:                    mintPubKey,
				MintAuthority:           ownerPubKey,
				Payer:                   feePayer,
				UpdateAuthority:         ownerPubKey,
				UpdateAuthorityIsSigner: true,
				IsMutable:               true,
				Data:                    metadataV2,
			}),
		}

		return instructions, nil
	}
}

// UpdateFungibleMetadataParams is the params for UpdateMetadata
type UpdateFungibleMetadataParams struct {
	Mint            string // required; The mint address of the token
	UpdateAuthority string // required; The update authority of the token
	MetadataUri     string // optional; new metadata json uri
}

// Validate validates the params.
func (p UpdateFungibleMetadataParams) Validate() error {
	if p.Mint == "" {
		return fmt.Errorf("mint address is required")
	}
	if p.UpdateAuthority == "" {
		return fmt.Errorf("update authority is required")
	}
	if p.MetadataUri == "" ||
		(!strings.HasPrefix(p.MetadataUri, "http://") &&
			!strings.HasPrefix(p.MetadataUri, "https://")) {
		return fmt.Errorf("metadata uri is invalid")
	}
	return nil
}

// UpdateFungibleMetadata updates the metadata of the fungible token.
func UpdateFungibleMetadata(params UpdateFungibleMetadataParams) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("validate update metadata: %w", err)
		}

		var (
			mintPubKey = common.PublicKeyFromString(params.Mint)
			authPubKey = common.PublicKeyFromString(params.UpdateAuthority)
		)

		tokenMetadataPubkey, err := token_metadata.GetTokenMetaPubkey(mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to derive token metadata pubkey: %w", err)
		}

		newMeta, err := metadata.MetadataFromURI(params.MetadataUri)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata from URI: %w", err)
		}

		instructions := []types.Instruction{
			token_metadata.UpdateMetadataAccount(token_metadata.UpdateMetadataAccountParam{
				MetadataAccount: tokenMetadataPubkey,
				UpdateAuthority: authPubKey,
				Data: &token_metadata.Data{
					Name:   newMeta.Name,
					Symbol: newMeta.Symbol,
					Uri:    params.MetadataUri,
				},
			}),
		}

		return instructions, nil
	}
}

// MintFungibleTokenParams is the params for MintFungibleToken
type MintFungibleTokenParams struct {
	Funder    string // base58 encoded public key of the account that will fund the associated token account. Must be a signer.
	Mint      string // base58 encoded public key of the mint
	MintOwner string // base58 encoded public key of the mint owner
	MintTo    string // base58 encoded public key of the account that will receive the minted tokens
	Amount    uint64 // amount of tokens to mint in basis points, for example, 1 token with 9 decimals = 1000000000 bps.
}

// Validate validates the params.
func (p MintFungibleTokenParams) Validate() error {
	if p.Funder == "" {
		return fmt.Errorf("funder public key is required")
	}
	if p.Mint == "" {
		return fmt.Errorf("mint address is required")
	}
	if p.MintOwner == "" {
		return fmt.Errorf("mint owner public key is required")
	}
	if p.MintTo == "" {
		return fmt.Errorf("mintTo public key is required")
	}
	if p.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

// MintFungibleToken mints the fungible token.
func MintFungibleToken(params MintFungibleTokenParams) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("validate mint token: %w", err)
		}

		var (
			funderPubKey = common.PublicKeyFromString(params.Funder)
			ownerPubKey  = common.PublicKeyFromString(params.MintOwner)
			mintPubKey   = common.PublicKeyFromString(params.Mint)
			mintToPubKey = common.PublicKeyFromString(params.MintTo)
		)

		mintToAta, _, err := common.FindAssociatedTokenAddress(mintToPubKey, mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to find associated token address: %w", err)
		}

		instructions := make([]types.Instruction, 0, 2)

		if exists, _ := c.DoesTokenAccountExist(ctx, mintToAta.ToBase58()); !exists {
			instructions = append(instructions,
				associated_token_account.CreateAssociatedTokenAccount(
					associated_token_account.CreateAssociatedTokenAccountParam{
						Funder:                 funderPubKey,
						Owner:                  mintToPubKey,
						Mint:                   mintPubKey,
						AssociatedTokenAccount: mintToAta,
					},
				),
			)
		}

		instructions = append(instructions,
			token.MintTo(token.MintToParam{
				Mint:    mintPubKey,
				To:      mintToAta,
				Auth:    ownerPubKey,
				Signers: []common.PublicKey{},
				Amount:  params.Amount,
			}),
		)

		return instructions, nil
	}
}

// BurnTokenParams are the parameters for the BurnToken instruction.
type BurnTokenParams struct {
	Mint              string // base58 encoded public key of the mint
	TokenAccountOwner string // base58 encoded public key of the token account owner
	Amount            uint64
}

// Validate checks that the required fields of the params are set.
func (p BurnTokenParams) Validate() error {
	if p.Mint == "" {
		return fmt.Errorf("mint is required")
	}
	if p.TokenAccountOwner == "" {
		return fmt.Errorf("token account owner is required")
	}
	if p.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

// BurnToken burns the specified token.
func BurnToken(params BurnTokenParams) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate params: %w", err)
		}

		var (
			mintPubKey     = common.PublicKeyFromString(params.Mint)
			ataOwnerPubKey = common.PublicKeyFromString(params.TokenAccountOwner)
		)

		ata, _, err := common.FindAssociatedTokenAddress(ataOwnerPubKey, mintPubKey)
		if err != nil {
			return nil, fmt.Errorf("failed to find associated token address: %w", err)
		}

		return []types.Instruction{
			token.Burn(token.BurnParam{
				Account: ata,
				Mint:    mintPubKey,
				Auth:    ataOwnerPubKey,
				Amount:  params.Amount,
			}),
		}, nil
	}
}

// CloseTokenAccountParams are the parameters for the CloseTokenAccount instruction.
type CloseTokenAccountParams struct {
	Owner             string  // base58 encoded public key of the owner of the token account.
	CloseTokenAccount *string // optional; base58 encoded public key of the token account to close; if not set, the associated token account will be derived from the owner and mint.
	Mint              *string // optional; base58 encoded public key of the mint; if not set, the CloseTokenAccount must be set.
	FeePayer          *string // optional;  base58 encoded public key of the fee payer of the transaction, if not set, the owner will be used; if set, the rent exemption balance will be transferred to it.
}

// Validate checks that the required fields of the params are set.
func (p CloseTokenAccountParams) Validate() error {
	if p.Owner == "" {
		return fmt.Errorf("owner is required")
	}
	if p.CloseTokenAccount == nil && p.Mint == nil {
		return fmt.Errorf("closeTokenAccount or mint is required")
	}
	if p.FeePayer != nil && *p.FeePayer == "" {
		return fmt.Errorf("invalid fee payer public key")
	}
	return nil
}

// CloseTokenAccount closes the specified token account.
func CloseTokenAccount(params CloseTokenAccountParams) InstructionFunc {
	return func(ctx context.Context, c SolanaClient) ([]types.Instruction, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate params: %w", err)
		}

		var (
			ownerPubKey = common.PublicKeyFromString(params.Owner)
			ata         common.PublicKey
			feePayer    common.PublicKey = ownerPubKey
		)

		if params.FeePayer != nil {
			feePayer = common.PublicKeyFromString(*params.FeePayer)
		}

		if params.CloseTokenAccount == nil && params.Mint != nil {
			mintPubKey := common.PublicKeyFromString(*params.Mint)
			var err error
			ata, _, err = common.FindAssociatedTokenAddress(ownerPubKey, mintPubKey)
			if err != nil {
				return nil, fmt.Errorf("failed to find associated token address: %w", err)
			}
		}

		return []types.Instruction{
			token.CloseAccount(token.CloseAccountParam{
				Account: ata,
				Auth:    ownerPubKey,
				To:      feePayer,
			}),
		}, nil
	}
}
