-- User profile and account queries. All writes are scoped by user_id, so a
-- caller cannot affect another account even if a handler forgets a check.

-- name: GetUserWithProfile :one
SELECT
    u.user_id,
    u.email,
    u.display_name,
    u.avatar_url,
    u.created_at,
    p.bio,
    p.favorite_genre,
    p.updated_at AS profile_updated_at
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.user_id
WHERE u.user_id = sqlc.arg('user_id');

-- name: UpdateUser :one
UPDATE users
SET
    display_name = coalesce(sqlc.narg('display_name'), display_name),
    avatar_url = coalesce(sqlc.narg('avatar_url'), avatar_url)
WHERE user_id = sqlc.arg('user_id')
RETURNING
    user_id,
    email,
    display_name,
    avatar_url,
    created_at;

-- name: UpsertUserProfile :one
INSERT INTO user_profiles (user_id, bio, favorite_genre)
VALUES (
    sqlc.arg('user_id'),
    sqlc.narg('bio'),
    sqlc.narg('favorite_genre')
)
ON CONFLICT (user_id) DO UPDATE
    SET bio = coalesce(EXCLUDED.bio, user_profiles.bio),
        favorite_genre = coalesce(
            EXCLUDED.favorite_genre,
            user_profiles.favorite_genre
        )
RETURNING
    user_id,
    bio,
    favorite_genre,
    updated_at;

-- name: GetUserStats :one
WITH params AS (
    SELECT sqlc.arg('user_id')::uuid AS target_id
)
SELECT
    (SELECT count(*) FROM shelves s
     WHERE s.user_id = p.target_id)::int AS shelf_count,
    (SELECT count(*) FROM reading_progress rp
     WHERE rp.user_id = p.target_id
       AND rp.status = 'completed')::int AS books_completed,
    (SELECT count(*) FROM reviews r
     WHERE r.user_id = p.target_id)::int AS reviews_written,
    (SELECT count(*) FROM ratings rt
     WHERE rt.user_id = p.target_id)::int AS ratings_given
FROM params p;

-- Cascades to the profile, shelves, progress, ratings, reviews, and reading
-- history. The append-only history trigger allows deletes that arrive through
-- a foreign-key cascade (migration 000005), so no session setting is needed.
-- name: DeleteUser :exec
DELETE FROM users
WHERE user_id = sqlc.arg('user_id');
