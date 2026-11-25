# Concurrency evidence

Required by `BACKEND_PLAN.md` §7.2 and §12. Each scenario runs real concurrent
PostgreSQL sessions; the scripts are in [`database/concurrency/`](../../database/concurrency/).
The API uses the default isolation level, `READ COMMITTED`, and retries a
transaction once on `40001`/`40P01`
(`backend/internal/services/reading.go` and `reviews.go`).

| Scenario | Mechanism | Result | Script |
| --- | --- | --- | --- |
| Two users rate one book at the same time | Summary row lock in `fn_refresh_book_rating_summary` (migration `000007`) | Both ratings persist; count 2, average 3.00 | `rating_summary_race.sh` |
| Same race, second session at `REPEATABLE READ` | Snapshot isolation | `40001` on the summary row; a retry succeeds | `rating_summary_race.sh` |
| Duplicate review race for one user/book | `reviews_user_book_unique` | One review; the second insert waits, then fails with `23505` (API: one `201`, one `409`) | `duplicate_review_race.sh` |
| Stale progress version (two tabs, one finishes the book) | `FOR UPDATE` on the status shelves and progress row + `version` check in `fn_update_reading_progress` | The second save waits, then fails with `BM001`; its retry with the new version succeeds; no update lost (API: one `200`, one `409`) | `stale_progress_race.sh`; API test `TestConcurrentProgressUpdates` |

## How to reproduce

On a migrated database (`000001`-`000007`; seeds are not needed):

```bash
DATABASE_URL=postgres://... database/concurrency/rating_summary_race.sh
DATABASE_URL=postgres://... database/concurrency/duplicate_review_race.sh
DATABASE_URL=postgres://... database/concurrency/stale_progress_race.sh
# or, with the local Docker database:
PSQL="docker exec -i bm-pg psql -U book_app -d book_metaverse" database/concurrency/rating_summary_race.sh
```

Each script creates throwaway users and books and deletes them at the end.
Session A starts a transaction, writes, sleeps 2 seconds, and commits; session
B starts 0.5 seconds later. An observer session lists who waits on whom
(`pg_stat_activity`, `pg_blocking_pids`). Times come from the server clock.

## Finding: lost update in the rating summary (fixed in `000007`)

The plan (§5.1, `000003` part E) assumed that the summary upsert's row lock serializes
concurrent ratings under `READ COMMITTED`. It does not. Migration `000003`
computed the aggregate and wrote it in one statement:

```sql
INSERT INTO book_rating_summaries (...)
SELECT p_book_id, avg(rating_value), count(*), ...
FROM ratings WHERE book_id = p_book_id
ON CONFLICT (book_id) DO UPDATE SET average_rating = EXCLUDED.average_rating, ...
```

Session B reads `ratings` with a snapshot taken when that statement starts,
before A commits, so B's aggregate does not include A's rating. B then waits
for A's lock on the summary row, and once A commits it overwrites A's correct
summary with its stale `EXCLUDED` values. The ratings table keeps both
ratings; the summary shows one.

Before the fix (migration version 6), recorded 2026-11-20:

```text
migration version: 6

=== Scenario 1: two users rate one book, both READ COMMITTED ===
[A] 08:11:31.258  A: begin, rating 5
[A] 08:11:31.262  A: inserted; summary row now locked by A, sleeping 2s
[A] 08:11:33.267  A: committed
[B] 08:11:31.779  B: begin, rating 1 (READ COMMITTED)
[B] 08:11:33.267  B: insert returned (it waited for A's lock)
[B] 08:11:33.269  B: committed
[observer] 08:11:32.504  waiting: pid 2154 Lock/transactionid blocked by {2147} -> INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('3fb5186b
result: ratings table: count=2 avg=3.00 | summary row: count=1 avg=1.00 reviews=0
expected: count=2 avg=3.00 in both
```

The summary says 1 rating with average 1.00 (only B's), while the table has 2
ratings with average 3.00. Through the API the effect is larger: 12 users
rating one book at once left `rating_count` at 1, 4, or 6 in three runs
(`TestConcurrentRatingsKeepSummaryConsistent` with `000007` rolled back).

**Fix.** `000007_rating_summary_locking` replaces the function. It first makes
sure the summary row exists, then locks it with `SELECT ... FOR UPDATE`, and
only then computes the aggregate in a separate statement. In PL/pgSQL each
statement gets a new snapshot under `READ COMMITTED`, so the aggregate is read
after the lock is granted and includes every committed rating. Writers on the
same book run one after another; different books never block each other. The
up migration also recomputes every existing summary once, so rows the race
already broke are repaired.

## Recorded output with `000007` (PostgreSQL 18, 2026-11-20)

```text
migration version: 7

=== Scenario 1: two users rate one book, both READ COMMITTED ===
[A] 08:11:36.590  A: begin, rating 5
[A] 08:11:36.592  A: inserted; summary row now locked by A, sleeping 2s
[A] 08:11:38.598  A: committed
[B] 08:11:37.112  B: begin, rating 1 (READ COMMITTED)
[B] 08:11:38.601  B: insert returned (it waited for A's lock)
[B] 08:11:38.602  B: committed
[observer] 08:11:37.820  waiting: pid 2252 Lock/transactionid blocked by {2245} -> INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('4b0f60a6
result: ratings table: count=2 avg=3.00 | summary row: count=2 avg=3.00 reviews=0
expected: count=2 avg=3.00 in both

=== Scenario 2: same race, B at REPEATABLE READ ===
[A] 08:11:38.721  A: begin, rating 5
[A] 08:11:38.724  A: inserted, sleeping 2s
[A] 08:11:40.734  A: committed
[B] 08:11:39.239  B: begin REPEATABLE READ, snapshot taken now (before A commits)
[B] 08:11:40.735  B: failed with 40001: could not serialize access due to concurrent update
[B] 08:11:40.735  B: rolled back; retrying in a new transaction
[B] 08:11:40.739  B: retry committed
[B stderr] ERROR:  could not serialize access due to concurrent update
[B stderr] CONTEXT:  SQL statement "INSERT INTO book_rating_summaries
[B stderr]         (book_id, average_rating, rating_count, review_count)
[B stderr]     VALUES (p_book_id, 0, 0, 0)
[B stderr]     ON CONFLICT (book_id) DO NOTHING"
[B stderr] PL/pgSQL function fn_refresh_book_rating_summary(uuid) line 11 at SQL statement
[B stderr] SQL statement "SELECT fn_refresh_book_rating_summary(NEW.book_id)"
[B stderr] PL/pgSQL function fn_ratings_summary_trigger() line 8 at PERFORM
[observer] 08:11:39.945  waiting: pid 2280 Lock/transactionid blocked by {2273} -> INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('4b0f60a6
result: ratings table: count=2 avg=3.00 | summary row: count=2 avg=3.00 reviews=0
expected: B's first attempt fails with 40001; after the retry count=2 avg=3.00
```

- **Scenario 1:** B's insert waits for A (`Lock/transactionid`, blocked by A's
  pid) and returns when A commits. The summary matches the table: 2 ratings,
  average 3.00.
- **Scenario 2:** at `REPEATABLE READ`, B's snapshot cannot see A's commit, so
  PostgreSQL refuses B's write to the summary row A changed (`40001`, could not
  serialize access due to concurrent update) instead of producing a stale
  summary. A retry in a new transaction succeeds. Under `READ COMMITTED` the
  lock-then-read order gives the same correct result without the error; the
  API's retry helper still covers `40001` and `40P01` if they occur.

### Duplicate review

```text
=== Duplicate review: two sessions, same user and book, READ COMMITTED ===
[A] 08:11:41.040  A: begin, insert review
[A] 08:11:41.043  A: inserted, sleeping 2s
[A] 08:11:43.052  A: committed
[B] 08:11:41.573  B: begin, insert the same user/book
[B] 08:11:43.053  B: failed with 23505: duplicate key value violates unique constraint "reviews_user_book_unique"
[B stderr] ERROR:  duplicate key value violates unique constraint "reviews_user_book_unique"
[B stderr] DETAIL:  Key (user_id, book_id)=(e58351f9-def0-41e5-9567-1506cb9c7ec2, e7808921-4a64-4029-9bdd-21d29e695d91) already exists.
[observer] 08:11:42.281  waiting: pid 2328 Lock/transactionid blocked by {2321} -> INSERT INTO reviews (user_id, book_id, review_text) VALUES ('e58351f9-
result: reviews for the book: 1 (First tab) | ratings table: count=0 avg=0 | summary row: count=0 avg=0.00 reviews=1
expected: B fails with 23505 on reviews_user_book_unique; one review; summary reviews=1
```

B's insert waits on A's uncommitted entry in the unique index, then fails with
`23505` on `reviews_user_book_unique` as soon as A commits. The API maps this to
`409 CONFLICT`; `TestDuplicateReviewRace` sends two identical requests at once
and gets one `201` and one `409`.

### Stale progress version

Recorded 2026-11-25 (PostgreSQL 18.7, migrations `000001`-`000008`).

```text
start: progress: status=reading page=40 version=1 | shelf=currently_reading | history: status_changed:reading

=== Stale progress: two tabs save version 1 at the same time, READ COMMITTED ===
[A] 20:24:50.040  A: begin, finish the book (page 300, expected version 1)
[A] 20:24:50.045  A: saved -> status=completed page=300 version=2
[A] 20:24:50.045  A: holding the locks for 2s
[A] 20:24:52.057  A: committed
[B] 20:24:50.565  B: begin, save page 120 (expected version 1)
[B] 20:24:52.061  B: failed with BM001: stale reading progress version
[B] 20:24:52.061  B: reload shows version 2, status completed; retry with it
[B] 20:24:52.068  B: retry saved -> status=reading page=120 version=3
[B stderr] ERROR:  stale reading progress version
[B stderr] CONTEXT:  PL/pgSQL function fn_update_reading_progress(uuid,uuid,integer,text,integer) line 44 at RAISE
[observer] 20:24:51.299  waiting: pid 493 Lock/transactionid blocked by {486} -> SELECT to_char(clock_timestamp(), 'HH24:MI:SS.MS') || ' B: saved -> st
result: progress: status=reading page=120 version=3 | shelf=currently_reading | history: status_changed:reading, book_completed:completed, status_changed:reading
expected: B waits for A's lock, fails with BM001 (stale version), reloads version 2, and its retry
          saves page 120 as version 3 (status back to reading, shelf currently_reading); no update is lost
```

Both tabs loaded version 1. A finishes the book. B's save then waits, because
`fn_update_reading_progress` first locks the reader's status shelves and then
the progress row (`FOR UPDATE`, always in the same order, so two saves cannot
deadlock). The observer shows B waiting on A's transaction. When A commits,
B's `FOR UPDATE` returns the *committed* row (under `READ COMMITTED` a locking
read sees the newest version), so B sees version 2, not the 1 it expected, and
raises `BM001` instead of overwriting the completion with page 120. The API
returns `409 STALE_VERSION`; the page reloads and the reader decides. Here
the retry saves page 120 as version 3, so history keeps all three changes:
started, completed, reading again.

Without the version check the second save would still be serialized by the
row lock, but it would silently replace the completion: a lost update from
the reader's point of view.

## API-level tests

`backend/internal/integration/reviews_integration_test.go`:

- `TestConcurrentRatingsKeepSummaryConsistent`: 12 users rate one book in
  parallel; the stored summary must equal the real count and average. It fails
  without `000007` and passes with it.
- `TestDuplicateReviewRace`: one `201` and one `409`; the review count stays 1.
