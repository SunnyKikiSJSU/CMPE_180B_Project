-- Reading progress and history reads.
--
-- Writes go through fn_update_reading_progress and fn_move_book_to_shelf,
-- which are called directly with pgx because they return a composite row and
-- enforce locking/validation inside PostgreSQL.

-- name: ListReadingProgress :many
WITH params AS (
    SELECT sqlc.narg('status')::text AS status
)
SELECT
    rp.book_id,
    rp.status,
    rp.current_page,
    coalesce(b.page_count, 0)::int AS total_pages,
    rp.percent_complete,
    rp.version,
    rp.updated_at,
    b.title,
    (
        SELECT a.name
        FROM book_authors ba
        JOIN authors a ON a.author_id = ba.author_id
        WHERE ba.book_id = b.book_id
        ORDER BY a.name
        LIMIT 1
    )::text AS primary_author,
    b.cover_url,
    (count(*) OVER ())::int AS total_count
FROM reading_progress rp
JOIN books b ON b.book_id = rp.book_id
CROSS JOIN params p
WHERE rp.user_id = sqlc.arg('user_id')
  AND (p.status IS NULL OR rp.status = p.status)
ORDER BY rp.updated_at DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: GetReadingProgress :one
SELECT
    rp.book_id,
    rp.status,
    rp.current_page,
    coalesce(b.page_count, 0)::int AS total_pages,
    rp.percent_complete,
    rp.version,
    rp.updated_at,
    b.title,
    (
        SELECT a.name
        FROM book_authors ba
        JOIN authors a ON a.author_id = ba.author_id
        WHERE ba.book_id = b.book_id
        ORDER BY a.name
        LIMIT 1
    )::text AS primary_author,
    b.cover_url
FROM reading_progress rp
JOIN books b ON b.book_id = rp.book_id
WHERE rp.user_id = sqlc.arg('user_id')
  AND rp.book_id = sqlc.arg('book_id');

-- Removing progress also takes the book off the caller's status shelves, so
-- shelf membership never claims a status that no longer exists. Custom shelf
-- memberships are kept. The returned row count is the progress row only.
-- name: DeleteReadingProgress :execrows
WITH params AS (
    SELECT
        sqlc.arg('user_id')::uuid AS target_user_id,
        sqlc.arg('book_id')::uuid AS target_book_id
),
removed_status_items AS (
    DELETE FROM user_shelf_items usi
    USING shelves s, params p
    WHERE usi.shelf_id = s.shelf_id
      AND s.user_id = p.target_user_id
      AND s.shelf_type <> 'custom'
      AND usi.book_id = p.target_book_id
)
DELETE FROM reading_progress rp
USING params p
WHERE rp.user_id = p.target_user_id
  AND rp.book_id = p.target_book_id;

-- name: ListReadingHistory :many
WITH params AS (
    SELECT sqlc.narg('book_id')::uuid AS book_id
)
SELECT
    rh.history_id,
    rh.book_id,
    b.title AS book_title,
    rh.event_type,
    rh.from_status,
    rh.to_status,
    rh.created_at
FROM reading_history rh
JOIN books b ON b.book_id = rh.book_id
CROSS JOIN params p
WHERE rh.user_id = sqlc.arg('user_id')
  AND (p.book_id IS NULL OR rh.book_id = p.book_id)
ORDER BY rh.created_at DESC
LIMIT sqlc.arg('limit')::int;
