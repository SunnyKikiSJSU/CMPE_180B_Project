BEGIN;

-- Rebuild cached rating/review summaries after any bulk seed or import. This
-- is idempotent and safe to run even when no ratings changed.
SELECT fn_refresh_book_rating_summary(book_id) FROM books;

COMMIT;
