BEGIN;

-- Fixes a lost update in the rating summary (found while verifying Phase 6).
--
-- fn_refresh_book_rating_summary recomputed average_rating, rating_count, and
-- review_count from ratings/reviews and upserted book_rating_summaries. Under
-- READ COMMITTED, two transactions that rate the same book at the same time
-- each computed the aggregate from a snapshot taken before the other one
-- committed; the second upsert then waited for the first one's row lock and
-- wrote its stale values. Two committed ratings could leave rating_count = 1
-- (reproduced: 2 ratings with average 3.00, summary 1 rating with average 1.00).
--
-- The function now takes the summary row lock FIRST (creating the row if it is
-- missing), and only then reads ratings and reviews. In a PL/pgSQL function
-- each statement takes a new snapshot under READ COMMITTED, so the aggregate
-- runs after the lock is granted and sees every committed rating and review.
-- Writers on the same book are serialized; different books do not block.

CREATE OR REPLACE FUNCTION fn_refresh_book_rating_summary(p_book_id uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    -- A cascading book delete can remove ratings after the book row is gone.
    -- Do not recreate a summary that would violate the book foreign key.
    IF NOT EXISTS (SELECT 1 FROM books WHERE book_id = p_book_id) THEN
        RETURN;
    END IF;

    -- Every book gets a summary row on insert (books_create_summary); create it
    -- here too so the lock below always has a row to take.
    INSERT INTO book_rating_summaries
        (book_id, average_rating, rating_count, review_count)
    VALUES (p_book_id, 0, 0, 0)
    ON CONFLICT (book_id) DO NOTHING;

    -- Serialize summary writers for this book before reading the aggregates.
    PERFORM 1
    FROM book_rating_summaries
    WHERE book_id = p_book_id
    FOR UPDATE;

    UPDATE book_rating_summaries s
    SET average_rating = agg.average_rating,
        rating_count   = agg.rating_count,
        review_count   = (SELECT count(*) FROM reviews WHERE book_id = p_book_id),
        updated_at     = now()
    FROM (
        SELECT
            coalesce(round(avg(rating_value)::numeric, 2), 0) AS average_rating,
            count(*) AS rating_count
        FROM ratings
        WHERE book_id = p_book_id
    ) agg
    WHERE s.book_id = p_book_id;
END;
$$;

-- Repair summaries that the race already left stale.
UPDATE book_rating_summaries s
SET average_rating = coalesce(
        (SELECT round(avg(r.rating_value)::numeric, 2) FROM ratings r WHERE r.book_id = s.book_id), 0),
    rating_count   = (SELECT count(*) FROM ratings r WHERE r.book_id = s.book_id),
    review_count   = (SELECT count(*) FROM reviews v WHERE v.book_id = s.book_id),
    updated_at     = now();

COMMIT;
