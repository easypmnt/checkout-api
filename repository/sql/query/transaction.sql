-- name: CreateTransaction :one
INSERT INTO transactions (
    payment_id, 
    reference, 
    source_wallet,
    source_mint,
    destination_wallet,
    destination_mint,
    amount, 
    discount_amount, 
    total_amount,
    accrued_bonus_amount,
    message,
    memo,
    apply_bonus,
    status
) 
VALUES (
    @payment_id, 
    @reference, 
    @source_wallet,
    @source_mint,
    @destination_wallet,
    @destination_mint,
    @amount, 
    @discount_amount, 
    @total_amount,
    @accrued_bonus_amount,
    @message,
    @memo,
    @apply_bonus,
    @status
)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = @id;

-- name: GetTransactionByReference :one
SELECT * FROM transactions WHERE reference = @reference;

-- name: GetTransactionsByPaymentID :many
SELECT * FROM transactions WHERE payment_id = @payment_id ORDER BY created_at DESC;

-- name: UpdateTransactionByReference :one
UPDATE transactions SET tx_signature = @tx_signature, status = @status WHERE reference = @reference RETURNING *;

-- name: GetTransactionByPaymentIDSourceWalletAndMint :one
SELECT * FROM transactions 
WHERE payment_id = @payment_id 
    AND source_wallet = @source_wallet 
    AND source_mint = @source_mint
    AND status = 'pending'::transaction_status
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPendingTransactions :many
SELECT * FROM transactions WHERE status = 'pending'::transaction_status;

-- name: MarkTransactionsAsExpired :exec
UPDATE transactions SET status = 'expired'::transaction_status 
WHERE status = 'pending'::transaction_status AND payment_id IN (
    SELECT id FROM payments WHERE status = 'expired'::payment_status
);