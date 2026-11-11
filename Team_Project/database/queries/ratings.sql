-- Rating summary, distribution, and owner-scoped writes.
-- book_rating_summaries is maintained by PostgreSQL triggers; this file only
-- reads it and upserts/deletes the caller's own rating row.

-- name: GetBookRatingSummary :one
SELECT
    book_id,
    average_rating,
    rating_count,
    review_count,
    updated_at
FROM book_rating_summaries
WHERE book_id = sqlc.arg('book_id');

-- name: GetRatingDistribution :many
SELECT
    rating_value,
    count(*)::int AS rating_count
FROM ratings
WHERE book_id = sqlc.arg('book_id')
GROUP BY rating_value
ORDER BY rating_value;

-- name: GetMyRating :one
SELECT
    rating_id,
    user_id,
    book_id,
    rating_value,
    created_at,
    updated_at
FROM ratings
WHERE user_id = sqlc.arg('user_id')
  AND book_id = sqlc.arg('book_id');

-- name: UpsertRating :one
INSERT INTO ratings (user_id, book_id, rating_value)
VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('book_id'),
    sqlc.arg('rating_value')
)
ON CONFLICT (user_id, book_id) DO UPDATE
    SET rating_value = EXCLUDED.rating_value,
        updated_at = now()
RETURNING
    rating_id,
    user_id,
    book_id,
    rating_value,
    created_at,
    updated_at;

-- name: DeleteMyRating :execrows
DELETE FROM ratings
WHERE user_id = sqlc.arg('user_id')
  AND book_id = sqlc.arg('book_id');

-- name: ListMyRatings :many
SELECT
    rt.rating_id,
    rt.user_id,
    rt.book_id,
    rt.rating_value,
    rt.created_at,
    rt.updated_at,
    b.title AS book_title,
    b.cover_url,
    (count(*) OVER ())::int AS total_count
FROM ratings rt
JOIN books b ON b.book_id = rt.book_id
WHERE rt.user_id = sqlc.arg('user_id')
ORDER BY rt.updated_at DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;
