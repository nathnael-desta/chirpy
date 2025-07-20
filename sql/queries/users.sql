-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
        gen_random_uuid(),
        NOW(),
        NOW(),
        $1
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
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
        gen_random_uuid(),
        NOW(),
        NOW(),
        $1,
        $2
    )
RETURNING *;