-- 000006: record where each catalog book came from.
--
-- Book search uses the Google Books API live. A Google result becomes a local
-- books row only when a user acts on it (shelf, rate, review, progress), so
-- every foreign key still points at a local book_id UUID. This migration adds
-- the columns needed to tell those imported rows apart from seed and
-- user-created rows:
--
--   source                    'seed', 'google', or 'user'. Google rows need
--                             attribution and a link back to Google Books.
--   published_date_precision  Google may return only "2004" or "2004-05".
--                             The date is stored as the first day of that
--                             year or month, and this column records how much
--                             of it is real.
--   imported_at               When a Google row was first imported.

ALTER TABLE books
    ADD COLUMN source TEXT NOT NULL DEFAULT 'user',
    ADD COLUMN published_date_precision TEXT,
    ADD COLUMN imported_at TIMESTAMPTZ;

-- Backfill before adding the constraints. Seed books carry synthetic IDs in
-- google_books_id ("seed-book-<n>" and the smoke-test ID); they are not real
-- Google volumes and must never be shown with Google attribution.
UPDATE books
SET source = 'seed'
WHERE google_books_id LIKE 'seed-book-%'
   OR google_books_id LIKE 'smoke-test-%';

-- Any other row that already has a Google volume ID came from Google.
UPDATE books
SET source = 'google',
    imported_at = coalesce(imported_at, now())
WHERE source = 'user'
  AND google_books_id IS NOT NULL;

UPDATE books
SET published_date_precision = 'day'
WHERE published_date IS NOT NULL;

ALTER TABLE books
    ADD CONSTRAINT books_source_valid
        CHECK (source IN ('seed', 'google', 'user')),
    -- A Google row is identified by its volume ID and must record its import.
    ADD CONSTRAINT books_google_source_complete
        CHECK (
            source <> 'google'
            OR (google_books_id IS NOT NULL AND imported_at IS NOT NULL)
        ),
    ADD CONSTRAINT books_published_date_precision_valid
        CHECK (
            published_date_precision IS NULL
            OR published_date_precision IN ('year', 'month', 'day')
        ),
    -- Precision is present exactly when a date is present.
    ADD CONSTRAINT books_published_date_precision_matches
        CHECK ((published_date IS NULL) = (published_date_precision IS NULL));
