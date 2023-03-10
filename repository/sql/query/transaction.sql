-- name: CreateTransaction :one
INSERT INTO transactions (payment_id, reference, status) 
VALUES (@payment_id, @reference, @status)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = @id;

-- name: GetTransactionByReference :one
SELECT * FROM transactions WHERE reference = @reference;

-- name: GetTransactionsByPaymentID :many
SELECT * FROM transactions WHERE payment_id = @payment_id;

-- name: UpdateTransactionByReference :one
UPDATE transactions SET tx_signature = @tx_signature, status = @status WHERE reference = @reference RETURNING *;
