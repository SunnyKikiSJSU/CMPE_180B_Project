-- Catalog search, detail, and catalog-write queries.
--
-- Two kinds of search exist:
--   * Outside search (GET /api/external/books) calls the Open Library search
--     API from the Go service and stores nothing. ListBooksByExternalIDs marks
--     which results are already in the local library.
--   * Library search (SearchBooks, GET /api/books) searches the local books
--     table: seed, imported, and user-created rows.
-- An Open Library work becomes a local row through ImportOpenLibraryBook when a
-- user imports it, because every user-activity table references books.book_id.
--
-- Library search uses both full-text ranking (title/description) and
-- trigram-style ILIKE membership through authors and categories. EXISTS keeps
-- the outer result one row per book; json_agg provides the many-to-many arrays.
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
    b.openlibrary_id,
    b.isbn13,
    b.title,
    b.description,
    b.published_date,
    b.publisher,
    b.page_count,
    b.cover_url,
    b.source,
    b.published_date_precision,
    coalesce(
        (
            SELECT a.name
            FROM book_authors ba
            JOIN authors a ON a.author_id = ba.author_id
            WHERE ba.book_id = b.book_id
            ORDER BY a.name
            LIMIT 1
        ),
        'Unknown author'
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
    b.openlibrary_id,
    b.isbn13,
    b.title,
    b.description,
    b.published_date,
    b.publisher,
    b.page_count,
    b.cover_url,
    b.source,
    b.published_date_precision,
    coalesce(
        (
            SELECT a.name
            FROM book_authors ba
            JOIN authors a ON a.author_id = ba.author_id
            WHERE ba.book_id = b.book_id
            ORDER BY a.name
            LIMIT 1
        ),
        'Unknown author'
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

-- Import dedupe (POST /api/books/import): the local row for an Open Library
-- work, found by work ID first, then by any ISBN-13 of the work's editions
-- (a seed or user-created book with the same ISBN is the same book).
-- name: FindImportedBook :one
SELECT
    b.book_id,
    b.openlibrary_id,
    b.isbn13
FROM books b
WHERE b.openlibrary_id = sqlc.arg('openlibrary_id')::text
   OR b.isbn13 = ANY (sqlc.arg('isbn13s')::text[])
ORDER BY (b.openlibrary_id IS NOT DISTINCT FROM sqlc.arg('openlibrary_id')::text) DESC, b.book_id
LIMIT 1;

-- Marks outside search results that already exist locally, so each result can
-- carry its local bookId. One call per search page; the same rule as
-- FindImportedBook (work ID, then ISBN-13).
-- name: ListBooksByExternalIDs :many
SELECT
    b.book_id,
    b.openlibrary_id,
    b.isbn13
FROM books b
WHERE b.openlibrary_id = ANY (sqlc.arg('openlibrary_ids')::text[])
   OR b.isbn13 = ANY (sqlc.arg('isbn13s')::text[]);

-- User-created catalog record (POST /api/books). It never sets an outside ID;
-- imported rows come only from ImportOpenLibraryBook.
-- name: CreateBook :one
INSERT INTO books (
    isbn13,
    title,
    description,
    published_date,
    published_date_precision,
    publisher,
    page_count,
    cover_url,
    source
)
VALUES (
    sqlc.narg('isbn13'),
    sqlc.arg('title'),
    sqlc.narg('description'),
    sqlc.narg('published_date')::date,
    CASE WHEN sqlc.narg('published_date')::date IS NULL THEN NULL ELSE 'day' END,
    sqlc.narg('publisher'),
    sqlc.narg('page_count'),
    sqlc.narg('cover_url'),
    'user'
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
    cover_url,
    source,
    published_date_precision;

-- Imported rows (Google, Open Library) are read-only here: they mirror an
-- outside catalog and are shown with its attribution. The service returns 403
-- for them; the source filter is the database-side guard.
-- name: UpdateBook :one
UPDATE books
SET
    title = coalesce(sqlc.narg('title'), title),
    description = coalesce(sqlc.narg('description'), description),
    published_date = coalesce(sqlc.narg('published_date')::date, published_date),
    published_date_precision = CASE
        WHEN sqlc.narg('published_date')::date IS NOT NULL THEN 'day'
        ELSE published_date_precision
    END,
    publisher = coalesce(sqlc.narg('publisher'), publisher),
    page_count = coalesce(sqlc.narg('page_count'), page_count),
    cover_url = coalesce(sqlc.narg('cover_url'), cover_url),
    isbn13 = coalesce(sqlc.narg('isbn13'), isbn13)
WHERE book_id = sqlc.arg('book_id')
  AND source NOT IN ('google', 'openlibrary')
RETURNING
    book_id,
    google_books_id,
    isbn13,
    title,
    description,
    published_date,
    publisher,
    page_count,
    cover_url,
    source,
    published_date_precision;

-- Creates the local row for an Open Library work (POST /api/books/import).
-- The service fetches the work from Open Library by ID and normalizes it;
-- client metadata is never trusted. Concurrent imports of the same work return
-- the same row: the no-op DO UPDATE makes RETURNING yield the existing row. An
-- ISBN-13 collision with a different row raises 23505 on books_isbn13_unique;
-- the service then reruns FindImportedBook and returns that row.
-- name: ImportOpenLibraryBook :one
INSERT INTO books (
    openlibrary_id,
    isbn13,
    title,
    description,
    published_date,
    published_date_precision,
    publisher,
    page_count,
    cover_url,
    source,
    imported_at
)
VALUES (
    sqlc.arg('openlibrary_id')::text,
    sqlc.narg('isbn13'),
    sqlc.arg('title'),
    sqlc.narg('description'),
    sqlc.narg('published_date'),
    sqlc.narg('published_date_precision'),
    sqlc.narg('publisher'),
    sqlc.narg('page_count'),
    sqlc.narg('cover_url'),
    'openlibrary',
    now()
)
ON CONFLICT (openlibrary_id) DO UPDATE
    SET openlibrary_id = books.openlibrary_id
RETURNING
    book_id,
    (xmax = 0) AS inserted;

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
