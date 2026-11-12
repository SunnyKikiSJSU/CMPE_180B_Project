-- Category listing, detail, and category-scoped book listing.

-- name: ListCategories :many
WITH params AS (
    SELECT nullif(btrim(sqlc.narg('q')::text), '') AS q
)
SELECT
    c.category_id,
    c.name,
    count(bc.book_id)::int AS book_count,
    (count(*) OVER ())::int AS total_count
FROM categories c
LEFT JOIN book_categories bc ON bc.category_id = c.category_id
CROSS JOIN params p
-- LIKE wildcards in user text are escaped so they match literally.
WHERE p.q IS NULL
   OR c.name ILIKE '%' || replace(replace(replace(p.q, '\', '\\'), '%', '\%'), '_', '\_') || '%'
GROUP BY c.category_id, c.name
ORDER BY c.name
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: GetCategoryByID :one
SELECT category_id, name
FROM categories
WHERE category_id = sqlc.arg('category_id');

-- name: ListBooksByCategory :many
SELECT
    b.book_id,
    b.isbn13,
    b.title,
    b.description,
    b.published_date,
    b.publisher,
    b.page_count,
    b.cover_url,
    b.google_books_id,
    b.source,
    b.published_date_precision,
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
                json_build_object('id', c2.category_id, 'name', c2.name)
                ORDER BY c2.name
            )
            FROM book_categories bc2
            JOIN categories c2 ON c2.category_id = bc2.category_id
            WHERE bc2.book_id = b.book_id
        ),
        '[]'::json
    ) AS categories,
    coalesce(s.average_rating, 0)::numeric(3, 2) AS average_rating,
    coalesce(s.rating_count, 0)::int AS rating_count,
    coalesce(s.review_count, 0)::int AS review_count,
    (count(*) OVER ())::int AS total_count
FROM books b
JOIN book_categories bc ON bc.book_id = b.book_id
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
WHERE bc.category_id = sqlc.arg('category_id')
ORDER BY b.title
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;
