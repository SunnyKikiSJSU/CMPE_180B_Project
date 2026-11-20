BEGIN;

-- Restore the 000003 definition verbatim (no summary row lock). Triggers
-- reference the function by name, so this swaps the behavior back.

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

    INSERT INTO book_rating_summaries
        (book_id, average_rating, rating_count, review_count, updated_at)
    SELECT
        p_book_id,
        coalesce(round(avg(rating_value)::numeric, 2), 0),
        count(*),
        (SELECT count(*) FROM reviews WHERE book_id = p_book_id),
        now()
    FROM ratings
    WHERE book_id = p_book_id
    ON CONFLICT (book_id) DO UPDATE
        SET average_rating = EXCLUDED.average_rating,
            rating_count   = EXCLUDED.rating_count,
            review_count   = EXCLUDED.review_count,
            updated_at     = now();
END;
$$;

COMMIT;
