BEGIN;

-- Fixes found while verifying 000003:
--   A. A completed book could never leave "completed": the validation trigger
--      forced completion whenever current_page = page_count, and moving a book
--      off the Finished shelf kept current_page at the last page.
--   B. A stale optimistic version raised SQLSTATE 40001, the same code as a
--      real serialization failure that the API retries. It now raises BM001.
--   C. The append-only history trigger blocked the ON DELETE CASCADE from
--      users and books, so deleting an account or a book failed.

-- ---------------------------------------------------------------------------
-- A. Reading-progress validation with explicit status/page transitions
-- ---------------------------------------------------------------------------
-- This trigger is the single owner of the page/status relationship, so every
-- write path (functions, sqlc queries, ad-hoc SQL) gets the same behavior:
--
--   * status becomes completed        -> current_page jumps to the last page
--   * status leaves completed and the
--     page was left at the last page  -> current_page restarts at 0
--   * page lowered on a completed book -> status becomes reading
--   * page reaches the last page      -> status becomes completed
--
-- percent_complete is always recomputed from current_page / page_count.
CREATE OR REPLACE FUNCTION fn_validate_reading_progress()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    v_page_count integer;
BEGIN
    IF NEW.status NOT IN ('want_to_read', 'reading', 'completed', 'paused') THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'status must be want_to_read, reading, completed, or paused';
    END IF;

    SELECT page_count INTO v_page_count
    FROM books
    WHERE book_id = NEW.book_id;

    -- Without a page count the client-supplied percentage is accepted as-is.
    IF v_page_count IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.current_page < 0 OR NEW.current_page > v_page_count THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = format(
                'current_page must be between 0 and %s',
                v_page_count
            );
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.status = 'completed' THEN
            NEW.current_page := v_page_count;
        ELSIF NEW.current_page = v_page_count THEN
            NEW.status := 'completed';
        END IF;
    ELSE
        IF NEW.status = 'completed' AND OLD.status <> 'completed' THEN
            NEW.current_page := v_page_count;
        ELSIF OLD.status = 'completed'
              AND NEW.status <> 'completed'
              AND NEW.current_page = OLD.current_page
              AND NEW.current_page = v_page_count THEN
            -- Re-reading, pausing, or re-queueing a finished book.
            NEW.current_page := 0;
        ELSIF OLD.status = 'completed'
              AND NEW.status = 'completed'
              AND NEW.current_page < v_page_count THEN
            NEW.status := 'reading';
        ELSIF NEW.current_page = v_page_count
              AND NEW.current_page <> OLD.current_page THEN
            NEW.status := 'completed';
        END IF;
    END IF;

    NEW.percent_complete := round(
        (NEW.current_page::numeric / v_page_count) * 100,
        2
    );

    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- C. Append-only history that still allows owner cascades
-- ---------------------------------------------------------------------------
-- Foreign-key cascades run as internal RI triggers, so a cascaded delete
-- reaches this trigger with pg_trigger_depth() > 1. A direct
-- DELETE FROM reading_history runs at depth 1 and is still rejected, as is
-- every UPDATE.
CREATE OR REPLACE FUNCTION fn_block_history_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE'
       AND (
            pg_trigger_depth() > 1
            OR current_setting('app.allow_history_delete', true) = 'on'
       ) THEN
        RETURN OLD;
    END IF;

    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'reading_history is append-only';
END;
$$;

-- ---------------------------------------------------------------------------
-- B. Stale-version error gets its own SQLSTATE
-- ---------------------------------------------------------------------------
-- BM001 is a project-specific code (class "BM" is unused by PostgreSQL). The
-- API maps it to 409 STALE_VERSION and never retries it. 40001 is left for
-- genuine serialization failures, which the API may retry once.
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
                    ELSE coalesce(p_current_page, 0)
                END,
                coalesce(p_status, 'want_to_read')
            )
        RETURNING * INTO v_progress;
    ELSE
        IF p_expected_version IS NOT NULL
           AND p_expected_version <> v_progress.version THEN
            RAISE EXCEPTION USING
                ERRCODE = 'BM001',
                MESSAGE = 'stale reading progress version';
        END IF;

        v_old_status := v_progress.status;

        -- The validation trigger owns the page/status relationship
        -- (completion, restart, reopen); this only applies the request.
        UPDATE reading_progress
        SET current_page = CASE
                WHEN p_status = 'want_to_read' THEN 0
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

COMMIT;
