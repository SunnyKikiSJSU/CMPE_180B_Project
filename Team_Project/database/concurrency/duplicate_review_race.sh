#!/usr/bin/env bash
# BACKEND_PLAN.md §7.2 "Duplicate review race for one user/book".
#
# One user submits the same review from two sessions. Session A inserts and
# keeps its transaction open; session B's insert waits on the unique index
# entry and fails with 23505 reviews_user_book_unique once A commits. The API
# maps that to 409 CONFLICT (POST /api/books/{bookId}/reviews).
# Usage: database/concurrency/duplicate_review_race.sh   (see _lib.sh for PSQL)
source "$(dirname "$0")/_lib.sh"

u=$(make_user 'Reviewer'); b=$(make_book 'Concurrency: duplicate review race')
trap 'cleanup "$(ids "$b")" "$(ids "$u")"' EXIT

echo "=== Duplicate review: two sessions, same user and book, READ COMMITTED ==="
session A <<SQL &
BEGIN;
SELECT $ts || '  A: begin, insert review';
INSERT INTO reviews (user_id, book_id, review_text) VALUES ('$u', '$b', 'First tab');
SELECT $ts || '  A: inserted, sleeping 2s';
SELECT pg_sleep(2);
COMMIT;
SELECT $ts || '  A: committed';
SQL
sleep 0.5
session B <<SQL &
\set ON_ERROR_STOP 0
BEGIN;
SELECT $ts || '  B: begin, insert the same user/book';
INSERT INTO reviews (user_id, book_id, review_text) VALUES ('$u', '$b', 'Second tab');
ROLLBACK;
$(failed B)
SQL
sleep 0.7; observe
wait; show A B observer
echo "result: $(sql -c "SELECT 'reviews for the book: ' || count(*) || ' (' || string_agg(review_text, ', ') || ')' FROM reviews WHERE book_id = '$b'") | $(summary_line "$b")"
echo "expected: B fails with 23505 on reviews_user_book_unique; one review; summary reviews=1"
