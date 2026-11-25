-- Bulk data for the index benchmarks (database/performance/README.md).
-- Run only on a disposable database that has migrations 000001-000008 and
-- nothing else: it disables triggers while loading (session_replication_role
-- needs a superuser), then writes the summaries the triggers would keep.
--
-- Shape: 2,000 readers with ordinary activity, one heavy reader
-- (perf-heavy@example.com) with thousands of rows in every table, 50,000
-- books, and one popular book that every reader rated. setseed() makes the
-- data the same on every run.
\set ON_ERROR_STOP on
\set QUIET on

-- Refuse to run anywhere that already has data (for example RDS or the
-- development database).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM books) OR EXISTS (SELECT 1 FROM users) THEN
        RAISE EXCEPTION 'database % already has data; run this only on an empty, migrated benchmark database', current_database();
    END IF;
END $$;

SELECT setseed(0.180);

-- Users go in with triggers on, so each gets a profile and four status shelves.
INSERT INTO users (email, password_hash, display_name)
SELECT format('perf-%s@example.com', g), 'not-a-login', format('Reader %s', g)
FROM generate_series(1, 2000) g;
INSERT INTO users (email, password_hash, display_name)
VALUES ('perf-heavy@example.com', 'not-a-login', 'Heavy reader');

SET session_replication_role = replica;

CREATE TEMP TABLE words AS
SELECT row_number() OVER () AS n, w
FROM unnest(string_to_array(
    'dragon river castle winter secret garden empire ocean shadow light ' ||
    'storm queen king forest mountain journey letter memory island star ' ||
    'night morning silver golden glass iron stone fire water wind ' ||
    'library city village kingdom war peace love loss friend enemy ' ||
    'mystery murder detective science history poetry travel ancient future machine ' ||
    'robot planet galaxy ship sailor soldier doctor teacher child mother ' ||
    'father sister brother daughter son house road bridge tower door ' ||
    'clock song dance painter music voice silence dream truth lie ' ||
    'hunger harvest summer autumn spring rain snow desert jungle valley ' ||
    'apple bread candle mirror needle paper ribbon saddle thread velvet', ' ')) w;

INSERT INTO authors (name)
SELECT format('Perf Author %s', g) FROM generate_series(1, 5000) g;
INSERT INTO categories (name)
SELECT format('Perf Category %s', g) FROM generate_series(1, 50) g;

-- Titles and descriptions from the word list, so full-text search has
-- realistic, repeated terms.
INSERT INTO books (title, description, page_count, published_date, published_date_precision, source)
SELECT
    initcap((SELECT string_agg(w, ' ') FROM words WHERE n IN (1 + (g * 7) % 100, 1 + (g * 13) % 100, 1 + (g * 29) % 100))),
    (SELECT string_agg(w, ' ') FROM words WHERE n IN (
        1 + (g * 3) % 100, 1 + (g * 11) % 100, 1 + (g * 17) % 100, 1 + (g * 23) % 100,
        1 + (g * 31) % 100, 1 + (g * 37) % 100, 1 + (g * 41) % 100, 1 + (g * 43) % 100)),
    100 + (g * 7919) % 700,
    date '1950-01-01' + (g * 37) % 27000,
    'day',
    'seed'
FROM generate_series(1, 50000) g;
INSERT INTO book_rating_summaries (book_id, average_rating, rating_count, review_count)
SELECT book_id, 0, 0, 0 FROM books ON CONFLICT (book_id) DO NOTHING;

CREATE TEMP TABLE b AS SELECT row_number() OVER (ORDER BY book_id) AS n, book_id, page_count FROM books;
CREATE UNIQUE INDEX ON b (n);
CREATE TEMP TABLE a AS SELECT row_number() OVER (ORDER BY author_id) AS n, author_id FROM authors WHERE name LIKE 'Perf Author %';
CREATE TEMP TABLE c AS SELECT row_number() OVER (ORDER BY category_id) AS n, category_id FROM categories WHERE name LIKE 'Perf Category %';
CREATE TEMP TABLE u AS SELECT row_number() OVER (ORDER BY user_id) AS n, user_id FROM users WHERE email LIKE 'perf-%' AND email <> 'perf-heavy@example.com';
SELECT user_id AS heavy FROM users WHERE email = 'perf-heavy@example.com' \gset

-- One or two authors and one to three categories per book.
INSERT INTO book_authors (book_id, author_id)
SELECT b.book_id, a.author_id FROM b JOIN a ON a.n = 1 + (b.n * 7) % 5000
UNION
SELECT b.book_id, a.author_id FROM b JOIN a ON a.n = 1 + (b.n * 11) % 5000 WHERE b.n % 3 = 0;
INSERT INTO book_categories (book_id, category_id)
SELECT DISTINCT b.book_id, c.category_id
FROM b CROSS JOIN generate_series(0, 2) k
JOIN c ON c.n = 1 + (b.n + k * 17) % 50
WHERE k <= b.n % 3;

-- Ordinary readers: 50 books each in progress and on a status shelf, 50
-- ratings, 20 reviews, 100 history events. The heavy reader has 10,000
-- books, 10,000 ratings, 5,000 reviews, and 30,000 history events.
CREATE TEMP TABLE activity AS
SELECT u.user_id, b.book_id, b.page_count, (random() * 400)::int AS age_days
FROM u CROSS JOIN generate_series(1, 50) k
JOIN b ON b.n = 1 + (u.n * 997 + k * 7919) % 50000
UNION ALL
SELECT :'heavy'::uuid, b.book_id, b.page_count, (random() * 400)::int
FROM b WHERE b.n <= 10000;

INSERT INTO reading_progress (user_id, book_id, current_page, percent_complete, status, updated_at, version)
SELECT user_id, book_id, cp, round(cp::numeric / page_count * 100, 2),
       CASE WHEN cp = 0 THEN 'want_to_read' WHEN cp = page_count THEN 'completed' ELSE 'reading' END,
       now() - make_interval(days => age_days, secs => random() * 86400), 1
FROM (SELECT *, (random() * page_count)::int AS cp FROM activity) x;

INSERT INTO user_shelf_items (shelf_id, book_id, added_at)
SELECT s.shelf_id, rp.book_id, rp.updated_at
FROM reading_progress rp
JOIN shelves s ON s.user_id = rp.user_id AND s.shelf_type = CASE rp.status
    WHEN 'want_to_read' THEN 'want_to_read' WHEN 'completed' THEN 'finished' ELSE 'currently_reading' END;

INSERT INTO ratings (user_id, book_id, rating_value, created_at, updated_at)
SELECT user_id, book_id, 1 + (random() * 4)::int, t, t
FROM (SELECT user_id, book_id, now() - make_interval(days => age_days) AS t FROM activity) x;

-- The popular book: every ordinary reader rated it.
SELECT book_id AS popular FROM b WHERE n = 25000 \gset
INSERT INTO ratings (user_id, book_id, rating_value)
SELECT u.user_id, :'popular'::uuid, 1 + (u.n % 5) FROM u
ON CONFLICT (user_id, book_id) DO NOTHING;

INSERT INTO reviews (user_id, book_id, review_text, created_at, updated_at)
SELECT user_id, book_id, 'Benchmark review.', t, t
FROM (
    SELECT user_id, book_id, now() - make_interval(days => age_days) AS t,
           row_number() OVER (PARTITION BY user_id ORDER BY book_id) AS k
    FROM activity
) x
WHERE k <= CASE WHEN user_id = :'heavy'::uuid THEN 5000 ELSE 20 END;

INSERT INTO reading_history (user_id, book_id, event_type, from_status, to_status, created_at)
SELECT a.user_id, a.book_id, 'progress_updated', NULL, 'reading',
       now() - make_interval(days => a.age_days, secs => k * 3600)
FROM activity a
CROSS JOIN generate_series(1, 3) k
WHERE a.user_id = :'heavy'::uuid OR k <= 2;

SET session_replication_role = origin;

-- What the summary triggers would have written.
UPDATE book_rating_summaries s
SET average_rating = coalesce(r.avg, 0), rating_count = coalesce(r.cnt, 0), review_count = coalesce(v.cnt, 0)
FROM books bk
LEFT JOIN (SELECT book_id, round(avg(rating_value), 2) AS avg, count(*) AS cnt FROM ratings GROUP BY book_id) r USING (book_id)
LEFT JOIN (SELECT book_id, count(*) AS cnt FROM reviews GROUP BY book_id) v USING (book_id)
WHERE s.book_id = bk.book_id;

VACUUM ANALYZE;

\echo 'Rows loaded:'
SELECT 'books' AS "table", count(*) AS "rows" FROM books
UNION ALL SELECT 'users', count(*) FROM users
UNION ALL SELECT 'book_authors', count(*) FROM book_authors
UNION ALL SELECT 'book_categories', count(*) FROM book_categories
UNION ALL SELECT 'reading_progress', count(*) FROM reading_progress
UNION ALL SELECT 'user_shelf_items', count(*) FROM user_shelf_items
UNION ALL SELECT 'ratings', count(*) FROM ratings
UNION ALL SELECT 'reviews', count(*) FROM reviews
UNION ALL SELECT 'reading_history', count(*) FROM reading_history;
