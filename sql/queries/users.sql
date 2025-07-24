-- name: CreateUser :one
INSERT INTO users (
        id,
        created_at,
        updated_at,
        email,
        hashed_password
    )
VALUES (
        gen_random_uuid(),
        NOW(),
        NOW(),
        $1,
        $2
    )
RETURNING *;
-- name: Reset :exec
DELETE FROM users;
-- name: EmailExists :one
SELECT 1
FROM users
WHERE email = $1
LIMIT 1;
-- name: CreateChirp :one
INSERT INTO chirps(id, created_at, updated_at, body, user_id)
VALUES (
        gen_random_uuid(),
        NOW(),
        NOW(),
        $1,
        $2
    )
RETURNING *;
-- name: GetAllChirps :many
SELECT *
FROM chirps
ORDER BY created_at ASC;
-- name: GetChirp :one
SELECT *
FROM chirps
WHERE id = $1;
-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3,
    NULL

)
RETURNING *;
-- name: GetRefreshToken :one
SELECT * 
FROM refresh_tokens
WHERE token = $1;