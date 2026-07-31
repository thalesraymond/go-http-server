-- name: CreateUser :one
INSERT INTO
    users (email, created_at, updated_at)
VALUES ($1, NOW(), NOW())
RETURNING
    *;