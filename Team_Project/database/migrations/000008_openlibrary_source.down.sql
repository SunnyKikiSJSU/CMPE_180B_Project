BEGIN;

-- Imported Open Library rows stay as books; they become 'user' rows because
-- the restored books_source_valid does not allow 'openlibrary'.
UPDATE books SET source = 'user' WHERE source = 'openlibrary';

ALTER TABLE books
    DROP CONSTRAINT books_openlibrary_source_complete,
    DROP CONSTRAINT books_source_valid,
    ADD CONSTRAINT books_source_valid
        CHECK (source IN ('seed', 'google', 'user')),
    DROP CONSTRAINT books_openlibrary_id_format,
    DROP CONSTRAINT books_openlibrary_id_unique,
    DROP COLUMN openlibrary_id;

COMMIT;
