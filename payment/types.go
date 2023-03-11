package payment

import (
	"time"

	"github.com/easypmnt/checkout-api/repository"
	"github.com/google/uuid"
)

// Predefined statuses of the payment.
const (
	StatusNew       = "new"       // New payment. No transactions yet.
	StatusPending   = "pending"   // Payment is in progress. Some transactions are created but not confirmed yet.
	StatusConfirmed = "confirmed" // Payment is confirmed. Transaction is confirmed on the blockchain.
	StatusFailed    = "failed"    // Payment is failed. Transaction is failed on the blockchain or not confirmed after a long time.
	StatusCanceled  = "canceled"  // Payment is canceled by the user.
)

// Default currencies.
var defaultCurrencies = map[string]string{
	"USDC": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
	"USDT": "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
	"SOL":  "So11111111111111111111111111111111111111112",
}

type (
	// Payment represents a payment entity in the system.
	Payment struct {
		ID          uuid.UUID `json:"id"`
		ExternalID  string    `json:"external_id,omitempty"` // ExternalID is an ID of the payment in the external system, e.g. order ID.
		Currency    string    `json:"currency,omitempty"`    // Custom currency of the payment. If not set, the default currency will be used.
		TotalAmount uint64    `json:"total_amount"`          // TotalAmount is the total amount of the payment in the smallest unit of the currency.
		Status      string    `json:"status"`                // Status is the status of the payment.
		Message     string    `json:"message,omitempty"`     // Message is a note to show in wallet.
		Memo        string    `json:"memo,omitempty"`        // Memo is a note that is attached to the blockchain transaction.
		CreatedAt   string    `json:"created_at"`
		UpdatedAt   string    `json:"updated_at"`
		ExpiresAt   string    `json:"expires_at,omitempty"` // ExpiresAt is the time when the payment will expire.

		Destinations []Destination `json:"destination,omitempty"`  // Payment destinations.
		Transactions []Transaction `json:"transactions,omitempty"` // Payment blockchain transactions.
	}

	// Destination represents a destination entity in the payment.
	Destination struct {
		WalletAddress  string `json:"wallet_address"`            // WalletAddress is the address of the wallet to send the payment to.
		Amount         uint64 `json:"amount"`                    // Amount is the amount of the payment in the smallest unit of the currency.
		TotalAmount    uint64 `json:"total_amount,omitempty"`    // TotalAmount is the total amount of the payment in the smallest unit of the currency.
		DiscountAmount uint64 `json:"discount_amount,omitempty"` // DiscountAmount is the discount amount of the payment in the smallest unit of the currency.
	}

	// Transaction represents a transaction entity in the system.
	Transaction struct {
		ID             uuid.UUID `json:"id"`
		PaymentID      uuid.UUID `json:"payment_id"`
		Reference      string    `json:"reference"`
		TxSignature    string    `json:"tx_signature"`
		Amount         uint64    `json:"amount"`
		DiscountAmount uint64    `json:"discount_amount,omitempty"`
		Status         string    `json:"status"`
		CreatedAt      string    `json:"created_at"`
		UpdatedAt      string    `json:"updated_at"`
	}
)

// Cast repository.PaymentInfo to payment.Payment.
func CastToPayment(info *repository.PaymentInfo) *Payment {
	result := &Payment{
		ID:          info.Payment.ID,
		ExternalID:  info.Payment.ExternalID.String,
		Currency:    info.Payment.Currency,
		TotalAmount: uint64(info.Payment.TotalAmount),
		Status:      string(info.Payment.Status),
		Message:     info.Payment.Message.String,
		Memo:        info.Payment.Memo.String,
		CreatedAt:   info.Payment.CreatedAt.Format(time.RFC3339),
	}

	if info.Payment.UpdatedAt.Valid {
		result.UpdatedAt = info.Payment.UpdatedAt.Time.Format(time.RFC3339)
	}
	if info.Payment.ExpiresAt.Valid {
		result.ExpiresAt = info.Payment.ExpiresAt.Time.Format(time.RFC3339)
	}

	destinations := make([]Destination, 0, len(info.Destinations))
	for _, dest := range info.Destinations {
		destinations = append(destinations, Destination{
			WalletAddress: dest.Destination,
			Amount:        uint64(dest.Amount.Int64),
		})
	}

	transactions := make([]Transaction, 0, len(info.Transactions))
	for _, tx := range info.Transactions {
		t := Transaction{
			ID:             tx.ID,
			PaymentID:      tx.PaymentID,
			Reference:      tx.Reference,
			TxSignature:    tx.TxSignature.String,
			Amount:         uint64(tx.Amount),
			DiscountAmount: uint64(tx.DiscountAmount),
			Status:         string(tx.Status),
			CreatedAt:      tx.CreatedAt.Format(time.RFC3339),
		}
		if tx.UpdatedAt.Valid {
			t.UpdatedAt = tx.UpdatedAt.Time.Format(time.RFC3339)
		}
		transactions = append(transactions, t)
	}

	result.Destinations = destinations
	result.Transactions = transactions

	return result
}
