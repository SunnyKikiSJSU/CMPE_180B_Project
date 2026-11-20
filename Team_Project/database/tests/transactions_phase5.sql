-- Transaction evidence for Phase 5 (BACKEND_PLAN.md §7.1): shelf move and
-- progress completion, each committed once and rolled back once.
-- Run on a migrated and seeded database:
--   psql -v ON_ERROR_STOP=0 -f database/tests/transactions_phase5.sql
-- It creates a throwaway user and deletes it at the end (seed data unchanged).
\set QUIET on
\pset footer off
SET client_min_messages = warning;

-- Setup: a demo user (four status shelves come from the users_create_defaults
-- trigger) and a seeded book with a known page count.
INSERT INTO users (email, password_hash, display_name)
VALUES ('tx-evidence@example.com', 'not-a-login', 'Transaction evidence');
SELECT user_id AS uid FROM users WHERE email = 'tx-evidence@example.com' \gset
SELECT book_id AS bid, page_count AS pages FROM books
WHERE source = 'seed' AND page_count IS NOT NULL ORDER BY title LIMIT 1 \gset
SELECT shelf_id AS want FROM shelves WHERE user_id = :'uid' AND shelf_type = 'want_to_read' \gset
SELECT shelf_id AS reading FROM shelves WHERE user_id = :'uid' AND shelf_type = 'currently_reading' \gset
SELECT shelf_id AS finished FROM shelves WHERE user_id = :'uid' AND shelf_type = 'finished' \gset

\echo '=== State query used after each step ==='
\set state 'SELECT rp.status, rp.current_page, rp.percent_complete, rp.version, (SELECT string_agg(s.shelf_type, \',\' ORDER BY s.shelf_type) FROM user_shelf_items i JOIN shelves s USING (shelf_id) WHERE s.user_id = rp.user_id AND i.book_id = rp.book_id) AS shelves, (SELECT count(*) FROM reading_history h WHERE h.user_id = rp.user_id AND h.book_id = rp.book_id) AS history_rows FROM reading_progress rp WHERE rp.user_id = :\'uid\' AND rp.book_id = :\'bid\''

\echo ''
\echo '=== Scenario 1a: shelf move, COMMIT ==='
\echo 'Add to Want to read, then move to Currently reading in one transaction.'
BEGIN;
SELECT status FROM fn_move_book_to_shelf(:'uid', :'bid', :'want');
SELECT status FROM fn_move_book_to_shelf(:'uid', :'bid', :'reading');
COMMIT;
:state;

\echo ''
\echo '=== Scenario 1b: shelf move, ROLLBACK ==='
\echo 'Move to Finished, then a step fails (unknown shelf): the whole transaction rolls back.'
BEGIN;
SELECT status FROM fn_move_book_to_shelf(:'uid', :'bid', :'finished');
\echo 'Inside the transaction, before the failure:'
:state;
SELECT status FROM fn_move_book_to_shelf(:'uid', :'bid', gen_random_uuid());
ROLLBACK;
\echo 'After ROLLBACK (same as after 1a: still reading, Finished undone, no new history):'
:state;

\echo ''
\echo '=== Scenario 2a: progress completion, COMMIT ==='
\echo 'Read to page 10, then to the last page: completed, moved to Finished, book_completed event.'
BEGIN;
SELECT status, current_page FROM fn_update_reading_progress(:'uid', :'bid', 10, NULL, NULL);
SELECT status, current_page FROM fn_update_reading_progress(:'uid', :'bid', :pages, NULL, NULL);
COMMIT;
:state;
SELECT event_type, from_status, to_status FROM reading_history
WHERE user_id = :'uid' AND book_id = :'bid' ORDER BY created_at DESC, event_type LIMIT 1;

\echo ''
\echo '=== Scenario 2b: progress update, ROLLBACK ==='
\echo 'Restart at page 5 (status reading), then a page beyond the book fails: everything rolls back.'
BEGIN;
SELECT status, current_page FROM fn_update_reading_progress(:'uid', :'bid', 5, 'reading', NULL);
SELECT status, current_page FROM fn_update_reading_progress(:'uid', :'bid', :pages + 1, NULL, NULL);
ROLLBACK;
\echo 'After ROLLBACK (same as after 2a: completed, version unchanged):'
:state;

\echo ''
\echo '=== Scenario 2c: stale version is rejected (BM001) ==='
SELECT version AS v FROM reading_progress WHERE user_id = :'uid' AND book_id = :'bid' \gset
SELECT status FROM fn_update_reading_progress(:'uid', :'bid', 1, NULL, :v - 1);
:state;

-- Cleanup: deleting the user cascades to progress, shelves, and history.
DELETE FROM users WHERE user_id = :'uid';
