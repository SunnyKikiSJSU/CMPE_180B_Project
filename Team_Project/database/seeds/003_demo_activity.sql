BEGIN;

-- Demonstration activity: 50 users, 3 custom shelves each, 500 shelf items,
-- 500 progress rows, 300 ratings, 300 reviews, and 500 history events.
--
-- Demo login: reader1@example.com / Password123!
--
-- PostgreSQL cannot compute Argon2id in SQL, so this seed uses bcrypt through
-- pgcrypto. The Go auth service must accept bcrypt for seeded demo accounts and
-- rehash to Argon2id on the first successful login. Users created through the
-- API are hashed with Argon2id directly and never use this path.
--
-- Reproducibility: every timestamp is derived from one fixed anchor
-- (2026-10-31 12:00 UTC), so report output does not change from day to day.
-- Activity spans roughly 2026-09-06 through 2026-10-28. Reports with a
-- rolling window (PopularBooks, ReviewActivity) must be called with a window
-- that reaches back to those dates when capturing evidence.
--
-- Distribution: each user shelves 10 distinct books. Three come from a
-- "popular" pool (books 1-20) and seven are spread over books 21-200, so the
-- popularity and ranking reports have a realistic long tail. Ratings vary by
-- book and by user.

CREATE TEMP TABLE seed_clock ON COMMIT DROP AS
SELECT TIMESTAMPTZ '2026-10-31 12:00:00+00' AS anchor;

CREATE TEMP TABLE seed_demo_users ON COMMIT DROP AS
SELECT
    n,
    ('00000000-0000-4000-8004-' || lpad(n::text, 12, '0'))::uuid AS user_id,
    'reader' || n || '@example.com' AS email,
    'Reader ' || n AS display_name,
    (SELECT anchor FROM seed_clock) - make_interval(days => 120 - n) AS created_at
FROM generate_series(1, 50) AS n;

-- Number seed books by the integer in google_books_id (seed-book-<n>), not by
-- text order, so book n here is book n in 002_reference_data.sql.
CREATE TEMP TABLE seed_demo_books ON COMMIT DROP AS
SELECT
    book_id,
    substring(google_books_id FROM '^seed-book-([0-9]+)$')::int AS n
FROM books
WHERE google_books_id ~ '^seed-book-[0-9]+$';

INSERT INTO users (user_id, email, password_hash, display_name, created_at)
SELECT
    u.user_id,
    u.email,
    crypt('Password123!', gen_salt('bf', 12)),
    u.display_name,
    u.created_at
FROM seed_demo_users u
ON CONFLICT DO NOTHING;

-- The users_create_defaults trigger creates the profile and four status
-- shelves. Fill in the optional profile fields for nicer dashboard data.
UPDATE user_profiles p
SET
    bio = 'Seeded reader profile used for report and dashboard evidence.',
    favorite_genre = (ARRAY[
        'Fantasy',
        'Science Fiction',
        'Romance',
        'Mystery',
        'Literary Fiction'
    ])[1 + ((u.n - 1) % 5)]
FROM seed_demo_users u
WHERE p.user_id = u.user_id
  AND (p.bio IS NULL OR p.bio = '');

-- The trigger stamps default shelves with now(); pin them to the user's
-- creation time so shelf data is reproducible too.
UPDATE shelves s
SET created_at = u.created_at
FROM seed_demo_users u
WHERE s.user_id = u.user_id
  AND s.shelf_type <> 'custom'
  AND s.created_at IS DISTINCT FROM u.created_at;

-- Three custom shelves per user, matching the frontend's shelf examples.
INSERT INTO shelves (shelf_id, user_id, name, shelf_type, created_at)
SELECT
    ('00000000-0000-4000-8005-' || lpad((u.n * 10 + s.idx)::text, 12, '0'))::uuid,
    u.user_id,
    s.name,
    'custom',
    u.created_at + make_interval(days => 1)
FROM seed_demo_users u
CROSS JOIN (VALUES
    (1, 'Quiet mornings'),
    (2, 'A little romance'),
    (3, 'Big ideas')
) AS s(idx, name)
ON CONFLICT DO NOTHING;

-- Ten distinct books per user.
--   rn 1-3:  ((n + 7k) mod 20) + 1           -> popular pool, books 1-20
--   rn 4-10: ((37n + 13k) mod 180) + 21      -> long tail, books 21-200
-- 7 and 13 are coprime with 20 and 180, so the picks never repeat within a
-- user. Status shelves cycle by rn: currently reading (1,5,9), finished
-- (2,6,10), paused (3,7), want to read (4,8).
CREATE TEMP TABLE seed_pairs ON COMMIT DROP AS
WITH picks AS (
    SELECT
        u.user_id,
        u.n AS user_n,
        k AS rn,
        CASE
            WHEN k <= 3 THEN ((u.n + k * 7) % 20) + 1
            ELSE ((u.n * 37 + k * 13) % 180) + 21
        END AS book_n
    FROM seed_demo_users u
    CROSS JOIN generate_series(1, 10) AS k
)
SELECT
    p.user_id,
    p.user_n,
    b.book_id,
    p.book_n,
    p.rn,
    CASE (p.rn % 4)
        WHEN 0 THEN 'want_to_read'
        WHEN 1 THEN 'currently_reading'
        WHEN 2 THEN 'finished'
        ELSE 'paused'
    END AS shelf_type,
    (SELECT anchor FROM seed_clock)
        - make_interval(
            days => 5 * (11 - p.rn) + (p.user_n % 5),
            hours => p.user_n % 24
        ) AS added_at
FROM picks p
JOIN seed_demo_books b ON b.n = p.book_n;

INSERT INTO user_shelf_items (shelf_id, book_id, added_at, moved_at)
SELECT
    s.shelf_id,
    p.book_id,
    p.added_at,
    p.added_at + make_interval(days => 2)
FROM seed_pairs p
JOIN shelves s
    ON s.user_id = p.user_id
   AND s.shelf_type = p.shelf_type
ON CONFLICT DO NOTHING;

-- The validation trigger recomputes percent_complete and, for completed rows,
-- sets current_page to the book's page count.
INSERT INTO reading_progress (
    progress_id,
    user_id,
    book_id,
    current_page,
    status,
    updated_at
)
SELECT
    ('00000000-0000-4000-8009-' || lpad((p.user_n * 100 + p.rn)::text, 12, '0'))::uuid,
    p.user_id,
    p.book_id,
    CASE p.shelf_type
        WHEN 'want_to_read' THEN 0
        WHEN 'currently_reading' THEN greatest(1, (coalesce(b.page_count, 200) * 3) / 10)
        WHEN 'finished' THEN coalesce(b.page_count, 200)
        ELSE greatest(1, coalesce(b.page_count, 200) / 2)
    END,
    CASE p.shelf_type
        WHEN 'want_to_read' THEN 'want_to_read'
        WHEN 'currently_reading' THEN 'reading'
        WHEN 'finished' THEN 'completed'
        ELSE 'paused'
    END,
    p.added_at + make_interval(days => 2)
FROM seed_pairs p
JOIN books b ON b.book_id = p.book_id
ON CONFLICT (user_id, book_id) DO NOTHING;

-- Six rated and reviewed books per user: every shelved book the user has
-- started (all statuses except want to read) with rn <= 7.
CREATE TEMP TABLE seed_review_pairs ON COMMIT DROP AS
SELECT
    user_id,
    user_n,
    book_id,
    book_n,
    rn,
    added_at + make_interval(days => 3) AS reviewed_at,
    least(5, greatest(1, 1 + (book_n % 4) + ((user_n + rn) % 3)))::smallint
        AS rating_value,
    row_number() OVER (ORDER BY user_n, rn) AS global_n
FROM seed_pairs
WHERE rn <= 7
  AND shelf_type <> 'want_to_read';

INSERT INTO ratings (
    rating_id,
    user_id,
    book_id,
    rating_value,
    created_at,
    updated_at
)
SELECT
    ('00000000-0000-4000-8006-' || lpad(global_n::text, 12, '0'))::uuid,
    user_id,
    book_id,
    rating_value,
    reviewed_at,
    reviewed_at
FROM seed_review_pairs
ON CONFLICT DO NOTHING;

INSERT INTO reviews (
    review_id,
    user_id,
    book_id,
    title,
    review_text,
    created_at,
    updated_at
)
SELECT
    ('00000000-0000-4000-8007-' || lpad(global_n::text, 12, '0'))::uuid,
    user_id,
    book_id,
    'Seed review ' || global_n,
    CASE
        WHEN rating_value >= 4 THEN
            'Loved this one. A seeded review used to demonstrate rating '
            || 'summaries, review lists, and review-activity reports.'
        WHEN rating_value = 3 THEN
            'Solid but uneven. A seeded review used to demonstrate rating '
            || 'summaries, review lists, and review-activity reports.'
        ELSE
            'Not for me. A seeded review used to demonstrate rating '
            || 'summaries, review lists, and review-activity reports.'
    END,
    reviewed_at + make_interval(hours => 2),
    reviewed_at + make_interval(hours => 2)
FROM seed_review_pairs
ON CONFLICT DO NOTHING;

-- One immutable history event per progress pair.
INSERT INTO reading_history (
    history_id,
    user_id,
    book_id,
    event_type,
    from_status,
    to_status,
    created_at
)
SELECT
    ('00000000-0000-4000-8008-' || lpad(history_n::text, 12, '0'))::uuid,
    p.user_id,
    p.book_id,
    'status_changed',
    NULL,
    rp.status,
    p.added_at + make_interval(days => 2)
FROM (
    SELECT
        user_id,
        book_id,
        added_at,
        row_number() OVER (ORDER BY user_n, rn) AS history_n
    FROM seed_pairs
) p
JOIN reading_progress rp
    ON rp.user_id = p.user_id
   AND rp.book_id = p.book_id
ON CONFLICT DO NOTHING;

-- Rebuild cached summaries after the bulk rating/review inserts.
SELECT fn_refresh_book_rating_summary(book_id) FROM books;

COMMIT;
