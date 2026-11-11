-- Report queries q03-q10.
--
-- The reading-history report matches the existing frontend ReadingReport type.
-- Popularity and recommendations pre-aggregate in CTEs so joining multiple
-- activity tables does not multiply rows and inflate counts.

-- name: ReadingReportSummary :one
WITH params AS (
    SELECT sqlc.arg('user_id')::uuid AS user_id
),
completed AS (
    SELECT
        count(*)::int AS books_read,
        coalesce(sum(b.page_count), 0)::int AS completed_pages
    FROM reading_progress rp
    JOIN books b ON b.book_id = rp.book_id
    CROSS JOIN params p
    WHERE rp.user_id = p.user_id
      AND rp.status = 'completed'
),
current_pages AS (
    SELECT coalesce(sum(rp.current_page), 0)::int AS pages
    FROM reading_progress rp
    CROSS JOIN params p
    WHERE rp.user_id = p.user_id
      AND rp.status IN ('reading', 'paused')
)
SELECT
    c.books_read,
    (c.completed_pages + cp.pages)::int AS pages_read
FROM completed c
CROSS JOIN current_pages cp;

-- name: FavoriteCategories :many
WITH params AS (
    SELECT sqlc.arg('user_id')::uuid AS user_id
)
SELECT
    c.category_id,
    c.name,
    count(*)::int AS book_count
FROM reading_progress rp
JOIN book_categories bc ON bc.book_id = rp.book_id
JOIN categories c ON c.category_id = bc.category_id
LEFT JOIN ratings rt
    ON rt.user_id = rp.user_id
   AND rt.book_id = rp.book_id
CROSS JOIN params p
WHERE rp.user_id = p.user_id
  AND (rp.status = 'completed' OR rt.rating_value >= 4)
GROUP BY c.category_id, c.name
ORDER BY book_count DESC, c.name
LIMIT sqlc.arg('limit')::int;

-- name: RecentReadingActivity :many
SELECT
    rp.book_id,
    rp.status,
    rp.current_page,
    coalesce(b.page_count, 0)::int AS total_pages,
    rp.updated_at
FROM reading_progress rp
JOIN books b ON b.book_id = rp.book_id
WHERE rp.user_id = sqlc.arg('user_id')
ORDER BY rp.updated_at DESC
LIMIT sqlc.arg('limit')::int;

-- name: UserReadingStats :one
WITH params AS (
    SELECT sqlc.arg('user_id')::uuid AS user_id
),
completed AS (
    SELECT
        count(*)::int AS books_completed,
        coalesce(sum(b.page_count), 0)::int AS completed_pages
    FROM reading_progress rp
    JOIN books b ON b.book_id = rp.book_id
    CROSS JOIN params p
    WHERE rp.user_id = p.user_id
      AND rp.status = 'completed'
),
in_progress AS (
    SELECT
        count(*)::int AS books_in_progress,
        coalesce(sum(rp.current_page), 0)::int AS current_pages
    FROM reading_progress rp
    CROSS JOIN params p
    WHERE rp.user_id = p.user_id
      AND rp.status IN ('reading', 'paused')
),
rating_stats AS (
    SELECT
        count(*)::int AS ratings_given,
        coalesce(round(avg(rt.rating_value)::numeric, 2), 0) AS average_rating
    FROM ratings rt
    CROSS JOIN params p
    WHERE rt.user_id = p.user_id
),
review_stats AS (
    SELECT count(*)::int AS reviews_written
    FROM reviews r
    CROSS JOIN params p
    WHERE r.user_id = p.user_id
)
SELECT
    c.books_completed,
    ip.books_in_progress,
    (c.completed_pages + ip.current_pages)::int AS pages_read,
    rs.ratings_given,
    rs.average_rating,
    rv.reviews_written
FROM completed c
CROSS JOIN in_progress ip
CROSS JOIN rating_stats rs
CROSS JOIN review_stats rv;

-- name: PopularBooks :many
WITH shelf_counts AS (
    SELECT
        usi.book_id,
        count(*)::int AS shelf_adds
    FROM user_shelf_items usi
    WHERE usi.added_at >= now() - (sqlc.arg('days')::int * interval '1 day')
    GROUP BY usi.book_id
),
rating_counts AS (
    SELECT
        rt.book_id,
        count(*)::int AS ratings,
        round(avg(rt.rating_value)::numeric, 2) AS average_rating
    FROM ratings rt
    WHERE rt.created_at >= now() - (sqlc.arg('days')::int * interval '1 day')
    GROUP BY rt.book_id
)
SELECT
    b.book_id,
    b.title,
    b.cover_url,
    (
        SELECT a.name
        FROM book_authors ba
        JOIN authors a ON a.author_id = ba.author_id
        WHERE ba.book_id = b.book_id
        ORDER BY a.name
        LIMIT 1
    )::text AS primary_author,
    coalesce(sc.shelf_adds, 0)::int AS shelf_adds,
    coalesce(rc.ratings, 0)::int AS ratings,
    coalesce(rc.average_rating, 0)::numeric(3, 2) AS average_rating,
    (coalesce(sc.shelf_adds, 0) * 2 + coalesce(rc.ratings, 0))::int AS score
FROM books b
LEFT JOIN shelf_counts sc ON sc.book_id = b.book_id
LEFT JOIN rating_counts rc ON rc.book_id = b.book_id
WHERE coalesce(sc.shelf_adds, 0) > 0
   OR coalesce(rc.ratings, 0) > 0
ORDER BY score DESC, b.title
LIMIT sqlc.arg('limit')::int;

-- name: TopRatedBooks :many
SELECT
    b.book_id,
    b.title,
    b.cover_url,
    (
        SELECT a.name
        FROM book_authors ba
        JOIN authors a ON a.author_id = ba.author_id
        WHERE ba.book_id = b.book_id
        ORDER BY a.name
        LIMIT 1
    )::text AS primary_author,
    s.average_rating,
    s.rating_count
FROM book_rating_summaries s
JOIN books b ON b.book_id = s.book_id
WHERE s.rating_count >= sqlc.arg('min_ratings')::int
ORDER BY s.average_rating DESC, s.rating_count DESC, b.title
LIMIT sqlc.arg('limit')::int;

-- name: RecommendedBooks :many
WITH params AS (
    SELECT sqlc.arg('user_id')::uuid AS user_id
),
preferred_categories AS (
    SELECT
        bc.category_id,
        count(*)::int AS weight
    FROM reading_progress rp
    JOIN book_categories bc ON bc.book_id = rp.book_id
    LEFT JOIN ratings rt
        ON rt.user_id = rp.user_id
       AND rt.book_id = rp.book_id
    CROSS JOIN params p
    WHERE rp.user_id = p.user_id
      AND (rp.status = 'completed' OR rt.rating_value >= 4)
    GROUP BY bc.category_id
),
read_books AS (
    SELECT rp.book_id
    FROM reading_progress rp
    CROSS JOIN params p
    WHERE rp.user_id = p.user_id
),
candidates AS (
    SELECT
        bc.book_id,
        sum(pc.weight)::int AS score
    FROM book_categories bc
    JOIN preferred_categories pc ON pc.category_id = bc.category_id
    WHERE NOT EXISTS (
        SELECT 1
        FROM read_books rb
        WHERE rb.book_id = bc.book_id
    )
    GROUP BY bc.book_id
)
SELECT
    b.book_id,
    b.title,
    b.cover_url,
    (
        SELECT a.name
        FROM book_authors ba
        JOIN authors a ON a.author_id = ba.author_id
        WHERE ba.book_id = b.book_id
        ORDER BY a.name
        LIMIT 1
    )::text AS primary_author,
    sc.score,
    coalesce(s.average_rating, 0)::numeric(3, 2) AS average_rating
FROM candidates sc
JOIN books b ON b.book_id = sc.book_id
LEFT JOIN book_rating_summaries s ON s.book_id = b.book_id
ORDER BY sc.score DESC, coalesce(s.average_rating, 0) DESC, b.title
LIMIT sqlc.arg('limit')::int;

-- name: AuthorRankings :many
SELECT
    a.author_id,
    a.name,
    count(DISTINCT ba.book_id)::int AS book_count,
    count(rt.rating_id)::int AS rating_count,
    coalesce(round(avg(rt.rating_value)::numeric, 2), 0) AS average_rating,
    (
        rank() OVER (
            ORDER BY
                coalesce(avg(rt.rating_value), 0) DESC,
                count(rt.rating_id) DESC
        )
    )::int AS ranking
FROM authors a
LEFT JOIN book_authors ba ON ba.author_id = a.author_id
LEFT JOIN ratings rt ON rt.book_id = ba.book_id
GROUP BY a.author_id, a.name
ORDER BY ranking, a.name
LIMIT sqlc.arg('limit')::int;

-- name: CategoryRankings :many
SELECT
    c.category_id,
    c.name,
    count(DISTINCT bc.book_id)::int AS book_count,
    count(rt.rating_id)::int AS rating_count,
    coalesce(round(avg(rt.rating_value)::numeric, 2), 0) AS average_rating,
    (
        rank() OVER (
            ORDER BY
                coalesce(avg(rt.rating_value), 0) DESC,
                count(rt.rating_id) DESC
        )
    )::int AS ranking
FROM categories c
LEFT JOIN book_categories bc ON bc.category_id = c.category_id
LEFT JOIN ratings rt ON rt.book_id = bc.book_id
GROUP BY c.category_id, c.name
ORDER BY ranking, c.name
LIMIT sqlc.arg('limit')::int;

-- name: ReviewActivity :many
WITH params AS (
    SELECT sqlc.arg('days')::int AS days
),
days AS (
    SELECT generate_series(
        date_trunc('day', now()) - (p.days - 1) * interval '1 day',
        date_trunc('day', now()),
        interval '1 day'
    ) AS day
    FROM params p
),
review_counts AS (
    SELECT
        date_trunc('day', created_at) AS day,
        count(*)::int AS reviews
    FROM reviews
    GROUP BY 1
),
rating_counts AS (
    SELECT
        date_trunc('day', created_at) AS day,
        count(*)::int AS ratings
    FROM ratings
    GROUP BY 1
)
SELECT
    d.day::date AS day,
    coalesce(rc.reviews, 0)::int AS reviews,
    coalesce(rt.ratings, 0)::int AS ratings
FROM days d
LEFT JOIN review_counts rc ON rc.day = d.day
LEFT JOIN rating_counts rt ON rt.day = d.day
ORDER BY d.day;

-- name: RatingSummaryByCategory :many
SELECT
    c.category_id,
    c.name,
    count(rt.rating_id)::int AS rating_count,
    coalesce(round(avg(rt.rating_value)::numeric, 2), 0) AS average_rating
FROM categories c
JOIN book_categories bc ON bc.category_id = c.category_id
JOIN ratings rt ON rt.book_id = bc.book_id
GROUP BY c.category_id, c.name
HAVING count(rt.rating_id) > 0
ORDER BY average_rating DESC, rating_count DESC, c.name
LIMIT sqlc.arg('limit')::int;

-- name: RatingSummaryByAuthor :many
SELECT
    a.author_id,
    a.name,
    count(rt.rating_id)::int AS rating_count,
    coalesce(round(avg(rt.rating_value)::numeric, 2), 0) AS average_rating
FROM authors a
JOIN book_authors ba ON ba.author_id = a.author_id
JOIN ratings rt ON rt.book_id = ba.book_id
GROUP BY a.author_id, a.name
HAVING count(rt.rating_id) > 0
ORDER BY average_rating DESC, rating_count DESC, a.name
LIMIT sqlc.arg('limit')::int;
