BEGIN;

-- ---------------------------------------------------------------------------
-- 1. Optimistic-locking column
-- ---------------------------------------------------------------------------
-- Clients may send the version they last saw. The progress function rejects a
-- stale write instead of silently overwriting another session's update.
ALTER TABLE reading_progress
    ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- ---------------------------------------------------------------------------
-- 2. Normalize duplicate authors, categories, and status shelves
-- ---------------------------------------------------------------------------
-- These tables had no case-insensitive uniqueness before this migration. The
-- rebuild-and-remap approach is safe even when several duplicates exist.

CREATE TEMP TABLE tmp_author_dedupe ON COMMIT DROP AS
SELECT author_id,
       first_value(author_id) OVER (
           PARTITION BY lower(btrim(name))
           ORDER BY author_id
       ) AS keep_id
FROM authors;

CREATE TEMP TABLE tmp_book_author_map ON COMMIT DROP AS
SELECT DISTINCT ba.book_id, d.keep_id
FROM book_authors ba
JOIN tmp_author_dedupe d ON d.author_id = ba.author_id;

DELETE FROM book_authors;

INSERT INTO book_authors (book_id, author_id)
SELECT book_id, keep_id
FROM tmp_book_author_map
ON CONFLICT DO NOTHING;

DELETE FROM authors a
USING tmp_author_dedupe d
WHERE a.author_id = d.author_id
  AND d.author_id <> d.keep_id;

CREATE TEMP TABLE tmp_category_dedupe ON COMMIT DROP AS
SELECT category_id,
       first_value(category_id) OVER (
           PARTITION BY lower(btrim(name))
           ORDER BY category_id
       ) AS keep_id
FROM categories;

CREATE TEMP TABLE tmp_book_category_map ON COMMIT DROP AS
SELECT DISTINCT bc.book_id, d.keep_id
FROM book_categories bc
JOIN tmp_category_dedupe d ON d.category_id = bc.category_id;

DELETE FROM book_categories;

INSERT INTO book_categories (book_id, category_id)
SELECT book_id, keep_id
FROM tmp_book_category_map
ON CONFLICT DO NOTHING;

DELETE FROM categories c
USING tmp_category_dedupe d
WHERE c.category_id = d.category_id
  AND d.category_id <> d.keep_id;

-- A user may have duplicate status shelves from earlier data. Keep the oldest
-- one, merge its items into the kept shelf, and remove the duplicate.
CREATE TEMP TABLE tmp_shelf_dedupe ON COMMIT DROP AS
SELECT shelf_id,
       first_value(shelf_id) OVER (
           PARTITION BY user_id, shelf_type
           ORDER BY created_at, shelf_id
       ) AS keep_id
FROM shelves
WHERE shelf_type <> 'custom';

CREATE TEMP TABLE tmp_shelf_item_map ON COMMIT DROP AS
SELECT usi.book_id,
       d.keep_id AS shelf_id,
       min(usi.added_at) AS added_at,
       max(usi.moved_at) AS moved_at
FROM user_shelf_items usi
JOIN tmp_shelf_dedupe d ON d.shelf_id = usi.shelf_id
GROUP BY usi.book_id, d.keep_id;

DELETE FROM user_shelf_items
WHERE shelf_id IN (SELECT shelf_id FROM tmp_shelf_dedupe);

INSERT INTO user_shelf_items (shelf_id, book_id, added_at, moved_at)
SELECT shelf_id, book_id, added_at, moved_at
FROM tmp_shelf_item_map
ON CONFLICT DO NOTHING;

DELETE FROM shelves s
USING tmp_shelf_dedupe d
WHERE s.shelf_id = d.shelf_id
  AND d.shelf_id <> d.keep_id;

-- A book must be on at most one status shelf. Existing rows may violate this
-- because the old schema did not enforce it. Keep the strongest status
-- (Finished, then Currently reading, then Paused, then Want to read).
CREATE TEMP TABLE tmp_duplicate_status_items ON COMMIT DROP AS
SELECT shelf_id, book_id
FROM (
    SELECT
        usi.shelf_id,
        usi.book_id,
        row_number() OVER (
            PARTITION BY s.user_id, usi.book_id
            ORDER BY
                CASE s.shelf_type
                    WHEN 'finished' THEN 1
                    WHEN 'currently_reading' THEN 2
                    WHEN 'paused' THEN 3
                    ELSE 4
                END,
                usi.added_at
        ) AS rn
    FROM user_shelf_items usi
    JOIN shelves s ON s.shelf_id = usi.shelf_id
    WHERE s.shelf_type <> 'custom'
) ranked
WHERE rn > 1;

DELETE FROM user_shelf_items usi
USING tmp_duplicate_status_items d
WHERE usi.shelf_id = d.shelf_id
  AND usi.book_id = d.book_id;

-- Case-insensitive uniqueness now matches the UpsertAuthor/UpsertCategory
-- queries used by the catalog importer.
CREATE UNIQUE INDEX authors_name_ci_unique_idx
    ON authors (lower(btrim(name)));

CREATE UNIQUE INDEX categories_name_ci_unique_idx
    ON categories (lower(btrim(name)));

-- One status shelf per user; custom shelves may repeat names only as allowed by
-- the existing (user_id, name) unique constraint.
CREATE UNIQUE INDEX shelves_one_per_status_type_idx
    ON shelves (user_id, shelf_type)
    WHERE shelf_type <> 'custom';

-- ---------------------------------------------------------------------------
-- 3. updated_at maintenance
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS user_profiles_set_updated_at ON user_profiles;
CREATE TRIGGER user_profiles_set_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

DROP TRIGGER IF EXISTS reading_progress_set_updated_at ON reading_progress;
CREATE TRIGGER reading_progress_set_updated_at
    BEFORE UPDATE ON reading_progress
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

DROP TRIGGER IF EXISTS ratings_set_updated_at ON ratings;
CREATE TRIGGER ratings_set_updated_at
    BEFORE UPDATE ON ratings
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

DROP TRIGGER IF EXISTS reviews_set_updated_at ON reviews;
CREATE TRIGGER reviews_set_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- 4. Default profile and shelves on registration
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_create_user_defaults(p_user_id uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO user_profiles (user_id)
    VALUES (p_user_id)
    ON CONFLICT (user_id) DO NOTHING;

    INSERT INTO shelves (user_id, name, shelf_type)
    SELECT p_user_id, v.name, v.shelf_type
    FROM (VALUES
        ('Want to read', 'want_to_read'),
        ('Currently reading', 'currently_reading'),
        ('Finished', 'finished'),
        ('Paused', 'paused')
    ) AS v(name, shelf_type)
    WHERE NOT EXISTS (
        SELECT 1
        FROM shelves s
        WHERE s.user_id = p_user_id
          AND s.shelf_type = v.shelf_type
    );
END;
$$;

CREATE OR REPLACE FUNCTION fn_users_after_insert()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM fn_create_user_defaults(NEW.user_id);
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS users_create_defaults ON users;
CREATE TRIGGER users_create_defaults
    AFTER INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION fn_users_after_insert();

-- ---------------------------------------------------------------------------
-- 5. Rating validation and summary maintenance
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_validate_rating()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.rating_value < 1 OR NEW.rating_value > 5 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'rating_value must be between 1 and 5';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS ratings_validate_value ON ratings;
CREATE TRIGGER ratings_validate_value
    BEFORE INSERT OR UPDATE ON ratings
    FOR EACH ROW EXECUTE FUNCTION fn_validate_rating();

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

CREATE OR REPLACE FUNCTION fn_ratings_summary_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM fn_refresh_book_rating_summary(OLD.book_id);
        RETURN OLD;
    END IF;

    PERFORM fn_refresh_book_rating_summary(NEW.book_id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION fn_reviews_summary_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM fn_refresh_book_rating_summary(OLD.book_id);
        RETURN OLD;
    END IF;

    PERFORM fn_refresh_book_rating_summary(NEW.book_id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION fn_books_after_insert()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO book_rating_summaries
        (book_id, average_rating, rating_count, review_count)
    VALUES (NEW.book_id, 0, 0, 0)
    ON CONFLICT (book_id) DO NOTHING;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS ratings_refresh_summary ON ratings;
CREATE TRIGGER ratings_refresh_summary
    AFTER INSERT OR UPDATE OR DELETE ON ratings
    FOR EACH ROW EXECUTE FUNCTION fn_ratings_summary_trigger();

DROP TRIGGER IF EXISTS reviews_refresh_summary ON reviews;
CREATE TRIGGER reviews_refresh_summary
    AFTER INSERT OR UPDATE OR DELETE ON reviews
    FOR EACH ROW EXECUTE FUNCTION fn_reviews_summary_trigger();

DROP TRIGGER IF EXISTS books_create_summary ON books;
CREATE TRIGGER books_create_summary
    AFTER INSERT ON books
    FOR EACH ROW EXECUTE FUNCTION fn_books_after_insert();

-- ---------------------------------------------------------------------------
-- 6. Reading-progress validation
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_validate_reading_progress()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    v_page_count integer;
BEGIN
    SELECT page_count INTO v_page_count
    FROM books
    WHERE book_id = NEW.book_id;

    IF v_page_count IS NOT NULL THEN
        IF NEW.current_page < 0 OR NEW.current_page > v_page_count THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = format(
                    'current_page must be between 0 and %s',
                    v_page_count
                );
        END IF;

        NEW.percent_complete := round(
            (NEW.current_page::numeric / v_page_count) * 100,
            2
        );

        -- Reaching the last page is completion. A caller that wants to move a
        -- finished book back to "want to read" must reset current_page to 0;
        -- fn_move_book_to_shelf does that automatically.
        IF NEW.current_page = v_page_count THEN
            NEW.status := 'completed';
        END IF;
    END IF;

    IF NEW.status NOT IN ('want_to_read', 'reading', 'completed', 'paused') THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'status must be want_to_read, reading, completed, or paused';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS reading_progress_validate ON reading_progress;
CREATE TRIGGER reading_progress_validate
    BEFORE INSERT OR UPDATE ON reading_progress
    FOR EACH ROW EXECUTE FUNCTION fn_validate_reading_progress();

-- ---------------------------------------------------------------------------
-- 7. Append-only reading history
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_block_history_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    -- Normal API paths never update or delete history. A deliberate account
    -- deletion must first run:
    --     SET LOCAL app.allow_history_delete = 'on';
    -- in the same transaction so the user/book cascade can proceed.
    IF TG_OP = 'DELETE'
       AND current_setting('app.allow_history_delete', true) = 'on' THEN
        RETURN OLD;
    END IF;

    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'reading_history is append-only';
END;
$$;

DROP TRIGGER IF EXISTS reading_history_immutable ON reading_history;
CREATE TRIGGER reading_history_immutable
    BEFORE UPDATE OR DELETE ON reading_history
    FOR EACH ROW EXECUTE FUNCTION fn_block_history_mutation();

-- ---------------------------------------------------------------------------
-- 8. Shelf movement and progress-update functions
-- ---------------------------------------------------------------------------
-- Both functions lock the user's status shelves in deterministic order before
-- locking reading_progress. That ordering keeps concurrent shelf/progress
-- operations from deadlocking.

CREATE OR REPLACE FUNCTION fn_move_book_to_shelf(
    p_user_id uuid,
    p_book_id uuid,
    p_shelf_id uuid
) RETURNS reading_progress
LANGUAGE plpgsql
AS $$
DECLARE
    v_shelf shelves%ROWTYPE;
    v_progress reading_progress%ROWTYPE;
    v_from_status text;
    v_status text;
BEGIN
    PERFORM 1
    FROM shelves
    WHERE user_id = p_user_id
      AND shelf_type <> 'custom'
    ORDER BY shelf_id
    FOR UPDATE;

    SELECT * INTO v_shelf
    FROM shelves
    WHERE shelf_id = p_shelf_id
      AND user_id = p_user_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = 'P0002',
            MESSAGE = 'shelf not found';
    END IF;

    PERFORM 1 FROM books WHERE book_id = p_book_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = 'P0002',
            MESSAGE = 'book not found';
    END IF;

    SELECT * INTO v_progress
    FROM reading_progress
    WHERE user_id = p_user_id
      AND book_id = p_book_id
    FOR UPDATE;

    IF NOT FOUND THEN
        INSERT INTO reading_progress (user_id, book_id, status)
        VALUES (p_user_id, p_book_id, 'want_to_read')
        RETURNING * INTO v_progress;
    END IF;

    v_from_status := v_progress.status;

    -- Custom shelves are additive and never change reading status. Status
    -- shelves are exclusive: remove the book from every other status shelf.
    IF v_shelf.shelf_type = 'custom' THEN
        v_status := v_progress.status;
    ELSE
        DELETE FROM user_shelf_items usi
        USING shelves s
        WHERE usi.shelf_id = s.shelf_id
          AND usi.book_id = p_book_id
          AND s.user_id = p_user_id
          AND s.shelf_type <> 'custom'
          AND s.shelf_id <> p_shelf_id;

        v_status := CASE v_shelf.shelf_type
            WHEN 'want_to_read' THEN 'want_to_read'
            WHEN 'currently_reading' THEN 'reading'
            WHEN 'finished' THEN 'completed'
            WHEN 'paused' THEN 'paused'
            ELSE v_progress.status
        END;
    END IF;

    INSERT INTO user_shelf_items (shelf_id, book_id, added_at, moved_at)
    VALUES (p_shelf_id, p_book_id, now(), now())
    ON CONFLICT (shelf_id, book_id) DO UPDATE
        SET moved_at = now();

    UPDATE reading_progress
    SET status = v_status,
        current_page = CASE
            WHEN v_shelf.shelf_type = 'want_to_read' THEN 0
            WHEN v_shelf.shelf_type = 'finished' THEN coalesce(
                (SELECT page_count FROM books WHERE book_id = p_book_id),
                current_page
            )
            ELSE current_page
        END,
        version = version + 1
    WHERE progress_id = v_progress.progress_id
    RETURNING * INTO v_progress;

    INSERT INTO reading_history
        (user_id, book_id, event_type, from_status, to_status)
    VALUES
        (p_user_id, p_book_id, 'shelf_moved', v_from_status, v_progress.status);

    RETURN v_progress;
END;
$$;

CREATE OR REPLACE FUNCTION fn_update_reading_progress(
    p_user_id uuid,
    p_book_id uuid,
    p_current_page integer,
    p_status text,
    p_expected_version integer
) RETURNS reading_progress
LANGUAGE plpgsql
AS $$
DECLARE
    v_progress reading_progress%ROWTYPE;
    v_old_status text;
    v_shelf_id uuid;
BEGIN
    PERFORM 1
    FROM shelves
    WHERE user_id = p_user_id
      AND shelf_type <> 'custom'
    ORDER BY shelf_id
    FOR UPDATE;

    PERFORM 1 FROM books WHERE book_id = p_book_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = 'P0002',
            MESSAGE = 'book not found';
    END IF;

    SELECT * INTO v_progress
    FROM reading_progress
    WHERE user_id = p_user_id
      AND book_id = p_book_id
    FOR UPDATE;

    IF NOT FOUND THEN
        INSERT INTO reading_progress
            (user_id, book_id, current_page, status)
        VALUES
            (
                p_user_id,
                p_book_id,
                CASE
                    WHEN p_status = 'want_to_read' THEN 0
                    WHEN p_status = 'completed' THEN coalesce(
                        (SELECT page_count FROM books WHERE book_id = p_book_id),
                        coalesce(p_current_page, 0)
                    )
                    ELSE coalesce(p_current_page, 0)
                END,
                coalesce(p_status, 'want_to_read')
            )
        RETURNING * INTO v_progress;
    ELSE
        IF p_expected_version IS NOT NULL
           AND p_expected_version <> v_progress.version THEN
            RAISE EXCEPTION USING
                ERRCODE = '40001',
                MESSAGE = 'stale reading progress version';
        END IF;

        v_old_status := v_progress.status;

        UPDATE reading_progress
        SET current_page = CASE
                WHEN p_status = 'want_to_read' THEN 0
                WHEN p_status = 'completed' THEN coalesce(
                    (SELECT page_count FROM books WHERE book_id = p_book_id),
                    coalesce(p_current_page, v_progress.current_page)
                )
                ELSE coalesce(p_current_page, v_progress.current_page)
            END,
            status = coalesce(p_status, v_progress.status),
            version = version + 1
        WHERE progress_id = v_progress.progress_id
        RETURNING * INTO v_progress;
    END IF;

    IF v_old_status IS DISTINCT FROM v_progress.status THEN
        INSERT INTO reading_history
            (user_id, book_id, event_type, from_status, to_status)
        VALUES
            (
                p_user_id,
                p_book_id,
                CASE
                    WHEN v_progress.status = 'completed' THEN 'book_completed'
                    ELSE 'status_changed'
                END,
                v_old_status,
                v_progress.status
            );

        SELECT shelf_id INTO v_shelf_id
        FROM shelves
        WHERE user_id = p_user_id
          AND shelf_type = CASE v_progress.status
                WHEN 'completed' THEN 'finished'
                WHEN 'reading' THEN 'currently_reading'
                WHEN 'paused' THEN 'paused'
                ELSE 'want_to_read'
            END;

        IF v_shelf_id IS NOT NULL THEN
            DELETE FROM user_shelf_items usi
            USING shelves s
            WHERE usi.shelf_id = s.shelf_id
              AND usi.book_id = p_book_id
              AND s.user_id = p_user_id
              AND s.shelf_type <> 'custom'
              AND s.shelf_id <> v_shelf_id;

            INSERT INTO user_shelf_items (shelf_id, book_id, added_at, moved_at)
            VALUES (v_shelf_id, p_book_id, now(), now())
            ON CONFLICT (shelf_id, book_id) DO UPDATE
                SET moved_at = now();
        END IF;
    ELSE
        INSERT INTO reading_history
            (user_id, book_id, event_type, from_status, to_status)
        VALUES
            (p_user_id, p_book_id, 'progress_updated', NULL, v_progress.status);
    END IF;

    RETURN v_progress;
END;
$$;

-- ---------------------------------------------------------------------------
-- 9. Backfills for existing rows
-- ---------------------------------------------------------------------------
SELECT fn_create_user_defaults(user_id) FROM users;

SELECT fn_refresh_book_rating_summary(book_id) FROM books;

COMMIT;
