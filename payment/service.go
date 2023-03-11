package payment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/easypmnt/checkout-api/jupiter"
	"github.com/easypmnt/checkout-api/repository"
	"github.com/easypmnt/checkout-api/solana"
	"github.com/easypmnt/checkout-api/utils"
	"github.com/google/uuid"
	"github.com/portto/solana-go-sdk/types"
)

type (
	// Service is the interface that wraps the basic payment methods.
	Service struct {
		repo      paymentRepository
		jupClient jupiterClient
		solClient solanaClient

		// defaultMerchantSettings is the default merchant settings
		// that will be used, if not set while payment creation.
		defaultMerchantSettings MerchantSettings

		// the URI to use for QR code payments.
		solanaPayBaseURI string
	}

	// ServiceOption is the type for service options that can be passed to NewService function.
	ServiceOption func(*Service)

	// MerchantSettings represents default merchant settings.
	MerchantSettings struct {
		WalletAddress string // WalletAddress is the base58 encoded public key of the wallet to send the payment to.
		ApplyBonus    bool   // ApplyBonus is a flag that indicates whether customer can apply bonus to the payment or not.
		MaxBonus      uint64 // MaxBonus is the maximum amount of bonus that can be applied to the payment.
		MaxBonusPerc  uint16 // MaxBonusPerc is the maximum percentage of bonus that can be applied to the payment.
		BonusRate     uint64 // BonusRate is the bonus rate that will be accrued to the payer's bonus account.
		BonusMintAddr string // BonusMintAddr is the base58 encoded public key of the mint address of the bonus token.
		BonusMintAuth string // BonusMintAuth is the base58 encoded public key of the mint authority of the bonus token.
	}

	paymentRepository interface {
		CreatePaymentWithDestinations(ctx context.Context, arg repository.CreatePaymentWithDestinationsParams) (repository.PaymentInfo, error)
		CreateTransactionWithCallback(ctx context.Context, arg repository.CreateTransactionWithCallbackParams) (repository.Transaction, error)
		UpdateTransaction(ctx context.Context, arg repository.UpdateTransactionParams) (repository.Transaction, error)
		GetPayment(ctx context.Context, paymentID uuid.UUID) (repository.Payment, error)
		GetPaymentInfo(ctx context.Context, paymentID uuid.UUID) (repository.PaymentInfo, error)
		GetPaymentInfoByExternalID(ctx context.Context, externalID string) (repository.PaymentInfo, error)
		UpdatePaymentStatus(ctx context.Context, arg repository.UpdatePaymentStatusParams) (repository.Payment, error)
		UpdatePaymentDestinations(ctx context.Context, arg repository.UpdatePaymentDestinationsParams) error
	}

	solanaClient interface {
		GetSOLBalance(ctx context.Context, base58Addr string) (solana.Balance, error)
		GetTokenBalance(ctx context.Context, base58Addr, base58MintAddr string) (solana.Balance, error)
		GetTokenSupply(ctx context.Context, base58MintAddr string) (solana.Balance, error)
		GetLatestBlockhash(ctx context.Context) (string, error)
		DoesTokenAccountExist(ctx context.Context, base58AtaAddr string) (bool, error)
		GetMinimumBalanceForRentExemption(ctx context.Context, size uint64) (uint64, error)
	}

	jupiterClient interface {
		BestSwap(params jupiter.BestSwapParams) (string, error)
	}
)

// NewService creates a new payment service.
func NewService(repo paymentRepository, sol solanaClient, jup jupiterClient, opts ...ServiceOption) *Service {
	s := &Service{repo: repo, solClient: sol, jupClient: jup}
	for _, opt := range opts {
		opt(s)
	}
	if s.defaultMerchantSettings.ApplyBonus {
		if (s.defaultMerchantSettings.MaxBonus == 0 && s.defaultMerchantSettings.MaxBonusPerc == 0) || s.defaultMerchantSettings.BonusMintAddr == "" || s.defaultMerchantSettings.BonusMintAuth == "" {
			s.defaultMerchantSettings.ApplyBonus = false
		}
	}
	return s
}

type (
	// CreatePaymentParams is the input for creating a new payment.
	CreatePaymentParams struct {
		ExternalID   string                    // ExternalID is the external payment id. It is optional.
		Currency     string                    // Currency is the payment currency. Example: SOL, USDC, or any SPL token mint address.
		Amount       int64                     // Amount is the total payment amount.
		Message      string                    // Message to show to the customer. It is optional.
		Memo         string                    // Memo is the memo to attach to the blockchain transaction. It is optional.
		TTL          int64                     // TTL is the time to live in seconds for the payment. It is optional.
		Destinations []CreateDestinationParams // Destinations is the list of payment destinations. Can be used to split the payment amount between multiple wallets.
	}

	CreateDestinationParams struct {
		Amount          int64  // Amount is the destination amount. You can use either amount or percentage, but not both.
		Percentage      int16  // Percentage is the destination percentage. You can use either amount or percentage, but not both.
		WalletAddress   string // WalletAddress is the base58 encoded public key of the wallet to send the payment to.
		ApplyBonus      bool   // ApplyBonus is a flag that indicates whether customer can apply bonus to the payment or not.
		MaxBonusAmount  int64  // MaxBonusAmount is the maximum amount of bonus that can be applied to the payment.
		MaxBonusPercent int16  // MaxBonusPercent is the maximum percentage of bonus that can be applied to the payment.
	}
)

// CreatePayment creates a new payment with the given params.
// It returns the created payment id and an error if any.
// TODO: refactor this function, it is too long.
func (s *Service) CreatePayment(ctx context.Context, arg CreatePaymentParams) (uuid.UUID, error) {
	if arg.Currency == "" {
		return uuid.Nil, fmt.Errorf("currency is required")
	}
	if tokenMint, ok := defaultCurrencies[arg.Currency]; ok {
		arg.Currency = tokenMint
	}

	paymentParams := repository.CreatePaymentParams{
		ExternalID:  sql.NullString{String: arg.ExternalID, Valid: arg.ExternalID != ""},
		Currency:    arg.Currency,
		TotalAmount: arg.Amount,
		Status:      repository.PaymentStatusNew,
		Message:     sql.NullString{String: arg.Message, Valid: arg.Message != ""},
		Memo:        sql.NullString{String: arg.Memo, Valid: arg.Memo != ""},
		ExpiresAt:   sql.NullTime{Time: time.Now().Add(time.Duration(arg.TTL) * time.Second), Valid: arg.TTL > 0},
	}

	var (
		totalAmount   int64
		totalPercent  int16
		usePercentage *bool
		destParams    = make([]repository.CreatePaymentDestinationParams, 0, len(arg.Destinations))
	)

	if len(arg.Destinations) > 0 {
		for _, dest := range arg.Destinations {
			if dest.Amount > 0 && dest.Percentage > 0 {
				return uuid.Nil, fmt.Errorf("amount and percentage can't be set at the same time")
			}
			if dest.Amount <= 0 && dest.Percentage <= 0 {
				return uuid.Nil, fmt.Errorf("amount or percentage should be set")
			}
			if usePercentage == nil {
				if dest.Amount <= 0 && dest.Percentage > 0 {
					usePercentage = utils.Pointer(true)
				} else {
					usePercentage = utils.Pointer(false)
				}
			} else {
				if (*usePercentage && dest.Amount > 0) || (!*usePercentage && dest.Percentage > 0) {
					return uuid.Nil, fmt.Errorf("can't mix percentage and amount, use only one of them for all destinations")
				}
			}

			totalAmount += dest.Amount
			totalPercent += dest.Percentage
			destParams = append(destParams, repository.CreatePaymentDestinationParams{
				Amount:             sql.NullInt64{Int64: dest.Amount, Valid: !*usePercentage},
				Percentage:         sql.NullInt16{Int16: dest.Percentage, Valid: *usePercentage},
				ApplyBonus:         dest.ApplyBonus,
				MaxBonusAmount:     dest.MaxBonusAmount,
				MaxBonusPercentage: dest.MaxBonusPercent,
			})
		}
	} else {
		// Use default destination, if no destinations provided.
		if arg.Amount <= 0 {
			return uuid.Nil, fmt.Errorf("amount should be greater than 0")
		}
		totalAmount = arg.Amount
		destParams = append(destParams, repository.CreatePaymentDestinationParams{
			Destination:        s.defaultMerchantSettings.WalletAddress,
			Amount:             sql.NullInt64{Int64: arg.Amount, Valid: arg.Amount > 0},
			Percentage:         sql.NullInt16{Int16: 10000, Valid: true},
			ApplyBonus:         s.defaultMerchantSettings.ApplyBonus,
			MaxBonusAmount:     int64(s.defaultMerchantSettings.MaxBonus),
			MaxBonusPercentage: int16(s.defaultMerchantSettings.MaxBonusPerc),
		})
	}

	if paymentParams.TotalAmount <= 0 && *usePercentage {
		return uuid.Nil, fmt.Errorf("total amount should be greater than 0 if percentage is used")
	}
	if totalPercent > 0 && totalPercent != 10000 {
		return uuid.Nil, fmt.Errorf("total percentage across all destinations should be equal to 10000")
	}
	if totalAmount > 0 {
		paymentParams.TotalAmount = totalAmount
	}

	// recalculate and sync percentage vs amount
	if *usePercentage {
		recTotalamount := int64(0)
		for i := range destParams {
			amount := int64(totalAmount * int64(destParams[i].Percentage.Int16) / 10000)
			recTotalamount += amount
			destParams[i].Amount = sql.NullInt64{
				Int64: int64(totalAmount * int64(destParams[i].Percentage.Int16) / 10000),
				Valid: true,
			}
		}
		if recTotalamount != paymentParams.TotalAmount {
			return uuid.Nil, fmt.Errorf("total amount should be equal to sum of all destinations")
		}
	} else {
		recTotalPercent := int16(0)
		for i := range destParams {
			percent := int16(destParams[i].Amount.Int64 * 10000 / paymentParams.TotalAmount)
			recTotalPercent += percent
			destParams[i].Percentage = sql.NullInt16{
				Int16: int16(destParams[i].Amount.Int64 * 10000 / paymentParams.TotalAmount),
				Valid: true,
			}
		}
		if recTotalPercent != 10000 {
			return uuid.Nil, fmt.Errorf("total percentage should be equal to 10000")
		}
	}

	payment, err := s.repo.CreatePaymentWithDestinations(ctx, repository.CreatePaymentWithDestinationsParams{
		Payment:      paymentParams,
		Destinations: destParams,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return payment.Payment.ID, nil
}

// CancelPayment cancels the payment with the given id.
// It returns an error if any.
func (s *Service) CancelPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.Status != repository.PaymentStatusNew {
		return fmt.Errorf("payment status is not new")
	}

	if _, err = s.repo.UpdatePaymentStatus(ctx, repository.UpdatePaymentStatusParams{
		ID:     paymentID,
		Status: repository.PaymentStatusCanceled,
	}); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// GetPaymentInfo returns the payment info with the given id.
// It returns an error if any.
func (s *Service) GetPaymentInfo(ctx context.Context, paymentID uuid.UUID) (*Payment, error) {
	paymentInfo, err := s.repo.GetPaymentInfo(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment info: %w", err)
	}

	return CastToPayment(&paymentInfo), nil
}

// GetPaymentInfoByExternalID returns the payment info with the given external id.
// It returns an error if any.
func (s *Service) GetPaymentInfoByExternalID(ctx context.Context, externalID string) (*Payment, error) {
	paymentInfo, err := s.repo.GetPaymentInfoByExternalID(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment info: %w", err)
	}

	return CastToPayment(&paymentInfo), nil
}

// GeneratePaymentLink generates a payment link for the given payment id to be used in the QR code.
// It returns the generated link and an error if any.
func (s *Service) GeneratePaymentLink(ctx context.Context, paymentID uuid.UUID) (string, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return "", fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.ExpiresAt.Valid && payment.ExpiresAt.Time.Before(time.Now()) {
		return "", fmt.Errorf("payment is expired")
	}
	if payment.Status != repository.PaymentStatusNew && payment.Status != repository.PaymentStatusFailed {
		return "", fmt.Errorf("payment status is not new")
	}

	return fmt.Sprintf("solana:%s/%s", s.solanaPayBaseURI, paymentID), nil
}

// GeneratePaymentTransactionParams contains the params for generating a payment transaction.
type GeneratePaymentTransactionParams struct {
	PaymentID  uuid.UUID // required; payment id
	Base58Addr string    // required; base58 encoded customer wallet address
	Currency   string    // optional; currency of the payment, if provided, it will be converted to the currency of the merchant
	ApplyBonus bool      // optional; whether to apply bonus to the payment, if it exists on customer wallet. Default is false
}

// GeneratePaymentTransaction generates a payment transaction for the given payment id.
// Returns base64 encoded transaction and an error if any.
// TODO: refactor this function, it's too long.
func (s *Service) GeneratePaymentTransaction(ctx context.Context, arg GeneratePaymentTransactionParams) (string, error) {
	payment, err := s.repo.GetPaymentInfo(ctx, arg.PaymentID)
	if err != nil {
		return "", fmt.Errorf("failed to get payment: %w", err)
	}
	if payment.Payment.ExpiresAt.Valid && payment.Payment.ExpiresAt.Time.Before(time.Now()) {
		return "", fmt.Errorf("payment is expired")
	}
	if payment.Payment.Status != repository.PaymentStatusNew && payment.Payment.Status != repository.PaymentStatusFailed {
		return "", fmt.Errorf("payment status is not new")
	}

	var bonusAmount int64
	if arg.ApplyBonus && s.defaultMerchantSettings.ApplyBonus {
		// Check if customer has bonus balance.
		bonusBalance, _ := s.solClient.GetTokenBalance(ctx, arg.Base58Addr, s.defaultMerchantSettings.BonusMintAddr)
		payment, bonusAmount, err = s.recalculatePaymentWithBonus(ctx, payment, bonusBalance)
		if err != nil {
			return "", fmt.Errorf("failed to recalculate payment with bonus: %w", err)
		}
	}

	if arg.Currency == payment.Payment.Currency {
		// Check if customer has enough balance.
		if err := s.checkBalance(
			ctx,
			arg.Base58Addr,
			arg.Currency,
			uint64(payment.Payment.TotalAmount-bonusAmount),
		); err != nil {
			return "", err
		}
	}

	txBuilder := solana.NewTransactionBuilder(s.solClient).SetFeePayer(arg.Base58Addr)

	if arg.Currency != payment.Payment.Currency {
		// Convert payment amount to the currency of the merchant.
		jupTx, err := s.jupClient.BestSwap(jupiter.BestSwapParams{
			UserPublicKey: arg.Base58Addr,
			InputMint:     arg.Currency,
			OutputMint:    payment.Payment.Currency,
			Amount:        uint64(payment.Payment.TotalAmount - bonusAmount),
		})
		if err != nil {
			return "", fmt.Errorf("failed to get best swap transaction: %w", err)
		}
		jtx, err := solana.DecodeTransaction(jupTx)
		if err != nil {
			return "", fmt.Errorf("failed to decode jupiter transaction: %w", err)
		}
		txBuilder = txBuilder.AddRawInstructionsToBeginning(jtx.Message.DecompileInstructions()...)
	}

	referenceAcc := types.NewAccount()

	// Transfer payment amount to the merchants.
	if payment.Payment.Currency == "SOL" || payment.Payment.Currency == defaultCurrencies["SOL"] {
		for _, dest := range payment.Destinations {
			txBuilder = txBuilder.AddInstruction(solana.TransferSOL(solana.TransferSOLParams{
				Sender:    arg.Base58Addr,
				Recipient: dest.Destination,
				Reference: referenceAcc.PublicKey.ToBase58(),
				Amount:    uint64(dest.TotalAmount),
			}))
		}
	} else {
		for _, dest := range payment.Destinations {
			txBuilder = txBuilder.AddInstruction(solana.TransferToken(solana.TransferTokenParam{
				Sender:    arg.Base58Addr,
				Recipient: dest.Destination,
				Mint:      payment.Payment.Currency,
				Reference: referenceAcc.PublicKey.ToBase58(),
				Amount:    uint64(dest.TotalAmount),
			}))
		}
	}

	// Mint bonus to the customer.
	if s.defaultMerchantSettings.ApplyBonus {
		amount := (payment.Payment.TotalAmount - bonusAmount) / int64(s.defaultMerchantSettings.BonusRate)
		if amount > 0 {
			authAcc, err := types.AccountFromBase58(s.defaultMerchantSettings.BonusMintAuth)
			if err != nil {
				return "", fmt.Errorf("failed to decode bonus mint auth account: %w", err)
			}
			txBuilder = txBuilder.AddInstruction(solana.MintFungibleToken(solana.MintFungibleTokenParams{
				Funder:    arg.Base58Addr,
				Mint:      s.defaultMerchantSettings.BonusMintAddr,
				MintOwner: authAcc.PublicKey.ToBase58(),
				MintTo:    arg.Base58Addr,
				Amount:    uint64(amount),
			})).AddSigner(authAcc)
		}
	}

	base64Tx, err := txBuilder.Build(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// Create transaction in the database.
	if _, err := s.repo.CreateTransactionWithCallback(ctx, repository.CreateTransactionWithCallbackParams{
		Transaction: repository.CreateTransactionParams{
			PaymentID:      arg.PaymentID,
			Reference:      referenceAcc.PublicKey.ToBase58(),
			Amount:         payment.Payment.TotalAmount - bonusAmount,
			DiscountAmount: bonusAmount,
			Status:         repository.TransactionStatusPending,
		},
		Destinations: func(destinations []repository.PaymentDestination) []repository.CreatePaymentDestinationParams {
			result := make([]repository.CreatePaymentDestinationParams, 0, len(destinations))
			for _, dest := range destinations {
				result = append(result, repository.CreatePaymentDestinationParams{
					PaymentID:          payment.Payment.ID,
					Destination:        dest.Destination,
					Amount:             dest.Amount,
					Percentage:         dest.Percentage,
					TotalAmount:        dest.TotalAmount,
					DiscountAmount:     dest.DiscountAmount,
					ApplyBonus:         dest.ApplyBonus,
					MaxBonusAmount:     dest.MaxBonusAmount,
					MaxBonusPercentage: dest.MaxBonusPercentage,
				})
			}
			return result
		}(payment.Destinations),
	}); err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	return base64Tx, nil
}

// Check if customer has enough balance.
func (s *Service) checkBalance(ctx context.Context, base58Addr, currency string, amount uint64) error {
	if currency == "SOL" || currency == defaultCurrencies["SOL"] {
		customerBalance, err := s.solClient.GetSOLBalance(ctx, base58Addr)
		if err != nil {
			return fmt.Errorf("failed to get customer SOL balance: %w", err)
		}
		if customerBalance.Amount <= amount {
			return fmt.Errorf("insufficient SOL balance for transaction")
		}
	} else {
		customerBalance, err := s.solClient.GetTokenBalance(ctx, base58Addr, currency)
		if err != nil {
			return fmt.Errorf("failed to get customer token balance: %w", err)
		}
		if customerBalance.Amount <= amount {
			return fmt.Errorf("insufficient token balance for transaction")
		}
	}

	return nil
}

// recalculatePaymentWithBonus recalculates the payment amount with the given bonus.
// It returns an error if any.
func (s *Service) recalculatePaymentWithBonus(ctx context.Context, payment repository.PaymentInfo, bonus solana.Balance) (repository.PaymentInfo, int64, error) {
	if len(payment.Destinations) == 0 || s.defaultMerchantSettings.BonusMintAddr == "" {
		return repository.PaymentInfo{}, 0, fmt.Errorf("no payment destinations found")
	}

	availableDiscountAmount := int64(bonus.Amount)
	if availableDiscountAmount <= 0 {
		return payment, 0, nil
	}
	if availableDiscountAmount > payment.Payment.TotalAmount {
		availableDiscountAmount = payment.Payment.TotalAmount
	}

	totalBonusAmount := int64(0)
	for i := range payment.Destinations {
		if payment.Destinations[i].ApplyBonus {
			bonusAmount := calcBonusAmount(availableDiscountAmount, payment.Destinations[i])
			if bonusAmount > 0 {
				payment.Destinations[i].DiscountAmount = bonusAmount
				payment.Destinations[i].TotalAmount = payment.Destinations[i].Amount.Int64 - bonusAmount
				totalBonusAmount += bonusAmount
			}
		}
	}

	return payment, totalBonusAmount, nil
}

func calcBonusAmount(availableBonus int64, dest repository.PaymentDestination) int64 {
	if availableBonus == 0 || !dest.ApplyBonus ||
		(dest.MaxBonusAmount <= 0 && dest.MaxBonusPercentage <= 0) {
		return 0
	}
	if dest.MaxBonusPercentage > 10000 {
		dest.MaxBonusPercentage = 10000
	}

	if dest.MaxBonusAmount <= 0 && dest.MaxBonusPercentage > 0 {
		dest.MaxBonusAmount = int64(dest.Amount.Int64 * int64(dest.MaxBonusPercentage) / 10000)
	}
	if dest.MaxBonusAmount > dest.Amount.Int64 {
		dest.MaxBonusAmount = dest.Amount.Int64
	}

	bonusAmount := int64(0)
	if dest.MaxBonusAmount > 0 {
		if availableBonus > dest.MaxBonusAmount {
			bonusAmount = dest.MaxBonusAmount
		} else {
			bonusAmount = availableBonus
		}
	}

	if dest.Percentage.Int16 > 0 && dest.Percentage.Int16 < 10000 {
		// get bonus percentage from available bonus amount
		bonusPercentage := bonusAmount * 10000 / availableBonus
		if bonusPercentage > int64(dest.Percentage.Int16) {
			bonusPercentage = int64(dest.Percentage.Int16)
			// recalculate bonus amount from bonus percentage
			bonusAmount = availableBonus * bonusPercentage / 10000
		}
	}

	return bonusAmount
}
