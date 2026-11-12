-- Reverts 000006. Imported Google rows stay in books; only the source
-- metadata columns and their constraints are removed.

ALTER TABLE books
    DROP CONSTRAINT IF EXISTS books_published_date_precision_matches,
    DROP CONSTRAINT IF EXISTS books_published_date_precision_valid,
    DROP CONSTRAINT IF EXISTS books_google_source_complete,
    DROP CONSTRAINT IF EXISTS books_source_valid;

ALTER TABLE books
    DROP COLUMN IF EXISTS imported_at,
    DROP COLUMN IF EXISTS published_date_precision,
    DROP COLUMN IF EXISTS source;
