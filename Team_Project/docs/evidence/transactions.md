# Transaction evidence

Required by `BACKEND_PLAN.md` §7.1 and §12. Each scenario runs in one
transaction (`READ COMMITTED`) and is shown once committed and once rolled back
after a failing step. The API runs the same functions in one `pgx.Tx` per
request (`backend/internal/services/reading.go` and `reviews.go`).

| Scenario | Functions | Status |
| --- | --- | --- |
| Shelf move | `fn_move_book_to_shelf` | Recorded below (Phase 5) |
| Progress completion | `fn_update_reading_progress` | Recorded below (Phase 5) |
| Review + rating | rating upsert, review upsert, summary trigger | Recorded below (Phase 6) |

## How to reproduce

On a database with migrations `000001`-`000007` and seeds `001`-`004`:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=0 -f database/tests/transactions_phase5.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=0 -f database/tests/transactions_phase6.sql
```

The Phase 6 script creates its own user and book and deletes both at the end.

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

- **3a, review + rating committed:** rating 4 and a review in one
  transaction; the triggers set the summary to 4.00, 1 rating, 1 review.
- **3b, review + rating rolled back:** the rating changes to 1 and the trigger
  rewrites the summary to 1.00 inside the transaction; the review step then
  fails (`23514`, `reviews_text_not_blank`) and `ROLLBACK` restores the rating
  and the summary written by the trigger, not only the rows the transaction
  wrote directly.

Concurrent writes to the same summary are covered in
[concurrency.md](concurrency.md).

The API-level counterparts are integration tests in
`backend/internal/integration/reading_integration_test.go`
(`TestShelfMovesKeepProgressAndHistoryInStep`, `TestProgressUpdates`, and
`TestConcurrentProgressUpdates`, where two requests with the same version give
one `200` and one `409`) and
`backend/internal/integration/reviews_integration_test.go`
(`TestSaveReviewAndRatingInOneTransaction` forces the review step of
`PUT /api/books/{bookId}/review` to fail after the rating step and checks that
the rating, review, and summary are unchanged).

## Recorded output, Phase 5 (PostgreSQL 18, 2026-11-19)

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

## Recorded output, Phase 6 (PostgreSQL 18, 2026-11-20)

```text
=== State query used after each step ===
New book (summary row created by the books_create_summary trigger):
 my_rating | my_review | average_rating | rating_count | review_count 
-----------+-----------+----------------+--------------+--------------
           |           |           0.00 |            0 |            0


=== Scenario 3a: review + rating, COMMIT ===
Rate 4 and write a review in one transaction; the triggers refresh the summary.
 my_rating |          my_review          | average_rating | rating_count | review_count 
-----------+-----------------------------+----------------+--------------+--------------
         4 | A careful, well-paced book. |           4.00 |            1 |            1


=== Scenario 3b: review + rating, ROLLBACK ===
Change the rating to 1, then the review step fails (blank text, CHECK reviews_text_not_blank).
Inside the transaction, after the rating step (the trigger already rewrote the summary):
 my_rating |          my_review          | average_rating | rating_count | review_count 
-----------+-----------------------------+----------------+--------------+--------------
         1 | A careful, well-paced book. |           1.00 |            1 |            1

ERROR:  new row for relation "reviews" violates check constraint "reviews_text_not_blank"
DETAIL:  Failing row contains (456eacba-7fe6-4d5e-a470-5964b91a3e85, c08d4427-04fb-430e-a6df-ff8f957ad30b, abe6fcb2-cf1a-4aaa-bf6a-b0e5e5adb00a, null,    , 2026-11-20 08:11:10.496279+00, 2026-11-20 08:11:10.496279+00).
After ROLLBACK (same as after 3a: rating 4, original review, summary 4.00 / 1 / 1):
 my_rating |          my_review          | average_rating | rating_count | review_count 
-----------+-----------------------------+----------------+--------------+--------------
         4 | A careful, well-paced book. |           4.00 |            1 |            1
```
