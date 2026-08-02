-- name: CreateUser :one
INSERT INTO
    users (
        id,
        email,
        hashed_password,
        created_at,
        updated_at
    )
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING
    *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: RemoveRefreshToken :exec
DELETE FROM refresh_tokens WHERE token = $1;