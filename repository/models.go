// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0

package repository

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusNew       PaymentStatus = "new"
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusExpired   PaymentStatus = "expired"
)

func (e *PaymentStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = PaymentStatus(s)
	case string:
		*e = PaymentStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for PaymentStatus: %T", src)
	}
	return nil
}

type NullPaymentStatus struct {
	PaymentStatus PaymentStatus
	Valid         bool // Valid is true if PaymentStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullPaymentStatus) Scan(value interface{}) error {
	if value == nil {
		ns.PaymentStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.PaymentStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullPaymentStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.PaymentStatus, nil
}

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusExpired   TransactionStatus = "expired"
)

func (e *TransactionStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TransactionStatus(s)
	case string:
		*e = TransactionStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for TransactionStatus: %T", src)
	}
	return nil
}

type NullTransactionStatus struct {
	TransactionStatus TransactionStatus
	Valid             bool // Valid is true if TransactionStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullTransactionStatus) Scan(value interface{}) error {
	if value == nil {
		ns.TransactionStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.TransactionStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullTransactionStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.TransactionStatus, nil
}

type Payment struct {
	ID                uuid.UUID      `json:"id"`
	ExternalID        sql.NullString `json:"external_id"`
	DestinationWallet string         `json:"destination_wallet"`
	DestinationMint   string         `json:"destination_mint"`
	Amount            int64          `json:"amount"`
	Status            PaymentStatus  `json:"status"`
	Message           sql.NullString `json:"message"`
	ExpiresAt         sql.NullTime   `json:"expires_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         sql.NullTime   `json:"updated_at"`
}

type Token struct {
	TokenType        string       `json:"token_type"`
	Credential       string       `json:"credential"`
	AccessTokenID    uuid.UUID    `json:"access_token_id"`
	RefreshTokenID   uuid.UUID    `json:"refresh_token_id"`
	AccessExpiresAt  time.Time    `json:"access_expires_at"`
	RefreshExpiresAt time.Time    `json:"refresh_expires_at"`
	UpdatedAt        sql.NullTime `json:"updated_at"`
	CreatedAt        time.Time    `json:"created_at"`
}

type Transaction struct {
	ID                 uuid.UUID         `json:"id"`
	PaymentID          uuid.UUID         `json:"payment_id"`
	Reference          string            `json:"reference"`
	SourceWallet       string            `json:"source_wallet"`
	SourceMint         string            `json:"source_mint"`
	DestinationWallet  string            `json:"destination_wallet"`
	DestinationMint    string            `json:"destination_mint"`
	Amount             int64             `json:"amount"`
	DiscountAmount     int64             `json:"discount_amount"`
	TotalAmount        int64             `json:"total_amount"`
	AccruedBonusAmount int64             `json:"accrued_bonus_amount"`
	Message            sql.NullString    `json:"message"`
	Memo               sql.NullString    `json:"memo"`
	ApplyBonus         sql.NullBool      `json:"apply_bonus"`
	TxSignature        sql.NullString    `json:"tx_signature"`
	Status             TransactionStatus `json:"status"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          sql.NullTime      `json:"updated_at"`
}
