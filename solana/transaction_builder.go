package solana

import (
	"context"

	"github.com/easypmnt/checkout-api/internal/utils"
	"github.com/pkg/errors"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/types"
)

type (
	// TransactionBuilder is a builder for Transaction.
	TransactionBuilder struct {
		client                SolanaClient
		instructions          []InstructionFunc
		rawInstructionsBefore []types.Instruction
		rawInstructionsAfter  []types.Instruction
		signers               []types.Account
		feePayer              *common.PublicKey // transaction fee payer
		addressLookup         []types.AddressLookupTableAccount
	}
)

// NewTransactionBuilder creates a new TransactionBuilder instance.
func NewTransactionBuilder(client SolanaClient) *TransactionBuilder {
	return &TransactionBuilder{
		client:                client,
		instructions:          []InstructionFunc{},
		rawInstructionsBefore: []types.Instruction{},
		rawInstructionsAfter:  []types.Instruction{},
		signers:               []types.Account{},
		addressLookup:         []types.AddressLookupTableAccount{},
	}
}

// AddInstruction adds a new instruction to the transaction.
func (b *TransactionBuilder) AddInstruction(instruction InstructionFunc) *TransactionBuilder {
	b.instructions = append(b.instructions, instruction)
	return b
}

// AddRawInstructionsToBeginning adds raw instructions to the beginning of the transaction.
func (b *TransactionBuilder) AddRawInstructionsToBeginning(instructions ...types.Instruction) *TransactionBuilder {
	b.rawInstructionsBefore = append(b.rawInstructionsBefore, instructions...)
	return b
}

// AddRawInstructionsToEnd adds raw instructions to the end of the transaction.
func (b *TransactionBuilder) AddRawInstructionsToEnd(instructions ...types.Instruction) *TransactionBuilder {
	b.rawInstructionsAfter = append(b.rawInstructionsAfter, instructions...)
	return b
}

// SetFeePayer sets the fee payer for the transaction.
func (b *TransactionBuilder) SetFeePayer(feePayer string) *TransactionBuilder {
	b.feePayer = utils.Pointer(common.PublicKeyFromString(feePayer))
	return b
}

// AddSigner adds a signer account to the transaction.
// The signer account must be added before calling Build().
func (b *TransactionBuilder) AddSigner(signer types.Account) *TransactionBuilder {
	b.signers = append(b.signers, signer)
	return b
}

// SetAddressLookupTableAccount adds a new address lookup table account to the transaction.
func (b *TransactionBuilder) SetAddressLookupTableAccount(account types.AddressLookupTableAccount) *TransactionBuilder {
	b.addressLookup = append(b.addressLookup, account)
	return b
}

// Build builds a new transaction with the given instructions.
// It returns base64 encoded transaction or an error.
func (b *TransactionBuilder) Build(ctx context.Context) (string, error) {
	// Validate the builder inputs before building the transaction.
	if err := b.Validate(); err != nil {
		return "", errors.Wrap(err, "failed to build transaction: validate")
	}

	// Prepare the instructions.
	instructions, err := b.PrepareInstructions(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to build transaction: prepare instructions")
	}

	latestBlockhash, err := b.client.GetLatestBlockhash(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to build transaction: get latest blockhash")
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:                   *b.feePayer,
			RecentBlockhash:            latestBlockhash,
			Instructions:               instructions,
			AddressLookupTableAccounts: b.addressLookup,
		}),
		Signers: b.signers,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to build transaction: new transaction")
	}

	base64Tx, err := EncodeTransaction(tx)
	if err != nil {
		return "", errors.Wrap(err, "failed to build transaction: encode transaction")
	}

	return base64Tx, nil
}

// Validate validates the transaction builder.
func (b *TransactionBuilder) Validate() error {
	if b.client == nil {
		return ErrClientNotSet
	}
	if b.feePayer == nil || *b.feePayer == (common.PublicKey{}) {
		return ErrFeePayerNotSet
	}
	if len(b.instructions) == 0 {
		return ErrNoInstruction
	}
	return nil
}

// PrepareInstructions prepares the instructions for the transaction.
// It returns a list of prepared instructions or an error.
func (b *TransactionBuilder) PrepareInstructions(ctx context.Context) ([]types.Instruction, error) {
	instructions := []types.Instruction{}
	if len(b.rawInstructionsBefore) > 0 {
		instructions = append(instructions, b.rawInstructionsBefore...)
	}
	for _, instruction := range b.instructions {
		ins, err := instruction(ctx, b.client)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare instructions")
		}
		instructions = append(instructions, ins...)
	}
	if len(b.rawInstructionsAfter) > 0 {
		instructions = append(instructions, b.rawInstructionsAfter...)
	}
	return instructions, nil
}
