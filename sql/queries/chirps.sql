-- name: CreateChirp :one
INSERT INTO
    chirps (
        id,
        body,
        user_id,
        created_at,
        updated_at
    )
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING
    *;