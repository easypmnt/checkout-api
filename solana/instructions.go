package solana

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/program/associated_token_account"
	"github.com/portto/solana-go-sdk/program/memo"
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
	return func(ctx context.Context, _ SolanaClient) ([]types.Instruction, error) {
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

		return []types.Instruction{instruction}, nil
	}
}
