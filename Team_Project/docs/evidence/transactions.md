# Transaction evidence

Required by `BACKEND_PLAN.md` §7.1 and §12. Each scenario runs in one
transaction (`READ COMMITTED`) and is shown once committed and once rolled back
after a failing step. The API runs the same functions in one `pgx.Tx` per
request (`backend/internal/services/reading.go`).

| Scenario | Functions | Status |
| --- | --- | --- |
| Shelf move | `fn_move_book_to_shelf` | Recorded below (Phase 5) |
| Progress completion | `fn_update_reading_progress` | Recorded below (Phase 5) |
| Review + rating | rating upsert, review upsert, summary trigger | Phase 6 |

## How to reproduce

On a database with migrations `000001`-`000006` and seeds `001`-`004`:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=0 -f database/tests/transactions_phase5.sql
```

The script creates a throwaway user (its four status shelves come from the
`users_create_defaults` trigger), uses the first seeded book with a page count,
and deletes the user at the end, so seed data is unchanged. `ERROR` lines are
the intended failing steps.

## What it shows

- **1a, shelf move committed:** two moves leave the book on exactly one status
  shelf (`currently_reading`), status `reading`, and two history rows.
- **1b, shelf move rolled back:** inside the transaction the book is
  `completed` on `finished` with a third history row; a failing step (unknown
  shelf, `P0002`) and `ROLLBACK` restore the 1a state exactly, including the
  version and history count.
- **2a, completion committed:** page 10, then the last page; the trigger sets
  `completed` and 100 %, the function moves the book to `finished` and writes
  `book_completed`.
- **2b, progress update rolled back:** a restart at page 5 followed by a page
  beyond the book (`22023`) rolls back to the 2a state.
- **2c, stale version:** an old `version` raises `BM001` (`409 STALE_VERSION` in
  the API) and changes nothing.

The API-level counterparts are integration tests in
`backend/internal/integration/reading_integration_test.go`
(`TestShelfMovesKeepProgressAndHistoryInStep`, `TestProgressUpdates`, and
`TestConcurrentProgressUpdates`, where two requests with the same version give
one `200` and one `409`).

## Recorded output (PostgreSQL 18, 2026-11-19)

```text
=== State query used after each step ===

=== Scenario 1a: shelf move, COMMIT ===
Add to Want to read, then move to Currently reading in one transaction.
    status    
--------------
 want_to_read

 status  
---------
 reading

 status  | current_page | percent_complete | version |      shelves      | history_rows 
---------+--------------+------------------+---------+-------------------+--------------
 reading |            0 |             0.00 |       3 | currently_reading |            2


=== Scenario 1b: shelf move, ROLLBACK ===
Move to Finished, then a step fails (unknown shelf): the whole transaction rolls back.
  status   
-----------
 completed

Inside the transaction, before the failure:
  status   | current_page | percent_complete | version | shelves  | history_rows 
-----------+--------------+------------------+---------+----------+--------------
 completed |          563 |           100.00 |       4 | finished |            3

ERROR:  shelf not found
CONTEXT:  PL/pgSQL function fn_move_book_to_shelf(uuid,uuid,uuid) line 22 at RAISE
After ROLLBACK (same as after 1a: still reading, Finished undone, no new history):
 status  | current_page | percent_complete | version |      shelves      | history_rows 
---------+--------------+------------------+---------+-------------------+--------------
 reading |            0 |             0.00 |       3 | currently_reading |            2


=== Scenario 2a: progress completion, COMMIT ===
Read to page 10, then to the last page: completed, moved to Finished, book_completed event.
 status  | current_page 
---------+--------------
 reading |           10

  status   | current_page 
-----------+--------------
 completed |          563

  status   | current_page | percent_complete | version | shelves  | history_rows 
-----------+--------------+------------------+---------+----------+--------------
 completed |          563 |           100.00 |       5 | finished |            4

   event_type   | from_status | to_status 
----------------+-------------+-----------
 book_completed | reading     | completed


=== Scenario 2b: progress update, ROLLBACK ===
Restart at page 5 (status reading), then a page beyond the book fails: everything rolls back.
 status  | current_page 
---------+--------------
 reading |            5

ERROR:  current_page must be between 0 and 563
CONTEXT:  PL/pgSQL function fn_validate_reading_progress() line 21 at RAISE
SQL statement "UPDATE reading_progress
        SET current_page = CASE
                WHEN p_status = 'want_to_read' THEN 0
                ELSE coalesce(p_current_page, v_progress.current_page)
            END,
            status = coalesce(p_status, v_progress.status),
            version = version + 1
        WHERE progress_id = v_progress.progress_id
        RETURNING *"
PL/pgSQL function fn_update_reading_progress(uuid,uuid,integer,text,integer) line 53 at SQL statement
After ROLLBACK (same as after 2a: completed, version unchanged):
  status   | current_page | percent_complete | version | shelves  | history_rows 
-----------+--------------+------------------+---------+----------+--------------
 completed |          563 |           100.00 |       5 | finished |            4


=== Scenario 2c: stale version is rejected (BM001) ===
ERROR:  stale reading progress version
CONTEXT:  PL/pgSQL function fn_update_reading_progress(uuid,uuid,integer,text,integer) line 44 at RAISE
  status   | current_page | percent_complete | version | shelves  | history_rows 
-----------+--------------+------------------+---------+----------+--------------
 completed |          563 |           100.00 |       5 | finished |            4

```
