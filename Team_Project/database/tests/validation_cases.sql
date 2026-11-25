-- Database validation cases (BACKEND_PLAN.md §9.2): constraints, triggers,
-- and functions, each with the expected and the actual result.
--
-- Run on a migrated database (000001-000008); seeds are not needed:
--   psql "$DATABASE_URL" -f database/tests/validation_cases.sql
--   docker exec -i bm-pg psql -U book_app -d book_metaverse < database/tests/validation_cases.sql
--
-- Everything runs in one transaction that is rolled back at the end, so the
-- database is unchanged. Each error case runs in its own subtransaction
-- (a PL/pgSQL EXCEPTION block), so one rejected statement does not abort the
-- others. The script exits non-zero if any case fails.
\set ON_ERROR_STOP on
\set QUIET on
\pset footer off
SET client_min_messages = warning;

BEGIN;

CREATE TEMP TABLE results (
    case_no  int PRIMARY KEY,
    name     text NOT NULL,
    expected text NOT NULL,
    actual   text NOT NULL,
    pass     boolean NOT NULL
) ON COMMIT DROP;

-- Named IDs for the throwaway rows, so dynamic SQL can refer to them.
CREATE TEMP TABLE ids (k text PRIMARY KEY, v uuid NOT NULL) ON COMMIT DROP;
CREATE FUNCTION pg_temp.id(p_key text) RETURNS uuid
LANGUAGE sql STABLE AS $$ SELECT v FROM ids WHERE k = p_key $$;

CREATE FUNCTION pg_temp.record(p_no int, p_name text, p_expected text, p_actual text)
RETURNS void LANGUAGE sql AS $$
    INSERT INTO results VALUES (p_no, p_name, p_expected, p_actual, p_actual = p_expected)
$$;

-- Runs p_sql and records whether it failed with p_state (and, when given,
-- the constraint or index p_constraint).
CREATE FUNCTION pg_temp.expect_error(p_no int, p_name text, p_sql text, p_state text, p_constraint text DEFAULT NULL)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    v_state text;
    v_constraint text;
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS v_state = RETURNED_SQLSTATE, v_constraint = CONSTRAINT_NAME;
    END;
    PERFORM pg_temp.record(p_no, p_name,
        'rejected ' || p_state || coalesce(' ' || p_constraint, ''),
        CASE WHEN v_state IS NULL THEN 'accepted'
             ELSE 'rejected ' || v_state || CASE WHEN p_constraint IS NULL THEN '' ELSE ' ' || coalesce(nullif(v_constraint, ''), '?') END
        END);
END;
$$;

-- Records the text p_sql returns, or the error it raised.
CREATE FUNCTION pg_temp.expect_value(p_no int, p_name text, p_sql text, p_expected text)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    v_actual text;
BEGIN
    BEGIN
        EXECUTE p_sql INTO v_actual;
    EXCEPTION WHEN OTHERS THEN
        v_actual := 'error ' || SQLSTATE || ': ' || SQLERRM;
    END;
    PERFORM pg_temp.record(p_no, p_name, p_expected, coalesce(v_actual, 'null'));
END;
$$;

-- Setup: two users (the users_create_defaults trigger adds a profile and four
-- status shelves), a book with 300 pages, and a book without a page count.
WITH u AS (
    INSERT INTO users (email, password_hash, display_name) VALUES
        ('validation-a@example.com', 'not-a-login', 'Validation A'),
        ('validation-b@example.com', 'not-a-login', 'Validation B')
    RETURNING user_id, email
)
INSERT INTO ids SELECT CASE email WHEN 'validation-a@example.com' THEN 'ua' ELSE 'ub' END, user_id FROM u;
WITH b AS (
    INSERT INTO books (title, page_count) VALUES
        ('Validation: paged', 300),
        ('Validation: unpaged', NULL)
    RETURNING book_id, page_count
)
INSERT INTO ids SELECT CASE WHEN page_count IS NULL THEN 'unpaged' ELSE 'paged' END, book_id FROM b;

-- The helpers return void; keep their empty result sets out of the report.
\o /dev/null

-- 1-3: users
SELECT pg_temp.expect_error(1, 'Duplicate email rejected',
    $$INSERT INTO users (email, password_hash, display_name) VALUES ('validation-a@example.com', 'x', 'Copy')$$,
    '23505', 'users_email_unique');
SELECT pg_temp.expect_error(2, 'Email with uppercase or spaces rejected (must be normalized)',
    $$INSERT INTO users (email, password_hash, display_name) VALUES (' Mixed@Example.com', 'x', 'Mixed')$$,
    '23514', 'users_email_normalized');
SELECT pg_temp.expect_error(3, 'Blank display name rejected',
    $$INSERT INTO users (email, password_hash, display_name) VALUES ('blank-name@example.com', 'x', '   ')$$,
    '23514', 'users_display_name_not_blank');

-- 4-7: ratings (fn_validate_rating runs before the CHECK constraint)
SELECT pg_temp.expect_error(4, 'Rating 0 rejected',
    $$INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ua'), pg_temp.id('paged'), 0)$$,
    '22023');
SELECT pg_temp.expect_error(5, 'Rating 6 rejected',
    $$INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ua'), pg_temp.id('paged'), 6)$$,
    '22023');
SELECT pg_temp.expect_value(6, 'Rating 5 accepted and summarized',
    $$WITH r AS (INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ua'), pg_temp.id('unpaged'), 5) RETURNING rating_value)
      SELECT 'stored ' || rating_value FROM r$$,
    'stored 5');
SELECT pg_temp.expect_error(7, 'Second rating by the same user for the same book rejected',
    $$INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ua'), pg_temp.id('unpaged'), 3)$$,
    '23505', 'ratings_user_book_unique');

-- 8-9: reviews
INSERT INTO reviews (user_id, book_id, review_text) VALUES (pg_temp.id('ua'), pg_temp.id('unpaged'), 'First review.');
SELECT pg_temp.expect_error(8, 'Second review by the same user for the same book rejected',
    $$INSERT INTO reviews (user_id, book_id, review_text) VALUES (pg_temp.id('ua'), pg_temp.id('unpaged'), 'Again.')$$,
    '23505', 'reviews_user_book_unique');
SELECT pg_temp.expect_error(9, 'Blank review text rejected',
    $$INSERT INTO reviews (user_id, book_id, review_text) VALUES (pg_temp.id('ub'), pg_temp.id('unpaged'), '  ')$$,
    '23514', 'reviews_text_not_blank');

-- 10-13: reading progress (fn_validate_reading_progress knows the page count;
-- the CHECK constraints cover books without one)
SELECT pg_temp.expect_error(10, 'Current page above the book''s page count rejected',
    $$INSERT INTO reading_progress (user_id, book_id, current_page, status) VALUES (pg_temp.id('ua'), pg_temp.id('paged'), 301, 'reading')$$,
    '22023');
SELECT pg_temp.expect_error(11, 'Negative current page rejected',
    $$INSERT INTO reading_progress (user_id, book_id, current_page, status) VALUES (pg_temp.id('ua'), pg_temp.id('unpaged'), -1, 'reading')$$,
    '23514', 'reading_progress_current_page_nonnegative');
SELECT pg_temp.expect_error(12, 'Percentage above 100 rejected',
    $$INSERT INTO reading_progress (user_id, book_id, percent_complete, status) VALUES (pg_temp.id('ua'), pg_temp.id('unpaged'), 150, 'reading')$$,
    '23514', 'reading_progress_percent_range');
SELECT pg_temp.expect_error(13, 'Unknown reading status rejected',
    $$INSERT INTO reading_progress (user_id, book_id, status) VALUES (pg_temp.id('ua'), pg_temp.id('paged'), 'skimming')$$,
    '22023');

-- 14-16: shelves
SELECT pg_temp.expect_error(14, 'Unknown shelf type rejected',
    $$INSERT INTO shelves (user_id, name, shelf_type) VALUES (pg_temp.id('ua'), 'Odd', 'wishlist')$$,
    '23514', 'shelves_type_valid');
SELECT pg_temp.expect_error(15, 'Second "finished" status shelf for one user rejected',
    $$INSERT INTO shelves (user_id, name, shelf_type) VALUES (pg_temp.id('ua'), 'Done too', 'finished')$$,
    '23505', 'shelves_one_per_status_type_idx');
INSERT INTO shelves (user_id, name) VALUES (pg_temp.id('ua'), 'Favourites');
SELECT pg_temp.expect_error(16, 'Duplicate custom shelf name for one user rejected',
    $$INSERT INTO shelves (user_id, name) VALUES (pg_temp.id('ua'), 'Favourites')$$,
    '23505', 'shelves_user_name_unique');

-- 17: foreign keys
SELECT pg_temp.expect_error(17, 'Rating for a book that does not exist rejected',
    $$INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ua'), gen_random_uuid(), 4)$$,
    '23503', 'ratings_book_fk');

-- 18-19: cached summaries kept by triggers
DO $$
DECLARE
    v_before int;
    v_after int;
BEGIN
    INSERT INTO reviews (user_id, book_id, review_text) VALUES (pg_temp.id('ub'), pg_temp.id('paged'), 'Temporary.');
    SELECT review_count INTO v_before FROM book_rating_summaries WHERE book_id = pg_temp.id('paged');
    DELETE FROM reviews WHERE user_id = pg_temp.id('ub') AND book_id = pg_temp.id('paged');
    SELECT review_count INTO v_after FROM book_rating_summaries WHERE book_id = pg_temp.id('paged');
    PERFORM pg_temp.record(18, 'Deleting a review updates review_count',
        'after insert 1, after delete 0', format('after insert %s, after delete %s', v_before, v_after));
END $$;

DO $$
DECLARE
    v_steps text[] := '{}';
    s record;
BEGIN
    INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ua'), pg_temp.id('paged'), 4), (pg_temp.id('ub'), pg_temp.id('paged'), 2);
    SELECT average_rating, rating_count INTO s FROM book_rating_summaries WHERE book_id = pg_temp.id('paged');
    v_steps := v_steps || format('insert 4,2 -> %s/%s', s.average_rating, s.rating_count);
    UPDATE ratings SET rating_value = 5 WHERE user_id = pg_temp.id('ub') AND book_id = pg_temp.id('paged');
    SELECT average_rating, rating_count INTO s FROM book_rating_summaries WHERE book_id = pg_temp.id('paged');
    v_steps := v_steps || format('update 2->5 -> %s/%s', s.average_rating, s.rating_count);
    DELETE FROM ratings WHERE user_id = pg_temp.id('ua') AND book_id = pg_temp.id('paged');
    SELECT average_rating, rating_count INTO s FROM book_rating_summaries WHERE book_id = pg_temp.id('paged');
    v_steps := v_steps || format('delete 4 -> %s/%s', s.average_rating, s.rating_count);
    PERFORM pg_temp.record(19, 'Rating insert, update, and delete keep average/count correct',
        'insert 4,2 -> 3.00/2; update 2->5 -> 4.50/2; delete 4 -> 5.00/1', array_to_string(v_steps, '; '));
END $$;

-- 20: reading_history is append-only
DO $$
DECLARE
    v_update text := 'accepted';
    v_delete text := 'accepted';
BEGIN
    INSERT INTO reading_history (user_id, book_id, event_type, to_status) VALUES (pg_temp.id('ua'), pg_temp.id('paged'), 'status_changed', 'reading');
    BEGIN
        UPDATE reading_history SET to_status = 'completed' WHERE user_id = pg_temp.id('ua');
    EXCEPTION WHEN OTHERS THEN v_update := 'rejected ' || SQLSTATE;
    END;
    BEGIN
        DELETE FROM reading_history WHERE user_id = pg_temp.id('ua');
    EXCEPTION WHEN OTHERS THEN v_delete := 'rejected ' || SQLSTATE;
    END;
    PERFORM pg_temp.record(20, 'Updating or deleting reading_history rejected',
        'update rejected 55000, delete rejected 55000', format('update %s, delete %s', v_update, v_delete));
END $$;

-- 21: fn_move_book_to_shelf keeps one status shelf and leaves custom shelves
DO $$
DECLARE
    v_status text;
    v_status_shelves text;
    v_custom boolean;
BEGIN
    PERFORM fn_move_book_to_shelf(pg_temp.id('ub'), pg_temp.id('paged'), (SELECT shelf_id FROM shelves WHERE user_id = pg_temp.id('ub') AND shelf_type = 'want_to_read'));
    INSERT INTO shelves (user_id, name) VALUES (pg_temp.id('ub'), 'Book club');
    PERFORM fn_move_book_to_shelf(pg_temp.id('ub'), pg_temp.id('paged'), (SELECT shelf_id FROM shelves WHERE user_id = pg_temp.id('ub') AND name = 'Book club'));
    SELECT status INTO v_status FROM fn_move_book_to_shelf(pg_temp.id('ub'), pg_temp.id('paged'), (SELECT shelf_id FROM shelves WHERE user_id = pg_temp.id('ub') AND shelf_type = 'currently_reading'));
    SELECT string_agg(s.shelf_type, ',' ORDER BY s.shelf_type) INTO v_status_shelves
    FROM user_shelf_items i JOIN shelves s USING (shelf_id)
    WHERE s.user_id = pg_temp.id('ub') AND i.book_id = pg_temp.id('paged') AND s.shelf_type <> 'custom';
    SELECT EXISTS (SELECT 1 FROM user_shelf_items i JOIN shelves s USING (shelf_id)
                   WHERE s.user_id = pg_temp.id('ub') AND s.name = 'Book club' AND i.book_id = pg_temp.id('paged')) INTO v_custom;
    PERFORM pg_temp.record(21, 'Moving to a status shelf leaves only that status shelf (custom kept)',
        'status reading; status shelves currently_reading; custom kept true',
        format('status %s; status shelves %s; custom kept %s', v_status, v_status_shelves, v_custom::text));
END $$;

-- 22: defaults created by the users_create_defaults trigger
SELECT pg_temp.expect_value(22, 'Registration creates a profile and the four status shelves',
    $$SELECT format('profile %s; shelves %s',
        EXISTS (SELECT 1 FROM user_profiles WHERE user_id = pg_temp.id('ua'))::text,
        (SELECT string_agg(shelf_type, ',' ORDER BY shelf_type) FROM shelves WHERE user_id = pg_temp.id('ua') AND shelf_type <> 'custom'))$$,
    'profile true; shelves currently_reading,finished,paused,want_to_read');

-- 23: many-to-many authors and categories
DO $$
DECLARE
    v_authors text;
    v_categories text;
BEGIN
    WITH a AS (INSERT INTO authors (name) VALUES ('Validation Author One'), ('Validation Author Two') RETURNING author_id)
    INSERT INTO book_authors (book_id, author_id) SELECT pg_temp.id('paged'), author_id FROM a;
    WITH c AS (INSERT INTO categories (name) VALUES ('Validation Fantasy'), ('Validation Classics') RETURNING category_id)
    INSERT INTO book_categories (book_id, category_id) SELECT pg_temp.id('paged'), category_id FROM c;
    SELECT string_agg(a.name, ' & ' ORDER BY a.name) INTO v_authors
    FROM book_authors ba JOIN authors a USING (author_id) WHERE ba.book_id = pg_temp.id('paged');
    SELECT string_agg(c.name, ' & ' ORDER BY c.name) INTO v_categories
    FROM book_categories bc JOIN categories c USING (category_id) WHERE bc.book_id = pg_temp.id('paged');
    PERFORM pg_temp.record(23, 'A book with two authors and two categories returns both',
        'authors Validation Author One & Validation Author Two; categories Validation Classics & Validation Fantasy',
        format('authors %s; categories %s', v_authors, v_categories));
END $$;

-- 24: deleting a user cascades to everything the user owns
DO $$
DECLARE
    v_before text;
    v_after text;
    v_count_sql text := $q$
        SELECT format('shelves %s, items %s, progress %s, ratings %s, reviews %s, history %s, profile %s',
            (SELECT count(*) FROM shelves WHERE user_id = $1),
            (SELECT count(*) FROM user_shelf_items i JOIN shelves s USING (shelf_id) WHERE s.user_id = $1),
            (SELECT count(*) FROM reading_progress WHERE user_id = $1),
            (SELECT count(*) FROM ratings WHERE user_id = $1),
            (SELECT count(*) FROM reviews WHERE user_id = $1),
            (SELECT count(*) FROM reading_history WHERE user_id = $1),
            (SELECT count(*) FROM user_profiles WHERE user_id = $1))$q$;
BEGIN
    INSERT INTO ratings (user_id, book_id, rating_value) VALUES (pg_temp.id('ub'), pg_temp.id('unpaged'), 1);
    INSERT INTO reviews (user_id, book_id, review_text) VALUES (pg_temp.id('ub'), pg_temp.id('paged'), 'Owned by B.');
    EXECUTE v_count_sql INTO v_before USING pg_temp.id('ub');
    DELETE FROM users WHERE user_id = pg_temp.id('ub');
    EXECUTE v_count_sql INTO v_after USING pg_temp.id('ub');
    PERFORM pg_temp.record(24, 'Deleting a user removes their shelves, progress, ratings, reviews, history',
        'before: shelves 5, items 2, progress 1, ratings 2, reviews 1, history 3, profile 1; after: shelves 0, items 0, progress 0, ratings 0, reviews 0, history 0, profile 0',
        format('before: %s; after: %s', v_before, v_after));
END $$;

-- 25-27: book source and date rules
SELECT pg_temp.expect_error(25, 'A "google" book without google_books_id and imported_at rejected',
    $$INSERT INTO books (title, source) VALUES ('No provenance', 'google')$$,
    '23514', 'books_google_source_complete');
SELECT pg_temp.expect_error(26, 'Unknown books.source rejected',
    $$INSERT INTO books (title, source) VALUES ('Mystery source', 'amazon')$$,
    '23514', 'books_source_valid');
DO $$
DECLARE
    v_no_precision text := 'accepted';
    v_no_date text := 'accepted';
    v_constraint text;
BEGIN
    BEGIN
        INSERT INTO books (title, published_date) VALUES ('Date only', '2020-01-01');
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS v_constraint = CONSTRAINT_NAME;
        v_no_precision := 'rejected ' || SQLSTATE || ' ' || v_constraint;
    END;
    BEGIN
        INSERT INTO books (title, published_date_precision) VALUES ('Precision only', 'year');
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS v_constraint = CONSTRAINT_NAME;
        v_no_date := 'rejected ' || SQLSTATE || ' ' || v_constraint;
    END;
    PERFORM pg_temp.record(27, 'A date without precision, or precision without a date, rejected',
        'date only rejected 23514 books_published_date_precision_matches; precision only rejected 23514 books_published_date_precision_matches',
        format('date only %s; precision only %s', v_no_precision, v_no_date));
END $$;

-- 28: import deduplication (the ImportOpenLibraryBook upsert from
-- database/queries/books.sql) and the ISBN-13 key
DO $$
DECLARE
    v_import text := $q$
        INSERT INTO books (openlibrary_id, isbn13, title, source, imported_at)
        VALUES ($1, $2, 'Validation import', 'openlibrary', now())
        ON CONFLICT (openlibrary_id) DO UPDATE SET openlibrary_id = books.openlibrary_id
        RETURNING book_id$q$;
    v_first uuid;
    v_second uuid;
    v_isbn text := 'accepted';
    v_constraint text;
BEGIN
    EXECUTE v_import INTO v_first USING 'OL999000001W', '9780000000017';
    EXECUTE v_import INTO v_second USING 'OL999000001W', '9780000000017';
    BEGIN
        EXECUTE v_import USING 'OL999000002W', '9780000000017';
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS v_constraint = CONSTRAINT_NAME;
        v_isbn := 'rejected ' || SQLSTATE || ' ' || v_constraint;
    END;
    INSERT INTO ids VALUES ('imported', v_first);
    PERFORM pg_temp.record(28, 'Importing a work twice returns the same book; another work with that ISBN-13 rejected',
        'same book_id true; other work rejected 23505 books_isbn13_unique',
        format('same book_id %s; other work %s', (v_first = v_second)::text, v_isbn));
END $$;

-- 29: UpdateBook (database/queries/books.sql) never edits an imported book
DO $$
DECLARE
    v_rows int;
    v_title text;
BEGIN
    UPDATE books SET title = 'Edited' WHERE book_id = pg_temp.id('imported') AND source NOT IN ('google', 'openlibrary');
    GET DIAGNOSTICS v_rows = ROW_COUNT;
    SELECT title INTO v_title FROM books WHERE book_id = pg_temp.id('imported');
    PERFORM pg_temp.record(29, 'UpdateBook does not change an imported (openlibrary) book',
        'rows updated 0; title Validation import', format('rows updated %s; title %s', v_rows, v_title));
END $$;

-- 30-34: further rules
SELECT pg_temp.expect_error(30, 'An "openlibrary" book without openlibrary_id rejected',
    $$INSERT INTO books (title, source, imported_at) VALUES ('No work ID', 'openlibrary', now())$$,
    '23514', 'books_openlibrary_source_complete');
SELECT pg_temp.expect_error(31, 'Malformed ISBN-13 rejected',
    $$INSERT INTO books (title, isbn13) VALUES ('Bad ISBN', '978-0-00-0')$$,
    '23514', 'books_isbn13_format');
SELECT pg_temp.expect_error(32, 'Zero page count rejected',
    $$INSERT INTO books (title, page_count) VALUES ('No pages', 0)$$,
    '23514', 'books_page_count_positive');
DO $$
DECLARE
    v_progress reading_progress;
BEGIN
    v_progress := fn_update_reading_progress(pg_temp.id('ua'), pg_temp.id('paged'), 299, 'reading', NULL);
    v_progress := fn_update_reading_progress(pg_temp.id('ua'), pg_temp.id('paged'), 300, NULL, v_progress.version);
    PERFORM pg_temp.record(33, 'Reaching the last page completes the book and moves it to Finished',
        'status completed; percent 100.00; shelf finished',
        format('status %s; percent %s; shelf %s', v_progress.status, v_progress.percent_complete,
            (SELECT string_agg(s.shelf_type, ',') FROM user_shelf_items i JOIN shelves s USING (shelf_id)
             WHERE s.user_id = pg_temp.id('ua') AND i.book_id = pg_temp.id('paged') AND s.shelf_type <> 'custom')));
END $$;
SELECT pg_temp.expect_error(34, 'Progress update with a stale version rejected',
    $$SELECT fn_update_reading_progress(pg_temp.id('ua'), pg_temp.id('paged'), 10, 'reading', 1)$$,
    'BM001');

\o
\pset footer on
\echo ''
\echo '=== Validation cases ==='
SELECT case_no AS "#", name, expected, actual, CASE WHEN pass THEN 'PASS' ELSE 'FAIL' END AS result
FROM results ORDER BY case_no;
SELECT count(*) FILTER (WHERE pass) AS passed, count(*) FILTER (WHERE NOT pass) AS failed, count(*) AS total
FROM results;
SELECT count(*) FILTER (WHERE NOT pass) > 0 AS any_failed FROM results \gset

ROLLBACK;

\if :any_failed
\echo 'Some validation cases failed.'
SELECT 1 / 0 AS validation_failed;
\endif
