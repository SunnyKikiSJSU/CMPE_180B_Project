BEGIN;

-- Restore the 000003 definitions verbatim. Triggers reference these functions
-- by name, so CREATE OR REPLACE swaps the behavior without touching them.

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

COMMIT;
