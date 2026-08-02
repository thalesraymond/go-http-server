-- name: CreateUser :one
INSERT INTO
    users (
        id,
        email,
        created_at,
        updated_at
    )
VALUES ($1, $2, NOW(), NOW())
RETURNING
    *;

-- name: DeleteAllUsers :exec
DELETE FROM users;