package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/easypmnt/checkout-api/jupiter"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/portto/solana-go-sdk/types"
)

type (
	// PaymentBuilder is a builder for creating a payment transaction.
	PaymentBuilder struct {
		sol    solanaClient
		jup    jupiterClient
		config Config
		tx     *Transaction

		availableBonusAmount uint64
		referenceAccount     types.Account
		bonusAuthAccount     *types.Account
	}
)

// NewPaymentTransactionBuilder creates a new PaymentTransactionBuilder.
func NewPaymentTransactionBuilder(sc solanaClient, jc jupiterClient, config Config) *PaymentBuilder {
	b := &PaymentBuilder{
		sol:              sc,
		jup:              jc,
		config:           config,
		referenceAccount: types.NewAccount(),
	}

	if b.config.ApplyBonus && b.config.BonusMintAddress == "" {
		panic("bonus mint address is required")
	}
	if b.config.AccrueBonus && b.config.BonusAuthAccount == "" {
		panic("bonus auth account is required")
	}
	if b.config.AccrueBonusRate == 0 {
		b.config.AccrueBonusRate = 100
	}

	mintAuth, err := types.AccountFromBase58(config.BonusAuthAccount)
	if err != nil {
		panic(fmt.Errorf("failed to parse bonus auth account: %w", err))
	}
	b.bonusAuthAccount = &mintAuth

	return b
}

// SetTransaction sets the transaction.
func (b *PaymentBuilder) SetTransaction(tx *Transaction, p *Payment) *PaymentBuilder {
	tx.DestinationWallet = p.DestinationWallet
	tx.DestinationMint = p.DestinationMint
	tx.Reference = b.referenceAccount.PublicKey.ToBase58()
	tx.Amount = p.Amount
	tx.Message = p.Message
	tx.Memo = p.ExternalID
	tx.DestinationMint = MintAddress(tx.DestinationMint, b.config.DestinationMint)
	tx.SourceMint = MintAddress(tx.SourceMint, tx.DestinationMint)
	if tx.DestinationWallet == "" {
		tx.DestinationWallet = b.config.DestinationWallet
	}
	if tx.TotalAmount == 0 {
		tx.TotalAmount = tx.Amount - tx.DiscountAmount
	}
	b.tx = tx
	return b
}

// GetReferenceAddress returns the reference address.
func (b *PaymentBuilder) GetReferenceAddress() string {
	return b.referenceAccount.PublicKey.ToBase58()
}

// Build builds the payment transaction.
func (b *PaymentBuilder) Build(ctx context.Context) (string, *Transaction, error) {
	if err := b.validate(); err != nil {
		return "", nil, fmt.Errorf("failed to validate builder parameters: %w", err)
	}

	bonusBalance, _ := b.sol.GetTokenBalance(ctx, b.tx.SourceWallet, b.config.BonusMintAddress)
	b.availableBonusAmount = bonusBalance.Amount
	b.tx = b.recalculateTotalAmount(b.tx)

	builder := solana.NewTransactionBuilder(b.sol).SetFeePayer(b.tx.SourceWallet)
	builder = b.burnBonus(builder)
	builder, err := b.swap(builder)
	if err != nil {
		return "", nil, err
	}
	if IsSOL(b.tx.DestinationMint) {
		builder = b.transferSOL(builder)
	} else {
		builder = b.transferToken(builder)
	}
	builder = b.mintBonus(builder)
	base64Tx, err := builder.Build(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	return base64Tx, b.tx, nil
}

// validate builder parameters.
func (b *PaymentBuilder) validate() error {
	if b.tx.SourceWallet == "" {
		return errors.New("source wallet address is required")
	}
	if b.tx.SourceMint == "" {
		return errors.New("source mint address is required")
	}
	if b.tx.DestinationWallet == "" {
		return errors.New("destination wallet address is required")
	}
	if b.tx.DestinationMint == "" {
		return errors.New("destination mint address is required")
	}
	if b.tx.Amount == 0 && b.tx.TotalAmount == 0 && b.tx.DiscountAmount == 0 {
		return errors.New("amount is required")
	}
	if b.tx.ApplyBonus && b.config.BonusMintAddress == "" {
		return errors.New("bonus mint address is required")
	}
	if b.config.AccrueBonus && b.bonusAuthAccount == nil {
		return errors.New("bonus auth account is required")
	}
	if b.config.AccrueBonus && b.config.AccrueBonusRate == 0 {
		return errors.New("accrue bonus rate is required")
	}
	return nil
}

func (b *PaymentBuilder) recalculateTotalAmount(tx *Transaction) *Transaction {
	if tx.Amount == 0 && tx.TotalAmount > 0 {
		tx.Amount = tx.TotalAmount + tx.DiscountAmount
		return tx
	}

	if tx.DiscountAmount > 0 && tx.Amount > 0 {
		tx.TotalAmount = tx.Amount - tx.DiscountAmount
		if tx.TotalAmount < 0 {
			tx.TotalAmount = 0
		}
		return tx
	}

	if b.tx.ApplyBonus && b.availableBonusAmount > 0 {
		maxCanBeApplyAmount := b.availableBonusAmount
		if maxCanBeApplyAmount > tx.Amount {
			maxCanBeApplyAmount = tx.Amount
		}
		if b.config.MaxApplyBonusAmount > 0 && b.config.MaxApplyBonusAmount < maxCanBeApplyAmount {
			maxCanBeApplyAmount = b.config.MaxApplyBonusAmount
		}
		if b.config.MaxApplyBonusPercent > 0 {
			bonusPercAmount := tx.Amount * uint64(b.config.MaxApplyBonusPercent) / 10000
			if bonusPercAmount < maxCanBeApplyAmount {
				maxCanBeApplyAmount = bonusPercAmount
			}
		}
		tx.DiscountAmount = maxCanBeApplyAmount
		tx.TotalAmount = tx.TotalAmount - tx.DiscountAmount
		if tx.TotalAmount < 0 {
			tx.TotalAmount = 0
		}
	}

	return tx
}

func (b *PaymentBuilder) burnBonus(builder *solana.TransactionBuilder) *solana.TransactionBuilder {
	if !b.tx.ApplyBonus || b.tx.DiscountAmount == 0 {
		return builder
	}

	return builder.AddInstruction(solana.BurnToken(solana.BurnTokenParams{
		Mint:              b.config.BonusMintAddress,
		TokenAccountOwner: b.tx.SourceWallet,
		Amount:            b.tx.DiscountAmount,
	}))
}

func (b *PaymentBuilder) mintBonus(builder *solana.TransactionBuilder) *solana.TransactionBuilder {
	if !b.config.AccrueBonus {
		return builder
	}

	bonusAmount := b.tx.TotalAmount * b.config.AccrueBonusRate / 10000
	if bonusAmount == 0 {
		return builder
	}

	b.tx.AccruedBonusAmount = bonusAmount

	return builder.AddInstruction(solana.MintFungibleToken(solana.MintFungibleTokenParams{
		Funder:    b.tx.SourceWallet,
		Mint:      b.config.BonusMintAddress,
		MintOwner: b.bonusAuthAccount.PublicKey.ToBase58(),
		MintTo:    b.tx.SourceWallet,
		Amount:    bonusAmount,
	})).AddSigner(*b.bonusAuthAccount)
}

func (b *PaymentBuilder) transferToken(builder *solana.TransactionBuilder) *solana.TransactionBuilder {
	return builder.AddInstruction(solana.TransferToken(solana.TransferTokenParam{
		Sender:    b.tx.SourceWallet,
		Recipient: b.tx.DestinationWallet,
		Mint:      b.tx.DestinationMint,
		Reference: b.tx.Reference,
		Amount:    b.tx.TotalAmount,
	}))
}

func (b *PaymentBuilder) transferSOL(builder *solana.TransactionBuilder) *solana.TransactionBuilder {
	return builder.AddInstruction(solana.TransferSOL(solana.TransferSOLParams{
		Sender:    b.tx.SourceWallet,
		Recipient: b.tx.DestinationWallet,
		Reference: b.tx.Reference,
		Amount:    b.tx.TotalAmount,
	}))
}

func (b *PaymentBuilder) swap(builder *solana.TransactionBuilder) (*solana.TransactionBuilder, error) {
	if b.tx.SourceMint == b.tx.DestinationMint {
		return builder, nil
	}

	jupTx, err := b.jup.BestSwap(jupiter.BestSwapParams{
		UserPublicKey: b.tx.SourceWallet,
		InputMint:     b.tx.SourceMint,
		OutputMint:    b.tx.DestinationMint,
		Amount:        b.tx.TotalAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get best swap transaction: %w", err)
	}

	jtx, err := solana.DecodeTransaction(jupTx)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jupiter transaction: %w", err)
	}

	return builder.AddRawInstructionsToBeginning(jtx.Message.DecompileInstructions()...), nil
}
