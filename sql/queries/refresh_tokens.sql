-- name: CreateRefreshToken :one
INSERT INTO
    refresh_tokens (
        user_id,
        token,
        expires_at,
        created_at,
        updated_at
    )
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING
    *;

-- name: GetRefreshTokenByID :one
SELECT * FROM refresh_tokens WHERE token = $1;