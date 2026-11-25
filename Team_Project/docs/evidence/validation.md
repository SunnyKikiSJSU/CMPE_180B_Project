# Validation cases

34 database rules, checked by
[`database/tests/validation_cases.sql`](../../database/tests/validation_cases.sql)
(BACKEND_PLAN.md §9.2: cases 1-29 from the plan, 30-34 added). Each case
records what the database should do and what it did: the SQLSTATE and the
constraint or index for a rejected statement, or the resulting state for a
trigger or function.

## How to reproduce

```bash
# Any migrated database (000001-000008); seeds are not needed.
psql "$DATABASE_URL" -f database/tests/validation_cases.sql
# Local Docker database:
docker exec -i bm-pg psql -U book_app -d book_metaverse < database/tests/validation_cases.sql
# RDS, from Team_Project on the EC2 instance:
docker compose --env-file .env -f docker-compose-rdbs.yml -p book-metaverse-rds \
  --profile client run --rm -T postgres-client < database/tests/validation_cases.sql
```

- **No side effects.** The script runs in one transaction and ends with
  `ROLLBACK`, so it is safe on a shared database, including RDS. It creates
  its own users and books (`validation-*@example.com`, `Validation: ...`).
- **One case cannot break another.** Each rejected statement runs in its own
  subtransaction (a PL/pgSQL `EXCEPTION` block), so the transaction continues.
- **Fails loudly.** The script prints the table below and exits non-zero
  (psql exit code 3) if any case fails.

## What the codes mean

| SQLSTATE | Meaning | Raised by |
| --- | --- | --- |
| `23505` | Unique violation | `UNIQUE` constraints and unique indexes |
| `23514` | Check violation | `CHECK` constraints |
| `23503` | Foreign-key violation | `REFERENCES` constraints |
| `22023` | Invalid parameter value | `fn_validate_rating`, `fn_validate_reading_progress` (BEFORE triggers, so they run before the `CHECK` constraints) |
| `55000` | Object not in prerequisite state | `fn_block_history_mutation` (`reading_history` is append-only) |
| `BM001` | Stale progress version (project code) | `fn_update_reading_progress`; the API returns `409 STALE_VERSION` |

Cases 11 and 12 use a book without a page count. There the trigger has nothing to check against, so the `CHECK` constraints are what reject the row. Case 10 uses a book with 300 pages, so the trigger rejects it.

## Results

Recorded 2026-11-25 on PostgreSQL 18.7 (local Docker, migrations
`000001`-`000008` and seeds `001`-`004`): **34 passed, 0 failed**. The same
script on a migrated database without seeds, and on the deployed **RDS**
database (PostgreSQL 18.3, `cmpe180b-team-project-db`, through the EC2
instance), also gave 34 passed, 0 failed, exit code 0.

| # | Case | Expected | Actual | Result |
| --- | --- | --- | --- | --- |
| 1 | Duplicate email rejected | `rejected 23505 users_email_unique` | `rejected 23505 users_email_unique` | PASS |
| 2 | Email with uppercase or spaces rejected (must be normalized) | `rejected 23514 users_email_normalized` | `rejected 23514 users_email_normalized` | PASS |
| 3 | Blank display name rejected | `rejected 23514 users_display_name_not_blank` | `rejected 23514 users_display_name_not_blank` | PASS |
| 4 | Rating 0 rejected | `rejected 22023` | `rejected 22023` | PASS |
| 5 | Rating 6 rejected | `rejected 22023` | `rejected 22023` | PASS |
| 6 | Rating 5 accepted and summarized | `stored 5` | `stored 5` | PASS |
| 7 | Second rating by the same user for the same book rejected | `rejected 23505 ratings_user_book_unique` | `rejected 23505 ratings_user_book_unique` | PASS |
| 8 | Second review by the same user for the same book rejected | `rejected 23505 reviews_user_book_unique` | `rejected 23505 reviews_user_book_unique` | PASS |
| 9 | Blank review text rejected | `rejected 23514 reviews_text_not_blank` | `rejected 23514 reviews_text_not_blank` | PASS |
| 10 | Current page above the book's page count rejected | `rejected 22023` | `rejected 22023` | PASS |
| 11 | Negative current page rejected | `rejected 23514 reading_progress_current_page_nonnegative` | `rejected 23514 reading_progress_current_page_nonnegative` | PASS |
| 12 | Percentage above 100 rejected | `rejected 23514 reading_progress_percent_range` | `rejected 23514 reading_progress_percent_range` | PASS |
| 13 | Unknown reading status rejected | `rejected 22023` | `rejected 22023` | PASS |
| 14 | Unknown shelf type rejected | `rejected 23514 shelves_type_valid` | `rejected 23514 shelves_type_valid` | PASS |
| 15 | Second "finished" status shelf for one user rejected | `rejected 23505 shelves_one_per_status_type_idx` | `rejected 23505 shelves_one_per_status_type_idx` | PASS |
| 16 | Duplicate custom shelf name for one user rejected | `rejected 23505 shelves_user_name_unique` | `rejected 23505 shelves_user_name_unique` | PASS |
| 17 | Rating for a book that does not exist rejected | `rejected 23503 ratings_book_fk` | `rejected 23503 ratings_book_fk` | PASS |
| 18 | Deleting a review updates review_count | `after insert 1, after delete 0` | `after insert 1, after delete 0` | PASS |
| 19 | Rating insert, update, and delete keep average/count correct | `insert 4,2 -> 3.00/2; update 2->5 -> 4.50/2; delete 4 -> 5.00/1` | `insert 4,2 -> 3.00/2; update 2->5 -> 4.50/2; delete 4 -> 5.00/1` | PASS |
| 20 | Updating or deleting reading_history rejected | `update rejected 55000, delete rejected 55000` | `update rejected 55000, delete rejected 55000` | PASS |
| 21 | Moving to a status shelf leaves only that status shelf (custom kept) | `status reading; status shelves currently_reading; custom kept true` | `status reading; status shelves currently_reading; custom kept true` | PASS |
| 22 | Registration creates a profile and the four status shelves | `profile true; shelves currently_reading,finished,paused,want_to_read` | `profile true; shelves currently_reading,finished,paused,want_to_read` | PASS |
| 23 | A book with two authors and two categories returns both | `authors Validation Author One & Validation Author Two; categories Validation Classics & Validation Fantasy` | `authors Validation Author One & Validation Author Two; categories Validation Classics & Validation Fantasy` | PASS |
| 24 | Deleting a user removes their shelves, progress, ratings, reviews, history | `before: shelves 5, items 2, progress 1, ratings 2, reviews 1, history 3, profile 1; after: shelves 0, items 0, progress 0, ratings 0, reviews 0, history 0, profile 0` | `before: shelves 5, items 2, progress 1, ratings 2, reviews 1, history 3, profile 1; after: shelves 0, items 0, progress 0, ratings 0, reviews 0, history 0, profile 0` | PASS |
| 25 | A "google" book without google_books_id and imported_at rejected | `rejected 23514 books_google_source_complete` | `rejected 23514 books_google_source_complete` | PASS |
| 26 | Unknown books.source rejected | `rejected 23514 books_source_valid` | `rejected 23514 books_source_valid` | PASS |
| 27 | A date without precision, or precision without a date, rejected | `date only rejected 23514 books_published_date_precision_matches; precision only rejected 23514 books_published_date_precision_matches` | `date only rejected 23514 books_published_date_precision_matches; precision only rejected 23514 books_published_date_precision_matches` | PASS |
| 28 | Importing a work twice returns the same book; another work with that ISBN-13 rejected | `same book_id true; other work rejected 23505 books_isbn13_unique` | `same book_id true; other work rejected 23505 books_isbn13_unique` | PASS |
| 29 | UpdateBook does not change an imported (openlibrary) book | `rows updated 0; title Validation import` | `rows updated 0; title Validation import` | PASS |
| 30 | An "openlibrary" book without openlibrary_id rejected | `rejected 23514 books_openlibrary_source_complete` | `rejected 23514 books_openlibrary_source_complete` | PASS |
| 31 | Malformed ISBN-13 rejected | `rejected 23514 books_isbn13_format` | `rejected 23514 books_isbn13_format` | PASS |
| 32 | Zero page count rejected | `rejected 23514 books_page_count_positive` | `rejected 23514 books_page_count_positive` | PASS |
| 33 | Reaching the last page completes the book and moves it to Finished | `status completed; percent 100.00; shelf finished` | `status completed; percent 100.00; shelf finished` | PASS |
| 34 | Progress update with a stale version rejected | `rejected BM001` | `rejected BM001` | PASS |

```text
 passed | failed | total
--------+--------+-------
     34 |      0 |    34
```

## Coverage

| Area | Cases | Mechanism |
| --- | --- | --- |
| Users and registration | 1-3, 22 | `UNIQUE`, `CHECK`, `users_create_defaults` trigger |
| Ratings | 4-7, 17, 19 | `fn_validate_rating`, `UNIQUE (user_id, book_id)`, FK, summary trigger |
| Reviews | 8-9, 18 | `UNIQUE (user_id, book_id)`, `CHECK`, summary trigger |
| Reading progress | 10-13, 33-34 | `fn_validate_reading_progress`, `CHECK`, `fn_update_reading_progress` (completion, version check) |
| Shelves | 14-16, 21 | `CHECK`, partial unique index `shelves_one_per_status_type_idx`, `fn_move_book_to_shelf` |
| History | 20 | `reading_history_immutable` trigger |
| Catalog | 23, 25-32 | Join tables, source and date `CHECK`s, `ON CONFLICT` import upsert, `books_isbn13_unique`, `UpdateBook`'s source guard |
| Cascades | 24 | `ON DELETE CASCADE` from `users` (history rows are deleted only through the cascade) |
