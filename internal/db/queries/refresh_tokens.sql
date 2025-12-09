-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE id = $1 LIMIT 1;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 AND NOT revoked AND expires_at > NOW()
LIMIT 1;

-- name: UpdateRefreshTokenLastUsed :exec
UPDATE refresh_tokens
SET last_used_at = NOW()
WHERE id = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = true, revoked_at = NOW()
WHERE id = $1;

-- name: RevokeRefreshTokenByHash :exec
UPDATE refresh_tokens
SET revoked = true, revoked_at = NOW()
WHERE token_hash = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = true, revoked_at = NOW()
WHERE user_id = $1 AND NOT revoked;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < NOW();

-- name: ListUserRefreshTokens :many
SELECT * FROM refresh_tokens
WHERE user_id = $1
ORDER BY created_at DESC;
