# SQL Query Catalog

These files are compiled by `sqlc` into `backend/internal/db`. Each query has a
`-- name:` annotation; do not rename a query without regenerating the Go code
and updating its repository call.

## Files

| File | Responsibility |
| --- | --- |
| `auth.sql` | User lookup by email/ID, creation, password update |
| `users.sql` | Profile read/update, account stats, account deletion |
| `books.sql` | Library search, book detail, catalog writes, Google import and dedupe lookups, author/category upserts |
| `authors.sql` | Author list/detail and author-scoped books |
| `categories.sql` | Category list/detail and category-scoped books |
| `shelves.sql` | Shelf CRUD, shelf-book listing, membership add/remove |
| `progress.sql` | Progress/history reads |
| `reviews.sql` | Review reads and owner-scoped writes |
| `ratings.sql` | Rating summary, distribution, and owner-scoped writes |
| `reports.sql` | Report queries q03-q10 |

## Required query catalog

| ID | Query | Endpoint |
| --- | --- | --- |
| `q01` | `SearchBooks` | `GET /api/books` |
| `q02` | `GetBookByID` + `ListBookReviews` | `GET /api/books/{bookId}` |
| `q03` | `ListShelfBooks` | `GET /api/shelves/{shelfId}/books` |
| `q04` | `RatingSummaryByCategory`, `RatingSummaryByAuthor` | report data |
| `q05` | `PopularBooks` | `GET /api/reports/popular` |
| `q06` | `RecommendedBooks` | `GET /api/reports/recommendations` |
| `q07` | `UserReadingStats`, `ReadingReportSummary` | `GET /api/reports/user-stats`, `GET /api/reports/reading-history` |
| `q08` | `AuthorRankings`, `CategoryRankings` | `GET /api/reports/authors`, `GET /api/reports/categories` |
| `q09` | `ReviewActivity` | `GET /api/reports/review-activity` |
| `q10` | `GetRatingDistribution` | `GET /api/books/{bookId}/ratings` |

## Google Books queries

Google search itself runs in the Go `googlebooks` client, not in SQL. These
queries connect its results to the library:

| Query | Endpoint | Role |
| --- | --- | --- |
| `ListBooksByExternalIDs` | `GET /api/external/books` | One lookup per result page; fills `bookId` for volumes (or ISBN-13s) already stored |
| `GetBookByExternalID` | `POST /api/books/import` | Dedupe by volume ID, then ISBN-13, before calling Google |
| `ImportGoogleBook` | `POST /api/books/import` | Inserts `source = 'google'`; `ON CONFLICT (google_books_id)` returns the existing row for concurrent imports |

`CreateBook` creates `source = 'user'` rows and never sets `google_books_id`.
`UpdateBook` skips `source = 'google'` rows. `SearchBooks`, `GetBookByID`,
`ListBooksByAuthor`, `ListBooksByCategory`, and `ListShelfBooks` return
`source` and `published_date_precision` (and `google_books_id`, from which the
service builds `infoLink` for Google rows).

## Pagination totals

Paginated list queries return `total_count` from `count(*) OVER ()`, which the
service copies into `X-Total-Count`. A window count is attached to rows, so a
page past the end returns no rows and no total. When a page is empty and
`offset > 0`, the service reruns the same query with `limit = 1, offset = 0`
to read the total. Do not report `0` in that case.

## Search patterns

`SearchBooks`, `ListAuthors`, and `ListCategories` escape `\`, `%`, and `_` in
user text before building the `ILIKE` pattern, so those characters match
literally.

## Writes handled outside sqlc

`fn_move_book_to_shelf` and `fn_update_reading_progress` are called directly
with `pgx.Tx` in the repository layer. They return a composite
`reading_progress` row and own the row locking, status-shelf synchronization,
and history inserts. Keeping them out of sqlc avoids duplicating transaction
logic in Go and keeps the database rules visible in `database/migrations`.

## Regeneration

A local sqlc install is not required. The pinned Docker tool service runs the
same generator:

```bash
# From Team_Project/, after creating .env
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc
```

With a local sqlc install:

```bash
cd backend
sqlc generate
```

The sqlc schema input is `database/schema/schema.sql`, a reduced table/column
mirror. The migrations remain canonical.
