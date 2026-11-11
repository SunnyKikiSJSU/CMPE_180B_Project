-- Catalog search, detail, and catalog-write queries.
--
-- Search uses both full-text ranking (title/description) and trigram-style
-- ILIKE membership through authors and categories. EXISTS keeps the outer
-- result one row per book; json_agg provides the many-to-many arrays.
--
-- User text is used as a substring pattern, so LIKE wildcards are escaped:
-- searching for `100%` or `a_b` matches those characters literally.

-- name: SearchBooks :many
WITH params AS (
    SELECT
        nullif(btrim(sqlc.narg('q')::text), '') AS q,
        nullif(btrim(sqlc.narg('title')::text), '') AS title,
        sqlc.narg('author_id')::uuid AS author_id,
        sqlc.narg('category_id')::uuid AS category_id,
        coalesce(sqlc.narg('sort')::text, 'relevance') AS sort
),
patterns AS (
    SELECT
        p.q,
        p.title,
        p.author_id,
        p.category_id,
        p.sort,
        '%' || replace(replace(replace(p.q, '\', '\\'), '%', '\%'), '_', '\_') || '%' AS q_like,
        '%' || replace(replace(replace(p.title, '\', '\\'), '%', '\%'), '_', '\_') || '%' AS title_like
    FROM params p
),
matched AS (
    SELECT
        b.book_id,
        CASE
            WHEN p.q IS NULL THEN 0
            ELSE ts_rank(
                b.search_vector,
                websearch_to_tsquery('english', p.q)
            )
        END AS relevance
    FROM books b
    CROSS JOIN patterns p
    WHERE p.q IS NULL
       OR b.search_vector @@ websearch_to_tsquery('english', p.q)
       OR b.title ILIKE p.q_like
       OR EXISTS (
            SELECT 1
            FROM book_authors ba
            JOIN authors a ON a.author_id = ba.author_id
            WHERE ba.book_id = b.book_id
              AND a.name ILIKE p.q_like
       )
       OR EXISTS (
            SELECT 1
            FROM book_categories bc
            JOIN categories c ON c.category_id = bc.category_id
            WHERE bc.book_id = b.book_id
              AND c.name ILIKE p.q_like
       )
)
SELECT
    b.book_id,
    b.google_books_id,
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
    (count(*) OVER ())::int AS total_count
FROM matched m
JOIN books b ON b.book_id = m.book_id
CROSS JOIN patterns p
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
WHERE (p.title IS NULL OR b.title ILIKE p.title_like)
  AND (
        p.author_id IS NULL
        OR EXISTS (
            SELECT 1
            FROM book_authors ba
            WHERE ba.book_id = b.book_id
              AND ba.author_id = p.author_id
        )
    )
  AND (
        p.category_id IS NULL
        OR EXISTS (
            SELECT 1
            FROM book_categories bc
            WHERE bc.book_id = b.book_id
              AND bc.category_id = p.category_id
        )
    )
ORDER BY
    CASE WHEN p.sort = 'relevance' THEN m.relevance END DESC,
    CASE WHEN p.sort = 'newest' THEN b.published_date END DESC NULLS LAST,
    CASE WHEN p.sort = 'title' THEN b.title END ASC,
    CASE WHEN p.sort = 'rating' THEN coalesce(s.average_rating, 0) END DESC,
    CASE WHEN p.sort = 'popular' THEN coalesce(s.rating_count, 0) END DESC,
    b.title ASC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: GetBookByID :one
SELECT
    b.book_id,
    b.google_books_id,
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
    coalesce(s.review_count, 0)::int AS review_count
FROM books b
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
WHERE b.book_id = sqlc.arg('book_id');

-- name: GetBookByExternalID :one
WITH params AS (
    SELECT
        sqlc.narg('google_books_id')::text AS google_books_id,
        sqlc.narg('isbn13')::text AS isbn13
)
SELECT
    b.book_id,
    b.google_books_id,
    b.isbn13,
    b.title
FROM books b
CROSS JOIN params p
WHERE (p.google_books_id IS NOT NULL AND b.google_books_id = p.google_books_id)
   OR (p.isbn13 IS NOT NULL AND b.isbn13 = p.isbn13)
LIMIT 1;

-- name: CreateBook :one
INSERT INTO books (
    google_books_id,
    isbn13,
    title,
    description,
    published_date,
    publisher,
    page_count,
    cover_url
)
VALUES (
    sqlc.narg('google_books_id'),
    sqlc.narg('isbn13'),
    sqlc.arg('title'),
    sqlc.narg('description'),
    sqlc.narg('published_date'),
    sqlc.narg('publisher'),
    sqlc.narg('page_count'),
    sqlc.narg('cover_url')
)
RETURNING
    book_id,
    google_books_id,
    isbn13,
    title,
    description,
    published_date,
    publisher,
    page_count,
    cover_url;

-- name: UpdateBook :one
UPDATE books
SET
    title = coalesce(sqlc.narg('title'), title),
    description = coalesce(sqlc.narg('description'), description),
    published_date = coalesce(sqlc.narg('published_date'), published_date),
    publisher = coalesce(sqlc.narg('publisher'), publisher),
    page_count = coalesce(sqlc.narg('page_count'), page_count),
    cover_url = coalesce(sqlc.narg('cover_url'), cover_url),
    isbn13 = coalesce(sqlc.narg('isbn13'), isbn13)
WHERE book_id = sqlc.arg('book_id')
RETURNING
    book_id,
    google_books_id,
    isbn13,
    title,
    description,
    published_date,
    publisher,
    page_count,
    cover_url;

-- name: UpsertAuthor :one
INSERT INTO authors (name)
VALUES (sqlc.arg('name'))
ON CONFLICT (lower(btrim(name))) DO UPDATE
    SET name = authors.name
RETURNING author_id, name;

-- name: UpsertCategory :one
INSERT INTO categories (name)
VALUES (sqlc.arg('name'))
ON CONFLICT (lower(btrim(name))) DO UPDATE
    SET name = categories.name
RETURNING category_id, name;

-- name: ClearBookAuthors :exec
DELETE FROM book_authors
WHERE book_id = sqlc.arg('book_id');

-- name: ClearBookCategories :exec
DELETE FROM book_categories
WHERE book_id = sqlc.arg('book_id');

-- name: AddBookAuthor :exec
INSERT INTO book_authors (book_id, author_id)
VALUES (sqlc.arg('book_id'), sqlc.arg('author_id'))
ON CONFLICT DO NOTHING;

-- name: AddBookCategory :exec
INSERT INTO book_categories (book_id, category_id)
VALUES (sqlc.arg('book_id'), sqlc.arg('category_id'))
ON CONFLICT DO NOTHING;
