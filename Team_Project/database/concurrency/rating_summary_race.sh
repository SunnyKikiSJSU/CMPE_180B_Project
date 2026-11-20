#!/usr/bin/env bash
# BACKEND_PLAN.md §7.2 "Two users rate the same book simultaneously".
#
# Session A rates a book and keeps its transaction open for 2 seconds.
# Session B rates the same book 0.5 seconds later. Expected: both ratings
# persist, and the summary shows count 2 with the correct average.
#
#   Scenario 1  READ COMMITTED (the API's level). B waits for A's lock on the
#               summary row, then recomputes and sees both ratings.
#   Scenario 2  B runs at REPEATABLE READ. Its snapshot cannot include A's
#               commit, so PostgreSQL stops it with 40001 instead of letting it
#               write a stale summary. A retry in a new transaction succeeds.
#
# Before migration 000007 scenario 1 lost A's rating in the summary (count 1).
# Usage: database/concurrency/rating_summary_race.sh   (see _lib.sh for PSQL)
source "$(dirname "$0")/_lib.sh"

ua=$(make_user 'Session A'); ub=$(make_user 'Session B')
b1=$(make_book 'Concurrency: rating race (READ COMMITTED)')
b2=$(make_book 'Concurrency: rating race (REPEATABLE READ)')
trap 'cleanup "$(ids "$b1" "$b2")" "$(ids "$ua" "$ub")"' EXIT

echo "migration version: $(sql -c 'SELECT version FROM schema_migrations')"
echo
echo "=== Scenario 1: two users rate one book, both READ COMMITTED ==="
session A <<SQL &
BEGIN;
SELECT $ts || '  A: begin, rating 5';
INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('$ua', '$b1', 5);
SELECT $ts || '  A: inserted; summary row now locked by A, sleeping 2s';
SELECT pg_sleep(2);
COMMIT;
SELECT $ts || '  A: committed';
SQL
sleep 0.5
session B <<SQL &
BEGIN;
SELECT $ts || '  B: begin, rating 1 (READ COMMITTED)';
INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('$ub', '$b1', 1);
SELECT $ts || '  B: insert returned (it waited for A''s lock)';
COMMIT;
SELECT $ts || '  B: committed';
SQL
sleep 0.7; observe
wait; show A B observer
echo "result: $(summary_line "$b1")"
echo "expected: count=2 avg=3.00 in both"

echo
echo "=== Scenario 2: same race, B at REPEATABLE READ ==="
session A <<SQL &
BEGIN;
SELECT $ts || '  A: begin, rating 5';
INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('$ua', '$b2', 5);
SELECT $ts || '  A: inserted, sleeping 2s';
SELECT pg_sleep(2);
COMMIT;
SELECT $ts || '  A: committed';
SQL
sleep 0.5
session B <<SQL &
\set ON_ERROR_STOP 0
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT $ts || '  B: begin REPEATABLE READ, snapshot taken now (before A commits)';
INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('$ub', '$b2', 1);
ROLLBACK;
$(failed B)
\set ON_ERROR_STOP 1
SELECT $ts || '  B: rolled back; retrying in a new transaction';
BEGIN ISOLATION LEVEL REPEATABLE READ;
INSERT INTO ratings (user_id, book_id, rating_value) VALUES ('$ub', '$b2', 1);
COMMIT;
SELECT $ts || '  B: retry committed';
SQL
sleep 0.7; observe
wait; show A B observer
echo "result: $(summary_line "$b2")"
echo "expected: B's first attempt fails with 40001; after the retry count=2 avg=3.00"
