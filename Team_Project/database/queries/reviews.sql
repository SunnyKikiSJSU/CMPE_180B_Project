-- Review reads and owner-scoped writes.
-- The unique constraint reviews_user_book_unique is the final duplicate guard.

-- name: ListBookReviews :many
WITH params AS (
    SELECT coalesce(sqlc.narg('sort')::text, 'newest') AS sort
)
SELECT
    r.review_id,
    r.book_id,
    r.user_id,
    r.title,
    r.review_text,
    r.created_at,
    r.updated_at,
    u.display_name,
    u.avatar_url,
    rt.rating_value,
    (count(*) OVER ())::int AS total_count
FROM reviews r
JOIN users u ON u.user_id = r.user_id
LEFT JOIN ratings rt
    ON rt.user_id = r.user_id
   AND rt.book_id = r.book_id
CROSS JOIN params p
WHERE r.book_id = sqlc.arg('book_id')
ORDER BY
    CASE WHEN p.sort = 'rating' THEN rt.rating_value END DESC NULLS LAST,
    r.created_at DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: GetReviewByID :one
SELECT
    r.review_id,
    r.book_id,
    r.user_id,
    r.title,
    r.review_text,
    r.created_at,
    r.updated_at,
    u.display_name,
    u.avatar_url,
    rt.rating_value
FROM reviews r
JOIN users u ON u.user_id = r.user_id
LEFT JOIN ratings rt
    ON rt.user_id = r.user_id
   AND rt.book_id = r.book_id
WHERE r.review_id = sqlc.arg('review_id');

-- name: GetMyReviewForBook :one
SELECT
    review_id,
    book_id,
    user_id,
    title,
    review_text,
    created_at,
    updated_at
FROM reviews
WHERE user_id = sqlc.arg('user_id')
  AND book_id = sqlc.arg('book_id');

-- name: CreateReview :one
INSERT INTO reviews (user_id, book_id, title, review_text)
VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('book_id'),
    sqlc.narg('title'),
    sqlc.arg('review_text')
)
RETURNING
    review_id,
    book_id,
    user_id,
    title,
    review_text,
    created_at,
    updated_at;

-- name: UpdateReview :one
UPDATE reviews
SET
    title = coalesce(sqlc.narg('title'), title),
    review_text = coalesce(sqlc.narg('review_text'), review_text)
WHERE review_id = sqlc.arg('review_id')
  AND user_id = sqlc.arg('user_id')
RETURNING
    review_id,
    book_id,
    user_id,
    title,
    review_text,
    created_at,
    updated_at;

-- name: DeleteReview :execrows
DELETE FROM reviews
WHERE review_id = sqlc.arg('review_id')
  AND user_id = sqlc.arg('user_id');

-- name: ListMyReviews :many
SELECT
    r.review_id,
    r.book_id,
    r.user_id,
    r.title,
    r.review_text,
    r.created_at,
    r.updated_at,
    b.title AS book_title,
    b.cover_url,
    rt.rating_value,
    (count(*) OVER ())::int AS total_count
FROM reviews r
JOIN books b ON b.book_id = r.book_id
LEFT JOIN ratings rt
    ON rt.user_id = r.user_id
   AND rt.book_id = r.book_id
WHERE r.user_id = sqlc.arg('user_id')
ORDER BY r.created_at DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;
