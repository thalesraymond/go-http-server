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

-- name: GetAllChirpsAsc :many
SELECT *
FROM chirps
WHERE
    COALESCE(
        sqlc.narg ('author_id')::uuid,
        user_id
    ) = user_id
ORDER BY created_at ASC;

-- name: GetAllChirpsDesc :many
SELECT *
FROM chirps
WHERE
    COALESCE(
        sqlc.narg ('author_id')::uuid,
        user_id
    ) = user_id
ORDER BY created_at DESC;

-- name: GetChirpByID :one
SELECT * FROM chirps WHERE id = $1;

-- name: DeleteChirpByID :exec
DELETE FROM chirps WHERE id = $1;