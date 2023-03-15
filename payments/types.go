package payments

import (
	"context"
	"time"

	"github.com/easypmnt/checkout-api/jupiter"
	"github.com/easypmnt/checkout-api/repository"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/google/uuid"
)

type (
	Config struct {
		ApplyBonus           bool
		BonusMintAddress     string
		BonusAuthAccount     string
		MaxApplyBonusAmount  uint64
		MaxApplyBonusPercent uint16 // 10000 = 100%, 100 = 1%, 1 = 0.01%
		AccrueBonus          bool
		AccrueBonusRate      uint64
		DestinationMint      string
		DestinationWallet    string
		PaymentTTL           time.Duration
		SolPayBaseURL        string
	}

	// solanaClient is an RPC client for Solana.
	solanaClient interface {
		GetLatestBlockhash(ctx context.Context) (string, error)
		DoesTokenAccountExist(ctx context.Context, base58AtaAddr string) (bool, error)
		GetMinimumBalanceForRentExemption(ctx context.Context, size uint64) (uint64, error)
		GetTokenBalance(ctx context.Context, base58Addr, base58MintAddr string) (solana.Balance, error)
	}

	// jupiterClient is an REST API client for Jupiter.
	jupiterClient interface {
		BestSwap(params jupiter.BestSwapParams) (string, error)
	}

	paymentRepository interface {
		CreatePayment(ctx context.Context, arg repository.CreatePaymentParams) (repository.Payment, error)
		GetPayment(ctx context.Context, id uuid.UUID) (repository.Payment, error)
		GetPaymentByExternalID(ctx context.Context, externalID string) (repository.Payment, error)
		MarkPaymentsExpired(ctx context.Context) error
		UpdatePaymentStatus(ctx context.Context, arg repository.UpdatePaymentStatusParams) (repository.Payment, error)

		CreateTransaction(ctx context.Context, arg repository.CreateTransactionParams) (repository.Transaction, error)
		GetTransactionByPaymentIDSourceWalletAndMint(ctx context.Context, arg repository.GetTransactionByPaymentIDSourceWalletAndMintParams) (repository.Transaction, error)
		GetTransactionByReference(ctx context.Context, reference string) (repository.Transaction, error)
		GetTransactionsByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]repository.Transaction, error)
		UpdateTransactionByReference(ctx context.Context, arg repository.UpdateTransactionByReferenceParams) (repository.Transaction, error)
		GetPendingTransactions(ctx context.Context) ([]repository.Transaction, error)
		MarkTransactionsAsExpired(ctx context.Context) error
	}
)
