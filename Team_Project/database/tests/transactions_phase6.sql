-- Transaction evidence for Phase 6 (BACKEND_PLAN.md §7.1): review + rating,
-- committed once and rolled back once. The statements are the ones
-- PUT /api/books/{bookId}/review runs in one pgx.Tx (UpsertRating, UpsertReview,
-- then the summary read); the summary itself is written by the ratings and
-- reviews triggers (fn_refresh_book_rating_summary).
-- Run on a migrated database (000001-000007):
--   psql -v ON_ERROR_STOP=0 -f database/tests/transactions_phase6.sql
-- It creates a throwaway user and book and deletes both at the end.
\set QUIET on
\pset footer off
SET client_min_messages = warning;

INSERT INTO users (email, password_hash, display_name)
VALUES ('tx6-evidence@example.com', 'not-a-login', 'Transaction evidence');
SELECT user_id AS uid FROM users WHERE email = 'tx6-evidence@example.com' \gset
INSERT INTO books (title, source) VALUES ('Transaction evidence book', 'user')
RETURNING book_id AS bid \gset

\echo '=== State query used after each step ==='
\set state 'SELECT (SELECT rating_value FROM ratings WHERE user_id = :\'uid\' AND book_id = :\'bid\') AS my_rating, (SELECT review_text FROM reviews WHERE user_id = :\'uid\' AND book_id = :\'bid\') AS my_review, s.average_rating, s.rating_count, s.review_count FROM book_rating_summaries s WHERE s.book_id = :\'bid\''
\echo 'New book (summary row created by the books_create_summary trigger):'
:state;

\echo ''
\echo '=== Scenario 3a: review + rating, COMMIT ==='
\echo 'Rate 4 and write a review in one transaction; the triggers refresh the summary.'
BEGIN;
INSERT INTO ratings (user_id, book_id, rating_value) VALUES (:'uid', :'bid', 4)
ON CONFLICT (user_id, book_id) DO UPDATE SET rating_value = EXCLUDED.rating_value, updated_at = now();
INSERT INTO reviews (user_id, book_id, title, review_text) VALUES (:'uid', :'bid', 'Solid', 'A careful, well-paced book.')
ON CONFLICT (user_id, book_id) DO UPDATE SET title = EXCLUDED.title, review_text = EXCLUDED.review_text, updated_at = now();
COMMIT;
:state;

\echo ''
\echo '=== Scenario 3b: review + rating, ROLLBACK ==='
\echo 'Change the rating to 1, then the review step fails (blank text, CHECK reviews_text_not_blank).'
BEGIN;
INSERT INTO ratings (user_id, book_id, rating_value) VALUES (:'uid', :'bid', 1)
ON CONFLICT (user_id, book_id) DO UPDATE SET rating_value = EXCLUDED.rating_value, updated_at = now();
\echo 'Inside the transaction, after the rating step (the trigger already rewrote the summary):'
:state;
INSERT INTO reviews (user_id, book_id, title, review_text) VALUES (:'uid', :'bid', NULL, '   ')
ON CONFLICT (user_id, book_id) DO UPDATE SET title = EXCLUDED.title, review_text = EXCLUDED.review_text, updated_at = now();
ROLLBACK;
\echo 'After ROLLBACK (same as after 3a: rating 4, original review, summary 4.00 / 1 / 1):'
:state;

-- Cleanup: deleting the book cascades to its rating, review, and summary.
DELETE FROM books WHERE book_id = :'bid';
DELETE FROM users WHERE user_id = :'uid';
