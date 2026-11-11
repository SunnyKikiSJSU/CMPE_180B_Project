-- Shelf CRUD and shelf-book membership queries.
-- Status shelves are created only by fn_create_user_defaults; user-created
-- shelves are always shelf_type = 'custom'.

-- name: ListShelves :many
SELECT
    s.shelf_id,
    s.name,
    s.shelf_type,
    s.created_at,
    count(usi.book_id)::int AS book_count
FROM shelves s
LEFT JOIN user_shelf_items usi ON usi.shelf_id = s.shelf_id
WHERE s.user_id = sqlc.arg('user_id')
GROUP BY s.shelf_id, s.name, s.shelf_type, s.created_at
ORDER BY
    CASE s.shelf_type
        WHEN 'want_to_read' THEN 1
        WHEN 'currently_reading' THEN 2
        WHEN 'finished' THEN 3
        WHEN 'paused' THEN 4
        ELSE 5
    END,
    s.created_at,
    s.shelf_id;

-- name: GetShelf :one
SELECT
    s.shelf_id,
    s.user_id,
    s.name,
    s.shelf_type,
    s.created_at,
    (
        SELECT count(*)::int
        FROM user_shelf_items usi
        WHERE usi.shelf_id = s.shelf_id
    ) AS book_count
FROM shelves s
WHERE s.shelf_id = sqlc.arg('shelf_id')
  AND s.user_id = sqlc.arg('user_id');

-- name: CreateShelf :one
INSERT INTO shelves (user_id, name, shelf_type)
VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('name'),
    'custom'
)
RETURNING
    shelf_id,
    name,
    shelf_type,
    created_at;

-- name: RenameShelf :one
UPDATE shelves
SET name = sqlc.arg('name')
WHERE shelf_id = sqlc.arg('shelf_id')
  AND user_id = sqlc.arg('user_id')
  AND shelf_type = 'custom'
RETURNING
    shelf_id,
    name,
    shelf_type,
    created_at;

-- name: DeleteShelf :execrows
DELETE FROM shelves
WHERE shelf_id = sqlc.arg('shelf_id')
  AND user_id = sqlc.arg('user_id')
  AND shelf_type = 'custom';

-- name: CountCustomShelves :one
SELECT count(*)::int
FROM shelves
WHERE user_id = sqlc.arg('user_id')
  AND shelf_type = 'custom';

-- name: ListShelfBooks :many
SELECT
    b.book_id,
    b.isbn13,
    b.title,
    b.description,
    b.published_date,
    b.publisher,
    b.page_count,
    b.cover_url,
    (
        SELECT a.name
        FROM book_authors ba
        JOIN authors a ON a.author_id = ba.author_id
        WHERE ba.book_id = b.book_id
        ORDER BY a.name
        LIMIT 1
    )::text AS primary_author,
    coalesce(
        (
            SELECT json_agg(
                json_build_object('id', a.author_id, 'name', a.name)
                ORDER BY a.name
            )
            FROM book_authors ba
            JOIN authors a ON a.author_id = ba.author_id
            WHERE ba.book_id = b.book_id
        ),
        '[]'::json
    ) AS authors,
    coalesce(
        (
            SELECT json_agg(
                json_build_object('id', c.category_id, 'name', c.name)
                ORDER BY c.name
            )
            FROM book_categories bc
            JOIN categories c ON c.category_id = bc.category_id
            WHERE bc.book_id = b.book_id
        ),
        '[]'::json
    ) AS categories,
    coalesce(s.average_rating, 0)::numeric(3, 2) AS average_rating,
    coalesce(s.rating_count, 0)::int AS rating_count,
    coalesce(s.review_count, 0)::int AS review_count,
    usi.added_at,
    usi.moved_at,
    rp.status AS progress_status,
    rp.current_page,
    coalesce(b.page_count, 0)::int AS total_pages,
    rp.percent_complete,
    rp.version AS progress_version,
    rp.updated_at AS progress_updated_at,
    (
        SELECT max(rh.created_at)
        FROM reading_history rh
        WHERE rh.user_id = sh.user_id
          AND rh.book_id = b.book_id
    ) AS last_activity_at,
    (count(*) OVER ())::int AS total_count
FROM shelves sh
JOIN user_shelf_items usi ON usi.shelf_id = sh.shelf_id
JOIN books b ON b.book_id = usi.book_id
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
LEFT JOIN reading_progress rp
    ON rp.user_id = sh.user_id
   AND rp.book_id = b.book_id
WHERE sh.shelf_id = sqlc.arg('shelf_id')
  AND sh.user_id = sqlc.arg('user_id')
ORDER BY usi.added_at DESC, b.title
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- The service must verify shelf ownership. For status shelves, prefer
-- fn_move_book_to_shelf so progress and history stay in sync.
-- name: AddShelfItem :exec
INSERT INTO user_shelf_items (shelf_id, book_id, added_at, moved_at)
VALUES (
    sqlc.arg('shelf_id'),
    sqlc.arg('book_id'),
    now(),
    now()
)
ON CONFLICT (shelf_id, book_id) DO UPDATE
    SET moved_at = now();

-- name: RemoveShelfItem :execrows
DELETE FROM user_shelf_items usi
USING shelves s
WHERE usi.shelf_id = s.shelf_id
  AND usi.shelf_id = sqlc.arg('shelf_id')
  AND usi.book_id = sqlc.arg('book_id')
  AND s.user_id = sqlc.arg('user_id');
