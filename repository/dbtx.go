package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type QueriesTx struct {
	*Queries
	dbConn *sql.DB
}

// NewWithConnection creates a new QueriesTx with a database connection to use for transactions.
// It is the caller's responsibility to close the connection.
func NewWithConnection(ctx context.Context, dbConn *sql.DB) (*QueriesTx, error) {
	q, err := Prepare(ctx, dbConn)
	if err != nil {
		return nil, err
	}

	return &QueriesTx{
		Queries: q,
		dbConn:  dbConn,
	}, nil
}

// CreatePaymentWithDestinationsParams is a struct for CreatePaymentWithDestinations method
type CreatePaymentWithDestinationsParams struct {
	Payment      CreatePaymentParams
	Destinations []CreatePaymentDestinationParams
}

// CreatePaymentWithDestinations creates a new payment with destinations
func (q *QueriesTx) CreatePaymentWithDestinations(ctx context.Context, arg CreatePaymentWithDestinationsParams) (Payment, []PaymentDestination, error) {
	tx, err := q.dbConn.Begin()
	if err != nil {
		return Payment{}, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repo := q.WithTx(tx)

	payment, err := repo.CreatePayment(ctx, arg.Payment)
	if err != nil {
		return Payment{}, nil, fmt.Errorf("failed to create payment: %w", err)
	}

	destinations := make([]PaymentDestination, 0, len(arg.Destinations))
	for _, dest := range arg.Destinations {
		dest.PaymentID = payment.ID
		destination, err := repo.CreatePaymentDestination(ctx, dest)
		if err != nil {
			return Payment{}, nil, fmt.Errorf("failed to create payment destination: %w", err)
		}
		destinations = append(destinations, destination)
	}

	if err := tx.Commit(); err != nil {
		return Payment{}, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return payment, destinations, nil
}

// CreateTransactionWithCallbackParams is a struct for CreateTransactionWithCallback method
type CreateTransactionWithCallbackParams struct {
	Transaction CreateTransactionParams
	Callback    func() error
}

// CreateTransactionWithCallback creates a new transaction with callback
func (q *QueriesTx) CreateTransactionWithCallback(ctx context.Context, arg CreateTransactionWithCallbackParams) (Transaction, error) {
	tx, err := q.dbConn.Begin()
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repo := q.WithTx(tx)

	transaction, err := repo.CreateTransaction(ctx, arg.Transaction)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	if err := arg.Callback(); err != nil {
		return Transaction{}, fmt.Errorf("failed to execute callback: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Transaction{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return transaction, nil
}
