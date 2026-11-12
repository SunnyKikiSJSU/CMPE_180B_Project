-- Author listing, detail, and author-scoped book listing.

-- name: ListAuthors :many
WITH params AS (
    SELECT nullif(btrim(sqlc.narg('q')::text), '') AS q
)
SELECT
    a.author_id,
    a.name,
    count(ba.book_id)::int AS book_count,
    (count(*) OVER ())::int AS total_count
FROM authors a
LEFT JOIN book_authors ba ON ba.author_id = a.author_id
CROSS JOIN params p
-- LIKE wildcards in user text are escaped so they match literally.
WHERE p.q IS NULL
   OR a.name ILIKE '%' || replace(replace(replace(p.q, '\', '\\'), '%', '\%'), '_', '\_') || '%'
GROUP BY a.author_id, a.name
ORDER BY a.name
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: GetAuthorByID :one
SELECT author_id, name
FROM authors
WHERE author_id = sqlc.arg('author_id');

-- name: ListBooksByAuthor :many
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
    a.name::text AS primary_author,
    coalesce(
        (
            SELECT json_agg(
                json_build_object('id', a2.author_id, 'name', a2.name)
                ORDER BY a2.name
            )
            FROM book_authors ba2
            JOIN authors a2 ON a2.author_id = ba2.author_id
            WHERE ba2.book_id = b.book_id
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
    (count(*) OVER ())::int AS total_count
FROM books b
JOIN book_authors ba ON ba.book_id = b.book_id
JOIN authors a ON a.author_id = ba.author_id
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
WHERE ba.author_id = sqlc.arg('author_id')
ORDER BY b.title
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;
