-- Index benchmarks: each index is timed with the app's own query (from
-- database/queries/, with literal arguments), first with the index and then
-- with it dropped. The drop happens inside a subtransaction that is rolled
-- back, so the database keeps every index. Run after 01_generate_data.sql.
--
-- Timing: EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON), one warm-up run and then
-- the median of 7 runs. "Buffers" is shared hit + read pages for the whole
-- plan (8 KB each); it does not depend on the machine.
\set ON_ERROR_STOP on
\set QUIET on
SET client_min_messages = warning;
-- One worker keeps timings comparable between runs and machines.
SET max_parallel_workers_per_gather = 0;

CREATE TEMP TABLE params (k text PRIMARY KEY, v text NOT NULL);
INSERT INTO params
SELECT 'heavy', user_id::text FROM users WHERE email = 'perf-heavy@example.com'
UNION ALL
SELECT 'heavy_shelf', s.shelf_id::text FROM shelves s JOIN users u USING (user_id)
WHERE u.email = 'perf-heavy@example.com' AND s.shelf_type = 'currently_reading'
UNION ALL
SELECT 'heavy_book', book_id::text FROM (
    SELECT rh.book_id FROM reading_history rh JOIN users u USING (user_id)
    WHERE u.email = 'perf-heavy@example.com' GROUP BY rh.book_id ORDER BY count(*) DESC, rh.book_id LIMIT 1) x
UNION ALL
SELECT 'popular', book_id::text FROM (
    SELECT book_id FROM ratings GROUP BY book_id ORDER BY count(*) DESC, book_id LIMIT 1) x
UNION ALL
SELECT 'author', author_id::text FROM (
    SELECT author_id FROM book_authors GROUP BY author_id ORDER BY count(*) DESC, author_id LIMIT 1) x
UNION ALL
SELECT 'category', category_id::text FROM (
    SELECT category_id FROM book_categories GROUP BY category_id ORDER BY count(*) DESC, category_id LIMIT 1) x;

CREATE TEMP TABLE cases (
    case_no    int PRIMARY KEY,
    index_name text NOT NULL,
    query_name text NOT NULL,
    sql        text NOT NULL
);

INSERT INTO cases VALUES
(1, 'reading_history_user_created_at_idx', 'ListReadingHistory (heavy reader, newest 50)', $q$
SELECT rh.history_id, rh.book_id, b.title AS book_title, rh.event_type, rh.from_status, rh.to_status, rh.created_at
FROM reading_history rh
JOIN books b ON b.book_id = rh.book_id
CROSS JOIN (SELECT NULL::uuid AS book_id) p
WHERE rh.user_id = '{heavy}'
  AND (p.book_id IS NULL OR rh.book_id = p.book_id)
ORDER BY rh.created_at DESC
LIMIT 50$q$),

(2, 'reading_progress_user_updated_at_idx', 'RecentReadingActivity (heavy reader, dashboard, 5 rows)', $q$
SELECT rp.book_id, rp.status, rp.current_page, coalesce(b.page_count, 0)::int AS total_pages, rp.updated_at
FROM reading_progress rp
JOIN books b ON b.book_id = rp.book_id
WHERE rp.user_id = '{heavy}'
ORDER BY rp.updated_at DESC
LIMIT 5$q$),

(3, 'book_authors_author_id_idx', 'ListBooksByAuthor (author page, first 20)', $q$
SELECT b.book_id, b.title, a.name::text AS primary_author,
       coalesce(s.average_rating, 0)::numeric(3, 2) AS average_rating,
       coalesce(s.rating_count, 0)::int AS rating_count,
       (count(*) OVER ())::int AS total_count
FROM books b
JOIN book_authors ba ON ba.book_id = b.book_id
JOIN authors a ON a.author_id = ba.author_id
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
WHERE ba.author_id = '{author}'
ORDER BY b.title
LIMIT 20 OFFSET 0$q$),

(4, 'reading_history_book_created_at_idx', 'ListReadingHistory for one book (heavy reader, book page, 10 rows)', $q$
SELECT rh.history_id, rh.book_id, b.title AS book_title, rh.event_type, rh.from_status, rh.to_status, rh.created_at
FROM reading_history rh
JOIN books b ON b.book_id = rh.book_id
CROSS JOIN (SELECT '{heavy_book}'::uuid AS book_id) p
WHERE rh.user_id = '{heavy}'
  AND (p.book_id IS NULL OR rh.book_id = p.book_id)
ORDER BY rh.created_at DESC
LIMIT 10$q$),

(5, 'authors_name_trgm_idx', 'ListAuthors search (q = "Author 123", first 20)', $q$
WITH params AS (SELECT 'Author 123'::text AS q)
SELECT a.author_id, a.name, count(ba.book_id)::int AS book_count, (count(*) OVER ())::int AS total_count
FROM authors a
LEFT JOIN book_authors ba ON ba.author_id = a.author_id
CROSS JOIN params p
WHERE p.q IS NULL
   OR a.name ILIKE '%' || replace(replace(replace(p.q, '\', '\\'), '%', '\%'), '_', '\_') || '%'
GROUP BY a.author_id, a.name
ORDER BY a.name
LIMIT 20 OFFSET 0$q$),

(6, 'ratings_book_rating_value_idx', 'GetRatingDistribution (most-rated book)', $q$
SELECT rating_value, count(*)::int AS rating_count
FROM ratings
WHERE book_id = '{popular}'
GROUP BY rating_value
ORDER BY rating_value$q$),

(7, 'book_categories_category_id_idx', 'ListBooksByCategory (category page, first 20)', $q$
SELECT b.book_id, b.title,
       coalesce(s.average_rating, 0)::numeric(3, 2) AS average_rating,
       coalesce(s.rating_count, 0)::int AS rating_count,
       (count(*) OVER ())::int AS total_count
FROM books b
JOIN book_categories bc ON bc.book_id = b.book_id
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
WHERE bc.category_id = '{category}'
ORDER BY b.title
LIMIT 20 OFFSET 0$q$),

(8, 'reviews_user_created_at_idx', 'ListMyReviews (heavy reader, first 20, with total count)', $q$
SELECT r.review_id, r.book_id, r.title, r.created_at, b.title AS book_title, rt.rating_value,
       (count(*) OVER ())::int AS total_count
FROM reviews r
JOIN books b ON b.book_id = r.book_id
LEFT JOIN ratings rt ON rt.user_id = r.user_id AND rt.book_id = r.book_id
WHERE r.user_id = '{heavy}'
ORDER BY r.created_at DESC
LIMIT 20 OFFSET 0$q$),

(9, 'user_shelf_items_shelf_added_at_idx', 'ListShelfBooks core (heavy reader''s Currently reading, first 20, with total count)', $q$
SELECT b.book_id, b.title, usi.added_at, rp.status, (count(*) OVER ())::int AS total_count
FROM shelves sh
JOIN user_shelf_items usi ON usi.shelf_id = sh.shelf_id
JOIN books b ON b.book_id = usi.book_id
LEFT JOIN reading_progress rp ON rp.user_id = sh.user_id AND rp.book_id = b.book_id
WHERE sh.shelf_id = '{heavy_shelf}' AND sh.user_id = '{heavy}'
ORDER BY usi.added_at DESC, b.title
LIMIT 20 OFFSET 0$q$);

-- Fill in the arguments.
DO $$
DECLARE p record;
BEGIN
    FOR p IN SELECT * FROM params LOOP
        UPDATE cases SET sql = replace(sql, '{' || p.k || '}', p.v);
    END LOOP;
END $$;

CREATE FUNCTION pg_temp.measure(p_sql text, OUT median_ms numeric, OUT buffers bigint, OUT plan text)
LANGUAGE plpgsql AS $$
DECLARE
    j json;
    times numeric[] := '{}';
    line text;
BEGIN
    EXECUTE 'EXPLAIN (ANALYZE, FORMAT JSON) ' || p_sql INTO j;  -- warm-up
    FOR i IN 1..7 LOOP
        EXECUTE 'EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) ' || p_sql INTO j;
        times := times || (j -> 0 ->> 'Execution Time')::numeric;
        buffers := (j -> 0 -> 'Plan' ->> 'Shared Hit Blocks')::bigint + (j -> 0 -> 'Plan' ->> 'Shared Read Blocks')::bigint;
    END LOOP;
    SELECT round(percentile_cont(0.5) WITHIN GROUP (ORDER BY t)::numeric, 3) INTO median_ms FROM unnest(times) t;
    plan := '';
    FOR line IN EXECUTE 'EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF, SUMMARY OFF) ' || p_sql LOOP
        plan := plan || line || E'\n';
    END LOOP;
END;
$$;

CREATE TEMP TABLE results (
    case_no int, index_name text, query_name text,
    before_ms numeric, after_ms numeric, before_buffers bigint, after_buffers bigint,
    before_plan text, after_plan text
);

DO $$
DECLARE
    c record;
    a record;
    b record;
BEGIN
    FOR c IN SELECT * FROM cases ORDER BY case_no LOOP
        a := pg_temp.measure(c.sql);
        BEGIN
            EXECUTE format('DROP INDEX %I', c.index_name);
            b := pg_temp.measure(c.sql);
            -- Undo the drop: leaving this block with an error rolls it back.
            RAISE EXCEPTION USING ERRCODE = 'P0001', MESSAGE = 'restore index';
        EXCEPTION WHEN SQLSTATE 'P0001' THEN
            NULL;
        END;
        INSERT INTO results VALUES (c.case_no, c.index_name, c.query_name,
            b.median_ms, a.median_ms, b.buffers, a.buffers, b.plan, a.plan);
    END LOOP;
END $$;

\pset footer off
\echo '=== Index benchmarks (median of 7 runs, ms; buffers = 8 KB pages) ==='
SELECT case_no AS "#", index_name AS "index", query_name AS "query",
       before_ms AS "without", after_ms AS "with",
       round(before_ms / nullif(after_ms, 0), 1) || 'x' AS "faster",
       before_buffers AS "buffers without", after_buffers AS "buffers with"
FROM results ORDER BY case_no;

\echo ''
\echo '=== Plans ==='
\pset format unaligned
\pset tuples_only on
SELECT format(E'--- %s. %s: %s\n--- WITHOUT the index\n%s--- WITH the index\n%s',
              case_no, index_name, query_name, before_plan, after_plan)
FROM results ORDER BY case_no;

-- Every index must still exist.
\pset format aligned
\pset tuples_only off
SELECT count(*) AS indexes_present FROM pg_indexes
WHERE indexname IN (SELECT index_name FROM cases);
