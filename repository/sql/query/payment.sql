-- name: CreatePayment :one
INSERT INTO payments (external_id, currency, total_amount, status, message, memo, expires_at) 
VALUES (@external_id, @currency, @total_amount, @status, @message, @memo, @expires_at)
RETURNING *;

-- name: GetPayment :one
SELECT * FROM payments WHERE id = @id;

-- name: GetPaymentByExternalID :one
SELECT * FROM payments WHERE external_id = @external_id::VARCHAR;

-- name: UpdatePaymentStatus :one
UPDATE payments SET status = @status WHERE id = @id RETURNING *;

-- name: CreatePaymentDestination :one
INSERT INTO payment_destinations (payment_id, destination, amount, percentage, total_amount, discount_amount, apply_bonus, max_bonus_amount, max_bonus_percentage)
VALUES (@payment_id, @destination, @amount, @percentage, @total_amount, @discount_amount, @apply_bonus, @max_bonus_amount, @max_bonus_percentage)
RETURNING *;

-- name: GetPaymentDestinations :many
SELECT * FROM payment_destinations WHERE payment_id = @payment_id;

-- name: DeletePaymentDestinations :exec
DELETE FROM payment_destinations WHERE payment_id = @payment_id;