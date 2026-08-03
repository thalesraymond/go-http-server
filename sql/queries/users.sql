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

-- name: UpdateUser :one
UPDATE users
SET
    email = $2,
    hashed_password = $3,
    updated_at = NOW()
WHERE
    id = $1
RETURNING
    *;

-- name: UpdateChirpyRedFlag :one
UPDATE users
SET
    is_chirpy_red = $2,
    updated_at = NOW()
WHERE
    id = $1
RETURNING
    *;