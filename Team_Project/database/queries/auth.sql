-- Authentication queries.
-- The service hashes and verifies passwords; these queries only persist and
-- retrieve the stored hash.

-- name: GetUserByEmail :one
SELECT
    user_id,
    email,
    password_hash,
    display_name,
    avatar_url,
    created_at
FROM users
WHERE email = sqlc.arg('email');

-- name: GetUserByID :one
SELECT
    user_id,
    email,
    password_hash,
    display_name,
    avatar_url,
    created_at
FROM users
WHERE user_id = sqlc.arg('user_id');

-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name, avatar_url)
VALUES (
    sqlc.arg('email'),
    sqlc.arg('password_hash'),
    sqlc.arg('display_name'),
    sqlc.narg('avatar_url')
)
RETURNING
    user_id,
    email,
    display_name,
    avatar_url,
    created_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = sqlc.arg('password_hash')
WHERE user_id = sqlc.arg('user_id');
