-- name: CreatePayment :one
INSERT INTO payments (
    external_id, 
    destination_wallet, 
    destination_mint,
    amount, 
    status, 
    message, 
    expires_at
) 
VALUES (
    @external_id, 
    @destination_wallet, 
    @destination_mint,
    @amount, 
    @status, 
    @message, 
    @expires_at
)
RETURNING *;

-- name: GetPayment :one
SELECT * FROM payments WHERE id = @id;

-- name: GetPaymentByExternalID :one
SELECT * FROM payments WHERE external_id = @external_id::VARCHAR;

-- name: UpdatePaymentStatus :one
UPDATE payments SET status = @status WHERE id = @id RETURNING *;

-- name: MarkPaymentsExpired :exec
UPDATE payments SET status = 'expired'::payment_status WHERE expires_at < NOW() AND status = 'new'::payment_status;