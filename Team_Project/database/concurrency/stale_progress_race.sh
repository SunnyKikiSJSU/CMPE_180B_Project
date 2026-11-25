#!/usr/bin/env bash
# BACKEND_PLAN.md §7.2 "Progress update vs. completion".
#
# One reader has the book open in two tabs, both showing progress version 1.
# Tab A finishes the book (page 300 of 300) and keeps its transaction open;
# tab B saves page 120 with the same version. fn_update_reading_progress
# locks the reader's status shelves and the progress row (FOR UPDATE), so B
# waits. When A commits, B re-reads the row, sees version 2, and fails with
# BM001 (the API's 409 STALE_VERSION) instead of silently undoing the
# completion. B then reloads and retries with the current version.
# Usage: database/concurrency/stale_progress_race.sh   (see _lib.sh for PSQL)
source "$(dirname "$0")/_lib.sh"

u=$(make_user 'Two tabs')
b=$(sql -c "INSERT INTO books (title, source, page_count) VALUES ('Concurrency: stale progress race', 'user', 300) RETURNING book_id")
trap 'cleanup "$(ids "$b")" "$(ids "$u")"' EXIT

state() {
  sql -c "SELECT 'progress: status=' || rp.status || ' page=' || rp.current_page || ' version=' || rp.version
                 || ' | shelf=' || (SELECT string_agg(s.shelf_type, ',') FROM user_shelf_items i JOIN shelves s USING (shelf_id)
                                    WHERE s.user_id = rp.user_id AND i.book_id = rp.book_id)
                 || ' | history: ' || (SELECT string_agg(event_type || coalesce(':' || to_status, ''), ', ' ORDER BY created_at, event_type)
                                       FROM reading_history h WHERE h.user_id = rp.user_id AND h.book_id = rp.book_id)
          FROM reading_progress rp WHERE rp.user_id = '$u' AND rp.book_id = '$b'"
}

sql -c "SELECT 1 FROM fn_update_reading_progress('$u', '$b', 40, 'reading', NULL)" >/dev/null
echo "start: $(state)"
echo ""
echo "=== Stale progress: two tabs save version 1 at the same time, READ COMMITTED ==="
session A <<SQL &
BEGIN;
SELECT $ts || '  A: begin, finish the book (page 300, expected version 1)';
SELECT $ts || '  A: saved -> status=' || status || ' page=' || current_page || ' version=' || version
FROM fn_update_reading_progress('$u', '$b', 300, NULL, 1);
SELECT $ts || '  A: holding the locks for 2s';
SELECT pg_sleep(2);
COMMIT;
SELECT $ts || '  A: committed';
SQL
sleep 0.5
session B <<SQL &
\set ON_ERROR_STOP 0
BEGIN;
SELECT $ts || '  B: begin, save page 120 (expected version 1)';
SELECT $ts || '  B: saved -> status=' || status || ' page=' || current_page || ' version=' || version
FROM fn_update_reading_progress('$u', '$b', 120, NULL, 1);
ROLLBACK;
$(failed B)
SELECT $ts || '  B: reload shows version ' || version || ', status ' || status || '; retry with it'
FROM reading_progress WHERE user_id = '$u' AND book_id = '$b';
\set ON_ERROR_STOP 1
SELECT version AS current_version FROM reading_progress WHERE user_id = '$u' AND book_id = '$b' \gset
SELECT $ts || '  B: retry saved -> status=' || status || ' page=' || current_page || ' version=' || version
FROM fn_update_reading_progress('$u', '$b', 120, NULL, :current_version);
SQL
sleep 0.7; observe
wait; show A B observer
echo "result: $(state)"
echo "expected: B waits for A's lock, fails with BM001 (stale version), reloads version 2, and its retry"
echo "          saves page 120 as version 3 (status back to reading, shelf currently_reading); no update is lost"
