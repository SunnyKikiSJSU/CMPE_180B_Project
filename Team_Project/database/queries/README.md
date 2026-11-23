# SQL Query Catalog

These files are compiled by `sqlc` into `backend/internal/db`. Each query has a
`-- name:` annotation; do not rename a query without regenerating the Go code
and updating its repository call.

## Files

| File | Responsibility |
| --- | --- |
| `auth.sql` | User lookup by email/ID, creation, password update |
| `users.sql` | Profile read/update, account stats, account deletion |
| `books.sql` | Library search, book detail, catalog writes, Open Library import and dedupe lookups, author/category upserts |
| `authors.sql` | Author list/detail and author-scoped books |
| `categories.sql` | Category list/detail and category-scoped books |
| `shelves.sql` | Shelf CRUD, shelf-book listing, membership add/remove |
| `progress.sql` | Progress/history reads |
| `reviews.sql` | Review reads and owner-scoped writes |
| `ratings.sql` | Rating summary, distribution, and owner-scoped writes |
| `reports.sql` | Report queries q04-q09 |

## Required query catalog

Purpose, parameters, definitions, and seed results:
[`docs/database/query-catalog.md`](../../docs/database/query-catalog.md). The
API serves the exact SQL at `GET /api/reports/queries`; the mapping below is
`catalogSpec` in `backend/internal/services/reports.go`.

| ID | Query | Endpoint |
| --- | --- | --- |
| `q01` | `SearchBooks` | `GET /api/books` |
| `q02` | `GetBookByID` | `GET /api/books/{bookId}` |
| `q03` | `ListShelfBooks` | `GET /api/shelves/{shelfId}/books` |
| `q04` | `TopRatedBooks`, `RatingSummaryByCategory`, `RatingSummaryByAuthor` | `GET /api/reports/top-rated` (the category/author summaries are evidence SQL with no endpoint) |
| `q05` | `PopularBooks` | `GET /api/reports/popular` |
| `q06` | `RecommendedBooks` | `GET /api/reports/recommendations` |
| `q07` | `UserReadingStats`, `ReadingReportSummary`, `FavoriteCategories`, `RecentReadingActivity` | `GET /api/reports/user-stats`, `GET /api/reports/reading-history` |
| `q08` | `AuthorRankings`, `CategoryRankings` | `GET /api/reports/authors`, `GET /api/reports/categories` |
| `q09` | `ReviewActivity` | `GET /api/reports/review-activity` |
| `q10` | `GetRatingDistribution` | `GET /api/books/{bookId}/ratings` |

## Open Library queries

Outside search runs in the Go `internal/openlibrary` client, not in SQL. These
queries connect its results to the library:

| Query | Endpoint | Role |
| --- | --- | --- |
| `ListBooksByExternalIDs` | `GET /api/external/books` | One lookup per result page; fills `bookId` for works (by work ID, or any ISBN-13 of the work) already stored |
| `FindImportedBook` | `POST /api/books/import` | Dedupe by work ID before calling Open Library, then by work ID or ISBN-13 after |
| `ImportOpenLibraryBook` | `POST /api/books/import` | Inserts `source = 'openlibrary'`; `ON CONFLICT (openlibrary_id)` returns the existing row for concurrent imports, and `inserted` (`xmax = 0`) tells the two cases apart |

`CreateBook` creates `source = 'user'` rows and never sets an outside ID.
`UpdateBook` skips imported rows (`source` `google` or `openlibrary`).
`SearchBooks`, `GetBookByID`, `ListBooksByAuthor`, `ListBooksByCategory`, and
`ListShelfBooks` return `source`, `published_date_precision`,
`google_books_id`, and `openlibrary_id`; the service builds `infoLink` from
the outside ID.

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
