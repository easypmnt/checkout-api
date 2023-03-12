-- name: StoreToken :one
INSERT INTO tokens (
	token_type,
	credential,
	access_token_id,
	refresh_token_id,
	access_expires_at,
	refresh_expires_at
) VALUES (
	@token_type,
	@credential,
	@access_token_id,
	@refresh_token_id,
	@access_expires_at,
	@refresh_expires_at
) ON CONFLICT (token_type, credential) DO UPDATE SET
	access_token_id = @access_token_id,
	refresh_token_id = @refresh_token_id,
	access_expires_at = @access_expires_at,
	refresh_expires_at = @refresh_expires_at
RETURNING *;

-- name: GetToken :one
SELECT * FROM tokens
WHERE token_type = @token_type
AND credential = @credential
AND access_token_id = @access_token_id
AND refresh_token_id = @refresh_token_id;

-- name: DeleteToken :exec
DELETE FROM tokens WHERE token_type = @token_type AND credential = @credential;

-- name: DeleteExpiredTokens :exec
DELETE FROM tokens WHERE refresh_expires_at < NOW();

-- name: DeleteTokensByCredential :exec
DELETE FROM tokens WHERE credential = @credential;
