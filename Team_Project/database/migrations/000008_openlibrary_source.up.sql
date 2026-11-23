BEGIN;

-- 000008: books imported from Open Library.
--
-- Outside search uses Open Library (openlibrary.org). Its catalog data is
-- openly reusable, so a result is stored as a local books row when a user
-- imports it (POST /api/books/import); every foreign key still points at the
-- local book_id. One row per Open Library *work* (all editions of a book):
--
--   openlibrary_id  The work ID, for example OL893414W. Deduplicates imports
--                   and builds the link back to openlibrary.org.
--   source          Gains 'openlibrary'. Such rows record imported_at, like
--                   Google rows (google_books_id stays for a later Google source).

ALTER TABLE books
    ADD COLUMN openlibrary_id TEXT;

ALTER TABLE books
    ADD CONSTRAINT books_openlibrary_id_unique UNIQUE (openlibrary_id),
    ADD CONSTRAINT books_openlibrary_id_format
        CHECK (openlibrary_id IS NULL OR openlibrary_id ~ '^OL[0-9]+W$'),
    DROP CONSTRAINT books_source_valid,
    ADD CONSTRAINT books_source_valid
        CHECK (source IN ('seed', 'google', 'openlibrary', 'user')),
    ADD CONSTRAINT books_openlibrary_source_complete
        CHECK (
            source <> 'openlibrary'
            OR (openlibrary_id IS NOT NULL AND imported_at IS NOT NULL)
        );

COMMIT;
