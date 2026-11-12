# Backend Implementation Plan — Book Metaverse Portal

**Course:** CMPE-180B  
**Plan date:** November 10, 2026  
**Scope:** Complete the Go REST API, PostgreSQL rules, seed data, tests, and the
frontend wiring needed to turn the finished UI shell into a working
database-driven application.

This plan is the implementation contract for the backend. It is written against
the repository as it exists today, not against the earlier drafts.

---

## 1. Current State (Verified)

| Area | What exists now | What is missing |
| --- | --- | --- |
| Frontend | Next.js 15 app with 11 routes, shared components, `bookService`, `readingService`, `apiClient`, and domain types | Pages render static/empty states; no TanStack Query provider; login writes `localStorage`; only 8 API calls are declared |
| Backend | `backend/cmd/server/main.go` with `/`, `/healthz`, `/readyz`; `database/sql` + pgx stdlib; Dockerfile | No auth, handlers, services, repositories, middleware, validation, or business endpoints |
| Database | Migrations `000001`-`000006`, 66 named queries in `database/queries/`, reduced sqlc schema mirror, and seeds `001`-`004` | Phase 1 is verified on a clean database (2026-11-10); `000006` (book sources for Google Books) verified the same day; no schema dictionary or evidence package yet |
| Docker | `postgres`, `api`, `migrate`, `seed`, and pinned `sqlc` (`tools`) services | No `web` service; API has no auth/CORS configuration |
| Docs | `docs/api/openapi.yaml`, `errors.md`, and `examples.md`; root `README.md`, `PLAN.md`, `DEVELOPMENT.md`, `infra/README.md` | No schema dictionary, query catalog, or evidence package |
| Database engine | PostgreSQL 18 is implemented in the migration, Compose, and Terraform | The older Milestone Plan still says MySQL; that wording is superseded and must be corrected in docs |

### Frontend calls that already define part of the contract

| Frontend caller | Current URL | Expected response |
| --- | --- | --- |
| `bookService.search` | `GET /api/books?q=` | `Book[]` |
| `bookService.getById` | `GET /api/books/{id}` | `Book` |
| `bookService.getReviews` | `GET /api/books/{id}/reviews` | `Review[]` |
| `bookService.rate` | `POST /api/books/{id}/ratings` | `Rating` |
| `readingService.getShelves` | `GET /api/shelves` | `Shelf[]` |
| `readingService.getProgress` | `GET /api/reading-progress` | `ReadingProgress[]` |
| `readingService.updateProgress` | `PATCH /api/reading-progress/{bookId}` | `ReadingProgress` |
| `readingService.getReport` | `GET /api/reports/reading-history` | `ReadingReport` |

These shapes must not break. The backend will return these exact objects, with
additional fields where useful. List endpoints return JSON arrays and expose
pagination through response headers so the existing service typings continue to
work.

---

## 2. Decisions to Freeze Before Coding

1. **PostgreSQL 18 is the database.** Do not implement MySQL. Update any
   remaining MySQL references in `Book-Metaverse-Portal-Milestone-Plan.md` and
   presentation material to PostgreSQL equivalents (`tsvector`/`pg_trgm`/GIN
   instead of `FULLTEXT`, PL/pgSQL functions instead of MySQL procedures).
2. **Explicit SQL is required.** No ORM. `sqlc` generates typed query code from
   files under `database/queries/`.
3. **Layered backend:**
   `HTTP handler -> service -> repository -> pgxpool -> PostgreSQL`.
4. **Router:** `chi/v5`. **Driver:** `pgx/v5` with `pgxpool` (replace the
   current `database/sql` connection in `main.go`).
5. **Auth:** Argon2id password hashes, HS256 JWT, 60-minute access token. The
   token is delivered as an `HttpOnly` cookie for the browser and may also be
   read from `Authorization: Bearer` for tests and CLI use. No refresh tokens in
   the MVP.
6. **JSON convention:** camelCase keys, UUID strings, RFC 3339 UTC timestamps.
7. **List convention:** raw arrays plus `X-Total-Count`, `X-Page`, and
   `X-Page-Size` response headers. Default page size 20, maximum 100.
8. **Errors:** one envelope for every non-2xx response (Section 4.3).
9. **Catalog writes:** any authenticated user may add or update catalog
   records. There is no admin role in the MVP; document this limitation.
10. **All workspace data is scoped to the authenticated user.** Ownership is
    enforced in SQL, not only in handlers.
11. **Google Books is the discovery source; PostgreSQL stays the source of
    truth.** Users search Google Books through the API
    (`GET /api/external/books`); nothing is stored by a search. A result is
    imported into `books` (`POST /api/books/import`) only when a user acts on
    it, so every shelf, progress, rating, and review row still references a
    local `book_id` UUID. The Google volume ID is stored in
    `books.google_books_id` for deduplication and is never used as a key.
    Library search (`GET /api/books`) keeps the PostgreSQL full-text search
    for books already in the database. See Section 6.4.

---

## 3. Target Architecture

```text
Browser (Next.js)
      |
      |  HTTPS / JSON, HttpOnly cookie or Bearer token
      v
Go API (chi)
      |
      +-- middleware: request ID, logging, recovery, CORS, body limit,
      |               rate limit, authentication
      |
      +-- handlers     parse HTTP, validate input, map errors
      +-- services     business rules, ownership, transactions
      +-- repositories typed sqlc queries + hand-written pgx for functions
      +-- googlebooks  HTTPS client for Google Books search/volume lookup only
      |                (server-side key, timeout, short in-memory cache)
      |
      v
PostgreSQL 18
      +-- tables and constraints
      +-- search indexes (tsvector, pg_trgm, GIN)
      +-- functions and triggers (summaries, defaults, validation)
      +-- immutable reading history
```

### Backend directory layout

```text
backend/
├── cmd/
│   └── server/main.go                 # Entry point: load config, open pool, build router, start/stop HTTP
├── internal/
│   ├── googlebooks/
│   │   ├── client.go                  # Search and GetVolume over HTTPS; key, country, timeout; maps Google errors
│   │   ├── normalize.go               # Volume -> ExternalBook/import params (dates, HTML, ISBN, categories, https covers)
│   │   └── cache.go                   # Small TTL cache of search pages to save quota
│   ├── config/config.go               # Reads env vars, validates them, returns one typed Config value
│   ├── database/
│   │   ├── pool.go                    # Creates/closes the pgxpool and exposes ping/stat helpers
│   │   └── migrate.go                 # Reads schema_migrations so /readyz knows the schema is current
│   ├── db/                            # sqlc-generated query code; committed, never hand-edited
│   ├── auth/
│   │   ├── password.go                # Argon2id hashing, PHC encoding, constant-time verification
│   │   └── jwt.go                     # Issues and validates HS256 access tokens and claims
│   ├── middleware/
│   │   ├── auth.go                    # Reads cookie/Bearer, validates JWT, injects the user ID
│   │   ├── cors.go                    # Exact-origin allowlist, credentials, exposed pagination headers
│   │   ├── logging.go                 # Structured request logs: route, status, duration, user ID
│   │   ├── ratelimit.go               # Per-IP token bucket; stricter bucket for login/register
│   │   └── requestid.go               # Generates/propagates X-Request-ID into logs and error bodies
│   ├── httpx/
│   │   ├── decode.go                  # Bounded JSON decoding, unknown-field rejection, field validation
│   │   ├── response.go                # JSON success responses, pagination headers, content type
│   │   └── errors.go                  # Maps domain errors to HTTP statuses and the error envelope
│   ├── domain/
│   │   ├── models.go                  # API-facing DTOs, kept separate from sqlc row structs
│   │   └── errors.go                  # ErrNotFound, ErrConflict, ErrStaleVersion, ErrValidation, etc.
│   ├── repositories/
│   │   ├── users.go                   # User/profile SQL and sqlc adapters
│   │   ├── catalog.go                 # Book search/detail/create plus author/category queries
│   │   ├── shelves.go                 # Shelf CRUD and shelf-book membership queries
│   │   ├── progress.go                # Progress/history queries and fn_* function calls
│   │   ├── reviews.go                 # Rating/review queries and summary reads
│   │   └── reports.go                 # q04-q10 report queries
│   ├── services/
│   │   ├── auth.go                    # Register/login/logout rules, hashing, token issuance
│   │   ├── users.go                   # Profile updates, password change, account deletion
│   │   ├── catalog.go                 # Library search, author/category upserts, catalog validation, Google search + import
│   │   ├── shelves.go                 # Shelf rules and fn_move_book_to_shelf transactions
│   │   ├── progress.go                # Page/status validation and fn_update_reading_progress calls
│   │   ├── reviews.go                 # Rating/review upserts and the combined review+rating transaction
│   │   └── reports.go                 # Report parameter validation and result mapping
│   ├── handlers/
│   │   ├── auth.go                    # HTTP layer for /api/auth/*
│   │   ├── users.go                   # HTTP layer for /api/users/me*
│   │   ├── books.go                   # HTTP layer for /api/books*, /api/books/import, /api/search, /api/external/books
│   │   ├── authors.go                 # HTTP layer for /api/authors*
│   │   ├── categories.go              # HTTP layer for /api/categories*
│   │   ├── shelves.go                 # HTTP layer for /api/shelves*
│   │   ├── progress.go                # HTTP layer for /api/reading-progress* and /api/reading-history
│   │   ├── reviews.go                 # HTTP layer for /api/books/{id}/reviews* and /api/reviews/*
│   │   ├── ratings.go                 # HTTP layer for /api/books/{id}/ratings*
│   │   ├── reports.go                 # HTTP layer for /api/reports/*
│   │   └── health.go                  # HTTP layer for /, /healthz, /readyz
│   └── router/router.go               # Declares every route, auth group, and middleware order
├── sqlc.yaml                          # sqlc schema/query paths, pgx output, UUID type override
├── go.mod / go.sum                    # Go module definition and locked dependency checksums
└── Dockerfile                         # Multi-stage build; runs the compiled API as non-root
```

Rules:

- Handlers never contain SQL.
- Services own transaction boundaries and authorization decisions.
- Repositories own SQL and row-to-domain mapping.
- Handlers validate transport-level shape; services validate business rules.
- Database constraints remain the final authority.

---

## 4. API Contract

### 4.1 Conventions

| Concern | Decision |
| --- | --- |
| Base path | `/api` |
| Content type | `application/json; charset=utf-8` |
| IDs | UUID v4 strings |
| Timestamps | RFC 3339 UTC, e.g. `2026-11-10T18:22:04Z` |
| Naming | camelCase JSON; snake_case SQL |
| Authentication | `bm_session` HttpOnly cookie or `Authorization: Bearer <jwt>` |
| Pagination | `?page=1&pageSize=20`; response headers `X-Total-Count`, `X-Page`, `X-Page-Size` |
| Sorting | `?sort=relevance\|newest\|rating\|title\|popular` |
| Date filters | `?from=2026-01-01&to=2026-12-31` (inclusive start, exclusive end) |
| Request body limit | 1 MiB |

### 4.2 Authentication

- `POST /api/auth/register` creates a user. The database trigger creates the
  profile and four default shelves in the same transaction.
- `POST /api/auth/login` verifies the Argon2id hash and issues the JWT.
- The JWT contains `sub` (user ID), `email`, `iat`, `exp`, `iss`
  (`book-metaverse`). Reject `alg: none` and any algorithm other than HS256.
- Cookie flags: `HttpOnly; Path=/; SameSite=Lax`; add `Secure` when
  `COOKIE_SECURE=true`.
- Public read endpoints: library search, Google Books search, book detail,
  reviews, rating summaries, authors, categories, and public reports.
- Importing a Google volume (`POST /api/books/import`) requires authentication.
- All shelf, progress, review-write, rating-write, user, and personal report
  endpoints require authentication.

### 4.3 Error envelope

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "A review for this book already exists.",
    "details": { "constraint": "reviews_user_book_unique" },
    "requestId": "01J9Z8R6K3W4T2M1"
  }
}
```

| Code | HTTP | When |
| --- | --- | --- |
| `VALIDATION_ERROR` | 400 | Malformed JSON, missing field, bad range, bad enum |
| `UNAUTHENTICATED` | 401 | Missing/invalid/expired token |
| `FORBIDDEN` | 403 | Authenticated but action is not allowed |
| `NOT_FOUND` | 404 | Resource does not exist or is not owned by the caller |
| `CONFLICT` | 409 | Duplicate email, review, rating, shelf name, or status shelf |
| `STALE_VERSION` | 409 | Optimistic progress version mismatch |
| `RATE_LIMITED` | 429 | Login/register, Google search/import, or general rate limit exceeded |
| `UPSTREAM_UNAVAILABLE` | 502 | Google Books failed, timed out, or returned an unusable response |
| `UPSTREAM_RATE_LIMITED` | 503 | Server's Google Books quota exhausted; `Retry-After` set |
| `DATABASE_UNAVAILABLE` | 503 | Pool cannot reach PostgreSQL |
| `INTERNAL` | 500 | Unexpected failure; log details, never return them |

PostgreSQL error mapping:

| SQLSTATE | Meaning | API result |
| --- | --- | --- |
| `23505` | unique violation | `CONFLICT` with constraint-aware message |
| `23503` | foreign key violation | `VALIDATION_ERROR` |
| `23514` | check violation | `VALIDATION_ERROR` |
| `22023` | invalid parameter (raised by triggers) | `VALIDATION_ERROR` |
| `BM001` | stale progress version (raised by `fn_update_reading_progress`) | `STALE_VERSION`; never retried |
| `40001` / `40P01` | serialization failure / deadlock | retry once, then `CONFLICT` |
| `P0001` | trigger `RAISE` | map by message/code metadata |
| connection errors | pool/driver | `DATABASE_UNAVAILABLE` |

### 4.4 Endpoint matrix

Every row below states **why the path exists**, not only what it returns. This
matters for implementation and review: a path with no clear purpose should not
be built. Any path not listed here must return `404`; there are no hidden debug
or admin routes in the MVP.

#### Health

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/healthz` | none | Liveness probe: confirms the process is running without touching the database |
| `GET` | `/readyz` | none | Readiness probe: confirms the pool connects and the schema migration version is current |
| `GET` | `/` | none | Small service-metadata response for smoke checks and load-balancer tests |

#### Authentication and users

| Method | Path | Auth | Purpose | Request | Response |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/api/auth/register` | none | Creates the account, profile, and four default shelves, then starts a session | `{email,password,displayName}` | `201 {user,accessToken}` + cookie |
| `POST` | `/api/auth/login` | none | Verifies email/password and starts a session | `{email,password}` | `200 {user,accessToken}` + cookie |
| `POST` | `/api/auth/logout` | yes | Ends the browser session by clearing the cookie | — | `204`, cookie cleared |
| `GET` | `/api/auth/me` | yes | Returns the current session user; called by `AuthBoundary` on page load | — | `{user}` |
| `GET` | `/api/users/me` | yes | Returns the signed-in user's full profile and reading counts for Settings | — | `User` with profile and counts |
| `PATCH` | `/api/users/me` | yes | Updates editable profile fields (name, avatar, bio, favorite genre) | `{displayName?,avatarUrl?,bio?,favoriteGenre?}` | `User` |
| `PUT` | `/api/users/me/password` | yes | Changes the password after verifying the current one | `{currentPassword,newPassword}` | `204` |
| `DELETE` | `/api/users/me` | yes | Permanently deletes the account and cascades all owned data | `{"confirm":true}` | `204` |

`User` shape:

```json
{
  "id": "uuid",
  "email": "reader@example.com",
  "displayName": "Reader",
  "avatarUrl": null,
  "createdAt": "2026-11-10T18:00:00Z",
  "profile": { "bio": null, "favoriteGenre": null, "updatedAt": "..." },
  "stats": {
    "shelfCount": 4,
    "booksCompleted": 0,
    "reviewsWritten": 0,
    "ratingsGiven": 0
  }
}
```

#### Catalog

| Method | Path | Auth | Purpose | Request / query | Response |
| --- | --- | --- | --- | --- | --- |
| `GET` | `/api/external/books` | optional | Google Books search for discovery; powers the main search on Explore, Catalog, and the dashboard. Stores nothing | `q,title,author,category,isbn,sort=relevance\|newest,page,pageSize<=40` | `ExternalBook[]` + pagination headers (`X-Total-Count` is Google's estimate) |
| `POST` | `/api/books/import` | yes | Turns a Google result into a local book so it can be shelved, rated, reviewed, or tracked. Idempotent | `{googleBooksId}` | `201 Book` (imported) or `200 Book` (already in library) |
| `GET` | `/api/books` | optional | Library search over books already in PostgreSQL (seed, imported, user-created); the "In the library" tab, filters, and evidence query `q01` | `q,title,authorId,categoryId,sort,page,pageSize` | `Book[]` + pagination headers |
| `GET` | `/api/search` | optional | Alias of `/api/books` (library search) for the existing frontend URL | alias of `/api/books` | same |
| `GET` | `/api/books/{bookId}` | optional | Book detail: metadata, authors, categories, summary, and the caller's progress/rating | — | `Book` with `myProgress`/`myRating` when authenticated |
| `POST` | `/api/books` | yes | Adds a book by hand when Google has no match (`source = user`) | book metadata + `authors: string[]` + `categories: string[]` | `201 Book` |
| `PATCH` | `/api/books/{bookId}` | yes | Corrects seed or user-created metadata without touching user activity; Google rows return `403` | partial metadata | `Book` |
| `GET` | `/api/authors` | none | Lists/searches authors and their book counts for the Authors page and filters | `q,page,pageSize` | `Author[]` with `bookCount` |
| `GET` | `/api/authors/{authorId}` | none | Author detail page with that author's books | `page,pageSize` | `Author` with `books: Book[]` |
| `GET` | `/api/categories` | none | Lists/searches categories and counts for filters and the Authors & categories page | `q,page,pageSize` | `Category[]` with `bookCount` |
| `GET` | `/api/categories/{categoryId}` | none | Category detail page with books in that category | `page,pageSize` | `Category` with `books: Book[]` |

`Book` shape (superset of the current frontend type):

```json
{
  "id": "uuid",
  "title": "The Night Circus",
  "author": "Erin Morgenstern",
  "authors": [{ "id": "uuid", "name": "Erin Morgenstern" }],
  "categories": [{ "id": "uuid", "name": "Fantasy" }],
  "description": "...",
  "publishedDate": "2011-09-13",
  "publisher": "Doubleday",
  "pageCount": 387,
  "isbn13": "9780385534635",
  "coverUrl": "https://...",
  "source": "google",
  "infoLink": "https://books.google.com/books?id=zyTCAlFPjgYC",
  "publishedDatePrecision": "day",
  "averageRating": 4.5,
  "ratingCount": 12,
  "reviewCount": 4,
  "myProgress": null,
  "myRating": null
}
```

`author` is the primary author name and is retained for compatibility with the
existing `Book` TypeScript interface; it is `"Unknown author"` when a book has
no authors. `authors` and `categories` carry the full many-to-many data.
`source` is `seed`, `google`, or `user`. `infoLink` is built from
`google_books_id` (`https://books.google.com/books?id=<id>`) and is set only
for `source = google`; seed rows hold synthetic IDs and never get a link.
`averageRating`, `ratingCount`, and `reviewCount` are always this site's own
users' data, never Google's.

`ExternalBook` shape (a Google search result; not stored):

```json
{
  "googleBooksId": "zyTCAlFPjgYC",
  "bookId": null,
  "title": "The Night Circus",
  "subtitle": null,
  "authors": ["Erin Morgenstern"],
  "categories": ["Fantasy"],
  "description": "Plain text, HTML stripped",
  "publishedDate": "2011-09-13",
  "publisher": "Doubleday",
  "pageCount": 387,
  "isbn13": "9780385534635",
  "coverUrl": "https://books.google.com/books/content?id=...",
  "infoLink": "https://books.google.com/books?id=zyTCAlFPjgYC",
  "source": "google"
}
```

`bookId` is the local UUID when the volume, or a book with the same ISBN-13,
is already in the library. The frontend uses it directly; otherwise it calls
`POST /api/books/import` first and uses the returned `id`.

Search and import flow:

```text
user types a query
  -> GET /api/external/books        (Google, cached, nothing stored)
  -> results with bookId or null
user clicks "Add to shelf" / "Rate" / "Start reading" on a result
  -> bookId null? POST /api/books/import {googleBooksId} -> Book.id
  -> POST /api/shelves/{shelfId}/books {bookId}  (unchanged endpoints)
```

#### Ratings and reviews

| Method | Path | Auth | Purpose | Request | Response |
| --- | --- | --- | --- | --- | --- |
| `GET` | `/api/books/{bookId}/ratings` | none | Returns the average, count, and 1-5 distribution shown on the book detail page | — | `{averageRating,ratingCount,distribution}` |
| `POST` | `/api/books/{bookId}/ratings` | yes | Creates or updates the caller's rating; the summary trigger recalculates immediately | `{value:1..5}` | `Rating` (upsert; includes `summary`) |
| `DELETE` | `/api/books/{bookId}/ratings` | yes | Removes the caller's rating and recalculates the summary | — | `204` |
| `GET` | `/api/books/{bookId}/reviews` | none | Public, paginated review list for a book | `page,pageSize,sort=newest\|rating` | `Review[]` |
| `POST` | `/api/books/{bookId}/reviews` | yes | Creates the caller's review; one review per user per book | `{title?,body}` | `201 Review`; `409` if one exists |
| `PUT` | `/api/books/{bookId}/review` | yes | Review-form save: rating + review in one transaction, so neither is half-written | `{rating?,title?,body}` | `{review,rating,summary}` |
| `PATCH` | `/api/reviews/{reviewId}` | owner | Edits only the owner's review text/title | `{title?,body}` | `Review` |
| `DELETE` | `/api/reviews/{reviewId}` | owner | Deletes only the owner's review; summary triggers recalculate counts | — | `204` |
| `GET` | `/api/users/me/reviews` | yes | Lists the caller's reviews for the Reviews page | `page,pageSize` | `Review[]` |
| `GET` | `/api/users/me/ratings` | yes | Lists the caller's ratings for the Ratings & reviews page | `page,pageSize` | `Rating[]` |

`Review` shape:

```json
{
  "id": "uuid",
  "bookId": "uuid",
  "userId": "uuid",
  "body": "A wonderful book.",
  "title": "Loved it",
  "rating": 5,
  "user": { "id": "uuid", "displayName": "Reader", "avatarUrl": null },
  "createdAt": "2026-11-10T18:00:00Z",
  "updatedAt": "2026-11-10T18:00:00Z"
}
```

`Rating` shape:

```json
{
  "bookId": "uuid",
  "userId": "uuid",
  "value": 5,
  "createdAt": "2026-11-10T18:00:00Z",
  "updatedAt": "2026-11-10T18:00:00Z",
  "summary": { "averageRating": 4.5, "ratingCount": 12, "reviewCount": 4 }
}
```

#### Shelves

| Method | Path | Auth | Purpose | Request | Response |
| --- | --- | --- | --- | --- | --- |
| `GET` | `/api/shelves` | yes | Lists the caller's status and custom shelves with book counts | — | `Shelf[]` with counts |
| `POST` | `/api/shelves` | yes | Creates a custom shelf; status shelves exist only from registration | `{name,type?}` (`type` defaults `custom`) | `201 Shelf` |
| `PATCH` | `/api/shelves/{shelfId}` | owner | Renames a custom shelf | `{name}` | `Shelf` |
| `DELETE` | `/api/shelves/{shelfId}` | owner | Deletes a custom shelf; status shelves are protected | — | `204` (custom shelves only) |
| `GET` | `/api/shelves/{shelfId}/books` | yes | Lists the books on one shelf with their progress metadata | `page,pageSize,sort` | `Book[]` |
| `POST` | `/api/shelves/{shelfId}/books` | yes | Adds a book; adding to a status shelf moves it and updates progress/history | `{bookId}` | `ShelfBookResult` (add or move) |
| `PATCH` | `/api/shelves/{shelfId}/books/{bookId}` | yes | Moves a book from this shelf to another shelf in one transaction | `{targetShelfId}` | `ShelfBookResult` |
| `DELETE` | `/api/shelves/{shelfId}/books/{bookId}` | yes | Removes the book from this shelf only; the catalog book is not deleted | — | `204` |

`Shelf` shape:

```json
{ "id": "uuid", "name": "Want to read", "type": "want_to_read", "count": 12, "createdAt": "..." }
```

`ShelfBookResult`:

```json
{ "shelfId": "uuid", "bookId": "uuid", "progress": { }, "historyEvent": "shelf_moved" }
```

Rules:

- Every user gets four status shelves at registration: Want to read,
  Currently reading, Finished, Paused.
- A book may be on any number of **custom** shelves.
- A book may be on at most **one status shelf** at a time. Moving it to a status
  shelf removes it from the other status shelves in the same transaction.
- Default status shelves cannot be deleted or renamed in the MVP. Their names
  are stable.
- Adding to a custom shelf is additive and does not change reading status.

#### Reading progress and history

| Method | Path | Auth | Purpose | Request | Response |
| --- | --- | --- | --- | --- | --- |
| `GET` | `/api/reading-progress` | yes | Lists all reading progress for the caller; powers the dashboard and Progress page | `status?,page,pageSize` | `ReadingProgress[]` |
| `GET` | `/api/reading-progress/{bookId}` | yes | Returns one book's current page/status for the detail page | — | `ReadingProgress` |
| `PATCH` | `/api/reading-progress/{bookId}` | yes | Partial update; the server recomputes percent and completion | `{currentPage?,percentComplete?,status?,version?}` | `ReadingProgress` |
| `PUT` | `/api/reading-progress/{bookId}` | yes | Full replacement of one progress record with the same validation | same, full update | `ReadingProgress` |
| `DELETE` | `/api/reading-progress/{bookId}` | yes | Removes the caller's progress record for a book | — | `204` |
| `GET` | `/api/reading-history` | yes | Returns the append-only reading timeline for the caller or one book | `bookId?,limit=20` | `ReadingHistoryEvent[]` |

`ReadingProgress` shape:

```json
{
  "bookId": "uuid",
  "status": "reading",
  "currentPage": 85,
  "totalPages": 200,
  "percentComplete": 42.5,
  "version": 3,
  "updatedAt": "2026-11-10T18:00:00Z",
  "book": { "id": "uuid", "title": "...", "author": "...", "coverUrl": "..." }
}
```

Rules:

- `percentComplete` is computed by the server from `currentPage` and
  `books.page_count`. A client-supplied percentage is accepted only when the book
  has no page count.
- `currentPage` may not exceed `books.page_count` when that value is known.
- Reaching the last page sets `status = completed` automatically and moves the
  book to the Finished shelf in the same transaction.
- Every status change writes one immutable `reading_history` row.
- A supplied `version` enables optimistic locking. Missing `version` falls back
  to `SELECT ... FOR UPDATE` and last-write-wins with an audit event.

#### Reports

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/reports/reading-history` | yes | Dashboard report that matches the existing `ReadingReport` frontend type: books read, pages, favorite categories, recent activity |
| `GET` | `/api/reports/user-stats` | yes | Detailed personal totals (completed, in progress, pages, average rating, reviews written) for the Reports page |
| `GET` | `/api/reports/popular` | none | Most-saved/most-rated books inside a time window (`days`) |
| `GET` | `/api/reports/top-rated` | none | Highest-rated books filtered by a minimum rating count |
| `GET` | `/api/reports/recommendations` | yes | Books suggested from categories the caller finished or rated highly, excluding already-read books |
| `GET` | `/api/reports/authors` | none | Author leaderboard ranked by rating/reader activity |
| `GET` | `/api/reports/categories` | none | Category leaderboard ranked by rating/reader activity |
| `GET` | `/api/reports/review-activity` | none | Daily review and rating counts for trend charts |
| `GET` | `/api/reports/queries` | none | Returns the report catalog and static SQL text so the UI can show "(view SQL)" |

`GET /api/reports/reading-history` response, matching the existing frontend
type:

```json
{
  "booksRead": 12,
  "pagesRead": 3450,
  "favoriteCategories": [{ "id": "uuid", "name": "Fantasy" }],
  "recentActivity": [
    {
      "bookId": "uuid",
      "status": "reading",
      "currentPage": 85,
      "totalPages": 200,
      "updatedAt": "2026-11-10T18:00:00Z"
    }
  ]
}
```

Pages-read definition: for completed books use `books.page_count`; for
in-progress and paused books use `reading_progress.current_page`; never count
the same book twice. This rule must be documented in the report SQL and in
`docs/database/query-catalog.md`.

---

## 5. Database Implementation

### 5.1 Migration plan

| Migration | Purpose | Contents |
| --- | --- | --- |
| `000002_search_and_extensions` | Makes keyword, author, and category search fast and typo-tolerant | `pg_trgm`; generated `books.search_vector`; GIN and trigram indexes |
| `000003_domain_rules` | Moves derived and protected data rules into the database so every client path gets the same behavior | `updated_at` triggers; default-shelf/profile creation; partial unique status-shelf index; rating validation; rating/review summary maintenance; progress validation; history immutability; `version` columns; author/category dedupe and case-insensitive uniqueness; backfills |
| `000004_reporting_indexes` | Keeps report and dashboard queries off sequential scans as the dataset grows | Additional composite/partial indexes used by reports and dashboards |
| `000005_progress_and_history_fixes` | Fixes progress transitions, stale-version SQLSTATE, and cascade deletes found in Phase 1 testing | Replaces three `000003` functions |
| `000006_book_sources` | Lets imported Google books be told apart from seed and user-created books, and stores partial Google dates | `books.source`, `books.published_date_precision`, `books.imported_at`, four CHECK constraints, backfill |

Each migration must have a matching `.down.sql` and must leave a clean database
in the same state as a fresh `migrate up`.

#### 000002 — Search

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE books
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;

CREATE INDEX books_search_vector_gin_idx
    ON books USING GIN (search_vector);

CREATE INDEX books_title_trgm_idx
    ON books USING GIN (title gin_trgm_ops);

CREATE INDEX authors_name_trgm_idx
    ON authors USING GIN (name gin_trgm_ops);

CREATE INDEX categories_name_trgm_idx
    ON categories USING GIN (name gin_trgm_ops);
```

`search_vector` makes title matches rank above description matches. Author and
category search use `pg_trgm` membership queries through the junction tables so
typos and partial names still match.

#### 000003 — Domain rules

**A. `updated_at` maintenance**

One reusable trigger function:

```sql
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;
```

Attach `BEFORE UPDATE` triggers to `user_profiles`, `reading_progress`,
`ratings`, and `reviews`.

**B. Default profile and shelves on registration**

```sql
CREATE OR REPLACE FUNCTION fn_create_user_defaults(p_user_id uuid)
RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO user_profiles (user_id)
    VALUES (p_user_id)
    ON CONFLICT (user_id) DO NOTHING;

    INSERT INTO shelves (user_id, name, shelf_type)
    SELECT p_user_id, v.name, v.shelf_type
    FROM (VALUES
        ('Want to read', 'want_to_read'),
        ('Currently reading', 'currently_reading'),
        ('Finished', 'finished'),
        ('Paused', 'paused')
    ) AS v(name, shelf_type)
    WHERE NOT EXISTS (
        SELECT 1 FROM shelves s
        WHERE s.user_id = p_user_id
          AND s.shelf_type = v.shelf_type
    );
END;
$$;

CREATE OR REPLACE FUNCTION fn_users_after_insert()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM fn_create_user_defaults(NEW.user_id);
    RETURN NEW;
END;
$$;

CREATE TRIGGER users_create_defaults
    AFTER INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION fn_users_after_insert();
```

Backfill existing users with `SELECT fn_create_user_defaults(user_id) FROM users;`.

**C. One status shelf per user**

```sql
CREATE UNIQUE INDEX shelves_one_per_status_type_idx
    ON shelves (user_id, shelf_type)
    WHERE shelf_type <> 'custom';
```

Before creating the index, deduplicate pre-existing status shelves: keep the
oldest shelf per `(user_id, shelf_type)`, move `user_shelf_items` to the kept
shelf with `ON CONFLICT DO NOTHING`, then delete duplicates.

**D. Rating validation trigger**

```sql
CREATE OR REPLACE FUNCTION fn_validate_rating()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.rating_value < 1 OR NEW.rating_value > 5 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'rating_value must be between 1 and 5';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER ratings_validate_value
    BEFORE INSERT OR UPDATE ON ratings
    FOR EACH ROW EXECUTE FUNCTION fn_validate_rating();
```

The existing `CHECK (rating_value BETWEEN 1 AND 5)` remains as the last line of
defense. The trigger provides a clearer error path and satisfies the
trigger-validation requirement.

**E. Rating and review summary maintenance**

One refresh function keeps both counts correct:

```sql
CREATE OR REPLACE FUNCTION fn_refresh_book_rating_summary(p_book_id uuid)
RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO book_rating_summaries
        (book_id, average_rating, rating_count, review_count, updated_at)
    SELECT
        p_book_id,
        coalesce(round(avg(rating_value)::numeric, 2), 0),
        count(*),
        (SELECT count(*) FROM reviews WHERE book_id = p_book_id),
        now()
    FROM ratings
    WHERE book_id = p_book_id
    ON CONFLICT (book_id) DO UPDATE
        SET average_rating = EXCLUDED.average_rating,
            rating_count   = EXCLUDED.rating_count,
            review_count   = EXCLUDED.review_count,
            updated_at     = now();
END;
$$;
```

Triggers:

- `AFTER INSERT OR UPDATE OR DELETE ON ratings` calls
  `fn_refresh_book_rating_summary(coalesce(NEW.book_id, OLD.book_id))`.
- `AFTER INSERT OR UPDATE OR DELETE ON reviews` calls the same function.
- `AFTER INSERT ON books` inserts a zero summary row.

The aggregate + `ON CONFLICT DO UPDATE` statement takes a row lock on the
summary, so concurrent ratings serialize correctly under `READ COMMITTED`.
`NULL` handling uses `coalesce(NEW.book_id, OLD.book_id)`; `NEW`/`OLD` must be
referenced through separate trigger functions or a guard because PostgreSQL
does not allow `NEW` in a `DELETE` trigger. Create two small trigger functions
(`fn_ratings_summary_trigger`, `fn_reviews_summary_trigger`) for clarity.

Backfill after migration:

```sql
SELECT fn_refresh_book_rating_summary(book_id) FROM books;
```

**F. Reading-progress validation**

`BEFORE INSERT OR UPDATE ON reading_progress`:

1. Load `books.page_count` for `NEW.book_id`.
2. If `page_count` is not null and `NEW.current_page > page_count`, raise
   `22023`.
3. Recompute `NEW.percent_complete` from `current_page / page_count`.
4. If `current_page = page_count`, force `NEW.status = 'completed'`.
5. Leave `updated_at` to the shared trigger.

A companion `AFTER INSERT OR UPDATE` trigger writes a `reading_history` row when
the status changes. To keep event semantics clean, the service writes
`progress_updated`, `status_changed`, `book_completed`, and `shelf_moved` events
inside the same transaction, and the database trigger only blocks invalid
states. Choose one owner: **the service owns history inserts**; the database
enforces validity. Do not implement both or every change will be logged twice.

**G. Immutable history**

```sql
CREATE OR REPLACE FUNCTION fn_block_history_mutation()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'reading_history is append-only';
END;
$$;

CREATE TRIGGER reading_history_immutable
    BEFORE UPDATE OR DELETE ON reading_history
    FOR EACH ROW EXECUTE FUNCTION fn_block_history_mutation();
```

The seed loader must insert history before this trigger is used for destructive
resets; use `TRUNCATE ... CASCADE` in a dedicated reset script rather than
deleting rows.

**H. Optimistic version**

```sql
ALTER TABLE reading_progress
    ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
```

The progress function increments `version` on every update and is called with
the expected version when the client supplies one.

**I. Author and category normalization**

- `authors` currently has no uniqueness rule. Add a case-insensitive unique
  index on `lower(btrim(name))` after deduplicating rows and remapping
  `book_authors`.
- `categories.name` is already unique, but matching is case-sensitive. Normalize
  names (trim, collapse whitespace) in the Google import and catalog service
  before lookup.
  Add a case-insensitive unique index only if a dedupe pass is included.

#### 000004 — Reporting indexes

| Index | Purpose |
| --- | --- |
| `user_shelf_items (shelf_id, added_at DESC)` | Shelf listings ordered by recency |
| `reading_progress (user_id, updated_at DESC)` | Dashboard and progress list |
| `ratings (book_id, rating_value)` | Rating distribution and averages |
| `reviews (user_id, created_at DESC)` | "My reviews" |
| `books (published_date DESC)` | Newest sort (already exists) |
| `reading_history (user_id, created_at DESC)` | Timeline (already exists) |
| `reviews (book_id, created_at DESC)` | Book review lists (already exists) |

Five of these indexes must be benchmarked before/after in a disposable database
using `EXPLAIN (ANALYZE, BUFFERS)`; record the evidence in
`docs/evidence/performance.md`.

#### 000006 — Book sources

| Column | Type | Rule |
| --- | --- | --- |
| `source` | `TEXT NOT NULL DEFAULT 'user'` | `seed`, `google`, or `user` |
| `published_date_precision` | `TEXT` | `year`, `month`, or `day`; present exactly when `published_date` is present |
| `imported_at` | `TIMESTAMPTZ` | Required when `source = 'google'` |

Constraints: `books_source_valid`, `books_google_source_complete` (a Google
row needs `google_books_id` and `imported_at`),
`books_published_date_precision_valid`, and
`books_published_date_precision_matches`.

Backfill: rows whose `google_books_id` starts with `seed-book-` or
`smoke-test-` become `seed`; any other row with a `google_books_id` becomes
`google`; every existing date gets precision `day`. The existing
`books_google_books_id_unique` and `books_isbn13_unique` constraints are the
import deduplication keys; no new index is needed. `book_id` stays the only key
that other tables reference.

### 5.2 Functions used by services

Two SQL functions centralize the cross-table transaction rules. They are
functions rather than MySQL-style procedures because sqlc can consume functions
and because PostgreSQL procedure transaction control is unnecessary here.

```sql
fn_move_book_to_shelf(p_user_id uuid, p_book_id uuid, p_shelf_id uuid)
    RETURNS reading_progress
```

1. Lock the target shelf `FOR UPDATE` and verify `shelf.user_id = p_user_id`.
2. Verify the book exists.
3. If the target is a status shelf, delete the book from the user's other status
   shelves.
4. Upsert `user_shelf_items` with `moved_at = now()`.
5. Create or lock `reading_progress`; map the shelf type to a status:
   `want_to_read -> want_to_read`, `currently_reading -> reading`,
   `finished -> completed`, `paused -> paused`, custom -> unchanged.
6. Increment `version`; insert a `shelf_moved` history row.
7. Return the progress row.

```sql
fn_update_reading_progress(
    p_user_id uuid,
    p_book_id uuid,
    p_current_page integer,
    p_status text,
    p_expected_version integer
) RETURNS reading_progress
```

1. Lock or create the progress row.
2. If `p_expected_version` is supplied and differs, raise SQLSTATE `BM001`
   (mapped to `STALE_VERSION`). `40001` is reserved for real serialization
   failures so the retry helper never retries a stale write (migration
   `000005`).
3. Let the validation trigger recompute percentage and completion status.
4. Synchronize the matching status shelf (remove other status memberships,
   insert the mapped shelf).
5. Increment `version`; insert `progress_updated`, `status_changed`, or
   `book_completed` history.
6. Return the row.

Go services call these functions with `pgx.Tx`, so the transaction boundary is
still explicit in Go and can be rolled back on any error. Rollback tests call
the function, force an error, roll back, and assert that no progress, shelf, or
history row changed.

### 5.3 Query catalog (minimum eight)

Files live in `database/queries/` and are compiled by sqlc. Each key query has a
matching example result in `docs/database/query-catalog.md`.

| ID | File | Purpose | Required SQL features |
| --- | --- | --- | --- |
| `q01` | `books.sql` | Library search by title, author, category, keyword (books already in PostgreSQL) | CTE, `EXISTS`, FTS, trigram, relevance rank, many-to-many joins |
| `q02` | `books.sql` | Book detail with authors, categories, summary, recent reviews | Multiple joins, aggregates, `json_agg`, `LIMIT` subquery |
| `q03` | `shelves.sql` | User shelf with metadata, progress, latest activity | Inner/outer joins, aggregate, `LEFT JOIN LATERAL` |
| `q04` | `reports.sql` | Average ratings and counts by book/category/author | `GROUP BY`, `HAVING`, `FILTER` |
| `q05` | `reports.sql` | Popular books in a time window | CTEs with pre-aggregation to avoid join fan-out |
| `q06` | `reports.sql` | Recommendations from highly rated/finished categories | CTE, `NOT EXISTS`, weighted ranking |
| `q07` | `reports.sql` | User reading statistics | CTE, `CASE`, `SUM`, average rating |
| `q08` | `reports.sql` | Author/category rankings | Window function `RANK() OVER` |
| `q09` | `reports.sql` | Review activity per day | `date_trunc`, `generate_series`, `FULL OUTER JOIN` |
| `q10` | `reports.sql` | Rating distribution per book | `GROUP BY`, `FILTER`, `CASE` |

Search query outline:

```sql
WITH matched AS (
    SELECT b.book_id,
           ts_rank(b.search_vector, websearch_to_tsquery('english', $1)) AS rank
    FROM books b
    WHERE b.search_vector @@ websearch_to_tsquery('english', $1)
       OR b.title ILIKE '%' || $1 || '%'
       OR EXISTS (
            SELECT 1
            FROM book_authors ba
            JOIN authors a ON a.author_id = ba.author_id
            WHERE ba.book_id = b.book_id
              AND a.name ILIKE '%' || $1 || '%'
       )
       OR EXISTS (
            SELECT 1
            FROM book_categories bc
            JOIN categories c ON c.category_id = bc.category_id
            WHERE bc.book_id = b.book_id
              AND c.name ILIKE '%' || $1 || '%'
       )
)
SELECT ...
FROM matched m
JOIN books b ON b.book_id = m.book_id
...
ORDER BY m.rank DESC, b.title;
```

Use `COUNT(*) OVER ()` or a separate count query for `X-Total-Count`.
Pre-aggregate authors and categories with `json_agg` so a page of books does not
produce a row per author/category.

### 5.4 sqlc configuration

`backend/sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "../database/schema/schema.sql"
    queries: "../database/queries"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_interface: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "pg_catalog.numeric"
            go_type: "float64"
          - db_type: "json"
            go_type: "encoding/json.RawMessage"
          - db_type: "jsonb"
            go_type: "encoding/json.RawMessage"
```

`database/schema/schema.sql` is a reduced table/column mirror for sqlc. The
migrations remain canonical; the mirror omits extensions, triggers, functions,
and backfills so sqlc does not have to parse PL/pgSQL.

Tasks:

1. Put every query in `database/queries/*.sql` with `-- name: QueryName :one`
   or `:many` annotations.
2. Run the pinned `sqlc/sqlc:1.27.0` Docker tool service and commit
   `backend/internal/db/`. A local sqlc install is optional. From
   `Team_Project/`:

   ```bash
   docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc
   ```
3. Add generated-code drift verification to CI or `scripts/check.sh`.
4. Keep `database/schema/schema.sql` in sync with migration table/column
   changes; it is the only schema file sqlc reads.

### 5.5 Seed data

The current `001_smoke_test.sql` is a connectivity test, not a demonstration
dataset. Extend the seed set:

| File | Purpose | Contents |
| --- | --- | --- |
| `001_smoke_test.sql` | Proves the migration and seed pipeline works end to end before the full dataset runs | Keep, but change the smoke shelf to `custom` so it does not conflict with the new one-per-status-type index |
| `002_reference_data.sql` | Gives searches and reports enough catalog volume to be meaningful | 50 authors, 15 categories, 200 books, junction rows |
| `003_demo_activity.sql` | Produces realistic shelves, ratings, reviews, progress, and history for the dashboard and report demos | 50 users, default shelves, custom shelves, 500 shelf items, 300 ratings, 300 reviews, progress and history |
| `004_refresh_summaries.sql` | Rebuilds cached rating/review summaries after the bulk inserts, avoiding stale or zero summary values | `SELECT fn_refresh_book_rating_summary(book_id) FROM books;` |

Seed requirements:

- Deterministic UUIDs. Use a helper expression such as
  `('00000000-0000-4000-8000-' || lpad(n::text, 12, '0'))::uuid` so reruns are
  idempotent.
- `ON CONFLICT DO NOTHING` on every insert.
- Demo password hashes. PostgreSQL cannot compute Argon2id in SQL, so
  `003_demo_activity.sql` uses `pgcrypto` bcrypt for the shared demo password
  (`Password123!`). The auth service must accept bcrypt for seeded demo accounts
  and rehash to Argon2id on first login. Accounts created through the API always
  use Argon2id directly.
- Fixed timestamps for report reproducibility.
- A documented demo login, for example `reader1@example.com / Password123!`.
- Keep the dataset volume at the course evidence floor: at least 200 books,
  50 authors, 15 categories, 50 users, 500 shelf entries, and 300 combined
  ratings/reviews.

Seed books keep `source = 'seed'` and synthetic `seed-book-<n>` IDs. They stay
because reports, rankings, recommendations, and the concurrency and
performance evidence need a fixed, reproducible activity dataset; real Google
books get activity only from real users.

No seed file may contain data fetched from Google Books. The repository is
public, and committing Google content would redistribute it. There is no bulk
import command: Google content enters a database only through
`POST /api/books/import`, one volume at a time, when a user acts on it.

---

## 6. Backend Implementation Detail

### 6.1 Configuration

`internal/config/config.go` loads and validates:

| Variable | Default | Rules |
| --- | --- | --- |
| `APP_ENV` | `local` | `local`, `test`, `production` |
| `PORT` | `8080` | 1-65535 |
| `DATABASE_URL` | — | wins over individual DB values |
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE` | existing defaults | password required outside local test |
| `DB_MAX_OPEN_CONNS` | `10` | 1-50 |
| `DB_MAX_IDLE_CONNS` | `5` | <= max open |
| `JWT_SECRET` | — | at least 32 bytes; process exits if missing outside tests |
| `JWT_TTL` | `60m` | 5m-24h |
| `COOKIE_SECURE` | `false` local, `true` production | boolean |
| `ALLOWED_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | comma-separated exact origins |
| `GOOGLE_BOOKS_API_KEY` | empty | server-side only; empty uses Google's small anonymous quota and logs a startup warning; required when `APP_ENV=production` |
| `GOOGLE_BOOKS_BASE_URL` | `https://www.googleapis.com/books/v1` | must be `https` outside tests; tests point it at an `httptest` server |
| `GOOGLE_BOOKS_COUNTRY` | `US` | ISO 3166-1 alpha-2; sent as `country=` to avoid `403 unknownLocation` on cloud hosts |
| `GOOGLE_BOOKS_TIMEOUT` | `5s` | 1s-15s; per Google request |
| `GOOGLE_BOOKS_CACHE_TTL` | `10m` | 0 disables; search pages only |
| `DB_SECRET_ARN` | empty | optional AWS Secrets Manager JSON secret |
| `LOG_LEVEL` | `info` | debug/info/warn/error |

Replace `database/sql` with:

```go
poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
poolConfig.MaxConns = int32(cfg.DBMaxOpen)
poolConfig.MinConns = int32(cfg.DBMaxIdle)
pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
```

Keep pool settings conservative for a small RDS instance. Use a 5-second
per-request database timeout and a 3-second pool acquisition timeout.

### 6.2 Router and middleware order

```text
RequestID
  -> RealIP
  -> Recoverer
  -> structured slog logger
  -> CORS
  -> 10-second request timeout
  -> 1 MiB body limit
  -> rate limiter
  -> route group authentication
  -> handler
```

- CORS must echo only allowed origins, allow `Authorization` and
  `Content-Type`, allow credentials, and expose `X-Total-Count`, `X-Page`, and
  `X-Page-Size`.
- The rate limiter is an in-memory token bucket keyed by client IP. Use 10/min
  for login/register, 30/min for `GET /api/external/books` and
  `POST /api/books/import` (they spend the shared Google quota), and 120/min
  for general API traffic. Document that a multi-instance deployment needs
  Redis later.
- The logger records request ID, method, route template, status, duration, and
  user ID. Never log passwords, tokens, cookies, or request bodies.

### 6.3 Auth implementation

`internal/auth/password.go`:

- Argon2id parameters: memory 64 MiB, iterations 3, parallelism 2, salt 16
  bytes, key 32 bytes.
- PHC-encoded storage:
  `$argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>`.
- Constant-time comparison after parsing.
- Password length 8-128 bytes; reject control characters.
- Accept a bcrypt hash (`$2a$`/`$2b$`) only for seeded demo accounts, then
  rehash to Argon2id on the first successful login. Accounts created through
  the API always start as Argon2id.

`internal/auth/jwt.go`:

- Issue HS256 tokens with `sub`, `email`, `iss`, `iat`, `exp`.
- Parse with explicit method validation and leeway of 30 seconds.
- Return typed errors for expired/malformed tokens.

Registration flow:

1. Normalize email: `strings.ToLower(strings.TrimSpace(email))` to satisfy
   `users_email_normalized`.
2. Validate email format, password, and display name.
3. Hash the password.
4. `INSERT INTO users ...` inside a transaction.
5. The `users_create_defaults` trigger creates the profile and four shelves.
6. Return the user and a fresh token; set the cookie.

Login flow:

1. Look up by exact normalized email.
2. Always run a dummy Argon2 verification when the user is missing to reduce
   timing differences.
3. Return `UNAUTHENTICATED` with a generic message on failure.
4. Issue token and set cookie.

Password change verifies the current password, hashes the new one, and updates
`users.password_hash`. Deleting a user cascades to all owned data; require a
JSON body `{"confirm": true}` for `DELETE /api/users/me`. Because
`reading_history` is append-only, the delete transaction must first run
`SET LOCAL app.allow_history_delete = 'on';` so the user cascade can remove its
history rows. Direct updates and deletes without that setting are rejected.

### 6.4 Catalog services

Library search (`GET /api/books`, `q01`):

- Trim `q`; empty query returns the default sort (newest) with pagination.
- Cap query length at 200 characters.
- Use `websearch_to_tsquery` so quoted phrases and `-` exclusions work.
- Combine `ts_rank`, title prefix, and trigram similarity in the ordering.
- For author/category filters, use `EXISTS` rather than joining in the outer
  query to avoid duplicate rows.
- Return `author` as the alphabetically first author for compatibility and the
  complete `authors` array.

Book detail:

- Use a single aggregate query with `json_agg` for authors/categories.
- Fetch the summary with a left join.
- When the caller is authenticated, fetch `myProgress` and `myRating` in the
  same query or in two small follow-up queries.
- Return `404` for unknown IDs.

Book create/update:

- `POST /api/books` creates `source = 'user'` rows only and does not accept a
  Google volume ID (`CreateBook`).
- Validate title, page count, ISBN-13 format, and published date.
- Upsert authors by case-insensitive normalized name, then upsert categories,
  then replace the book's junction rows in one transaction.
- Reject a duplicate ISBN-13 with `CONFLICT`.
- `PATCH` on a `source = 'google'` book returns `403 FORBIDDEN`; `UpdateBook`
  also filters `source <> 'google'` as the database-side guard.

Google Books search (`GET /api/external/books`):

- Require at least one of `q`, `title`, `author`, `category`, `isbn`; trim
  each and cap at 200 characters.
- Build Google's `q` from the free text plus `intitle:`, `inauthor:`,
  `subject:`, and `isbn:` terms. Quote multi-word operator values.
- Call `GET {base}/volumes` with `q`, `startIndex = (page - 1) * pageSize`,
  `maxResults = pageSize` (max 40), `orderBy`, `printType=books`, `country`,
  `projection=lite` where it carries enough fields, and `key` when set.
- Normalize each volume (`normalize.go`):
  - `title` required; skip results without one.
  - `authors` and `categories` default to empty arrays. Category paths like
    `Fiction / Fantasy / Epic` keep the last segment, trimmed and
    de-duplicated case-insensitively.
  - `description`: strip HTML tags, decode entities, collapse whitespace.
  - `isbn13` from `industryIdentifiers` type `ISBN_13`; else convert
    `ISBN_10` to ISBN-13 (`978` prefix + new check digit); else null.
  - `pageCount <= 0` becomes null.
  - `coverUrl`: `imageLinks.thumbnail` (or `smallThumbnail`), with `http://`
    rewritten to `https://`.
  - `infoLink`: `https://books.google.com/books?id=<volumeId>`.
  - Ignore Google's `averageRating`, `ratingsCount`, prices, and access info.
- Run `ListBooksByExternalIDs` once per page with the page's volume IDs and
  ISBN-13 values and fill `bookId` on matches.
- Cache the normalized page (before the `bookId` lookup) by the normalized
  query string for `GOOGLE_BOOKS_CACHE_TTL`. The `bookId` lookup always runs
  so a fresh import shows up immediately.
- Map Google errors as listed in `docs/api/errors.md`. Never forward Google's
  body, and never log the key or the full request URL with the key.

Google Books import (`POST /api/books/import`):

1. Validate `googleBooksId` (`^[A-Za-z0-9_-]{1,64}$`).
2. `GetBookByExternalID(google_books_id)`. If found, return it with `200`.
3. Fetch `GET {base}/volumes/{id}` from Google. The client never supplies
   metadata. `404` from Google becomes `NOT_FOUND`.
4. Normalize as above. `publishedDate` `YYYY` -> `YYYY-01-01` + `year`,
   `YYYY-MM` -> `YYYY-MM-01` + `month`, `YYYY-MM-DD` -> `day`; anything else
   -> null date and null precision. Reject a volume without a title with
   `VALIDATION_ERROR`.
5. If the ISBN-13 matches an existing row (`GetBookByExternalID(isbn13)`),
   return that row with `200`; do not overwrite seed or user metadata.
6. In one transaction: `ImportGoogleBook`, then `UpsertAuthor` /
   `UpsertCategory` and `AddBookAuthor` / `AddBookCategory` for each name.
   Commit and return the `Book` with `201`.
7. On `23505` `books_isbn13_unique` (a concurrent import of a different volume
   for the same book), roll back, rerun step 5, and return `200`.

The service never copies Google's ratings into `ratings` or
`book_rating_summaries`; the summary trigger creates an empty summary row for
the new book.

### 6.5 Shelf and progress services

- `POST /api/shelves` accepts only `custom` for user-created shelves. Default
  status shelves are created by registration.
- Validate shelf name 1-80 characters; limit each user to 50 custom shelves.
- `DELETE` rejects status shelves with `FORBIDDEN`.
- Every add/move calls `fn_move_book_to_shelf` inside a transaction.
- Every progress update calls `fn_update_reading_progress` inside a transaction.
- Both functions return the resulting progress row; map it to the API DTO.
- Validate ownership before calling any function. A resource owned by another
  user returns `404`, not `403`, to avoid leaking existence.

### 6.6 Review and rating services

- Rating upsert uses `INSERT ... ON CONFLICT (user_id, book_id) DO UPDATE`.
- Review create relies on the unique constraint; map `23505` to `CONFLICT` with
  a clear message.
- `PUT /api/books/{bookId}/review` performs rating upsert + review upsert +
  summary read in one transaction. This is the primary combined transaction
  demonstration.
- Review edits and deletes check `user_id = $authenticatedUser` in the SQL
  `WHERE` clause.
- A review response includes the reviewer's current rating via a left join on
  `ratings`.
- Summary triggers fire inside the same transaction, so a rolled-back review
  cannot leave a changed summary.

### 6.7 Report services

Implement queries `q04`-`q10` as read-only repository calls. Report handlers
must:

- Validate `limit` (1-100), `days` (1-365), and `minRatings` (0-1000).
- Return `[]` rather than `null` for empty results.
- Include the SQL name in a response header or a `queryId` field so the report
  page can link to `docs/database/query-catalog.md`.

`GET /api/reports/queries` returns a static, embedded catalog:

```json
[
  {
    "id": "q05",
    "name": "Popular books",
    "endpoint": "/api/reports/popular",
    "sql": "WITH ..."
  }
]
```

Store the SQL text in Go constants generated from files under
`database/queries/` at build time or copy it into a read-only reports package.
Never accept user SQL.

### 6.8 Health and readiness

`/healthz` returns process status only.

`/readyz`:

1. Ping the pool with a 3-second timeout.
2. Query the migration version table (`schema_migrations`).
3. Return `200` only when the pool connects and the version is at least the
   latest expected migration (currently `6`); otherwise `503` with a reason.

`/readyz` does not call Google. A Google outage disables discovery only; the
rest of the API stays ready.

---

## 7. Transactions, Locking, and Concurrency

### 7.1 Required transaction scenarios

| Scenario | Steps | Evidence |
| --- | --- | --- |
| Review + rating | Upsert rating, upsert review, verify summary, commit; repeat with a forced invalid step and roll back | `docs/evidence/transactions.md` |
| Shelf move | Lock shelf, remove other status memberships, upsert item, update progress, insert history | same |
| Progress completion | Validate page, update progress, move to Finished, insert completion event | same |

All three use `pgx.Tx` with `defer tx.Rollback(ctx)` and an explicit commit.
Do not hide transactions inside repositories.

### 7.2 Concurrency scenarios

| Scenario | Mechanism | Expected result |
| --- | --- | --- |
| Two users rate the same book simultaneously | Summary upsert row lock | Both ratings persist; count = 2; average correct |
| Duplicate review race for one user/book | Unique constraint | One `201`; one `409` |
| Progress update vs. completion | `FOR UPDATE` + optional version check | Serialized updates; stale version gets `409` |
| Shelf move race | `FOR UPDATE` on shelf/progress | Final state matches the last committed move; no orphan items |
| Deadlock/lock timeout | Retry helper | At most one retry; then `409`/`503` with request ID |
| Two users import the same Google volume | `ON CONFLICT (google_books_id)` in `ImportGoogleBook` | One `books` row; both callers get the same `id` |

Use the default `READ COMMITTED` isolation level and document it. Implement a
small retry helper for SQLSTATE `40001` and `40P01` with a maximum of two
attempts. Never retry non-idempotent requests without a request ID or version
guard.

---

## 8. Security and Operational Rules

- Never return `password_hash` or internal SQL errors to clients.
- Validate and bound every input; reject unknown JSON fields in decoders.
- Use parameterized SQL only. `sqlc` enforces this for generated queries; the
  hand-written function calls use `$1`-style arguments.
- Set `Cache-Control: no-store` on auth and personal data responses.
- CORS: exact origins, credentials enabled, no `*` with credentials.
- Rate-limit authentication endpoints; log failures without logging credentials.
- Cookie `Secure` in production; `SameSite=Lax`; no token in URLs.
- Graceful shutdown: drain HTTP, close the pool, log the reason.
- Request IDs propagate to logs and error responses.
- Limit database privileges in production to the application schema (no
  superuser); RDS credentials come from Secrets Manager.
- Google Books:
  - The API key lives only on the server (env var locally, Secrets Manager in
    AWS). The browser never calls Google directly.
  - Restrict the key in Google Cloud Console to the Books API.
  - Treat every Google field as untrusted input: strip HTML from
    descriptions before storing or returning them, accept only `https` cover
    URLs, and let React escape all text. Never render Google HTML.
  - Import only by volume ID, re-fetched from Google on the server.
  - Show Google attribution ("Powered by Google") and the `infoLink` on every
    search result and on the detail page of `source = google` books. Pages
    that list stored books (shelves, progress, reports) show one page-level
    "Book information from Google Books" notice when any listed book has
    `source = google`.
  - Store only books a user acted on, and never commit Google data to the
    repository. Google's API terms restrict permanent copies of API content;
    whether storing acted-on books is acceptable for this course project is an
    open decision (Section 16).

---

## 9. Testing Plan

### 9.1 Go tests

| Level | Location | Coverage |
| --- | --- | --- |
| Unit | `backend/internal/auth`, `backend/internal/httpx` | Argon2 hash/verify, JWT issue/parse/expiry, validation helpers, error mapping |
| Handler | `backend/internal/handlers/*_test.go` | `httptest` status codes, JSON shapes, auth requirements, validation errors |
| Repository | `backend/internal/repositories/*_test.go` | Row mapping, pagination, search filters |
| Integration | `backend/internal/integration` | Real PostgreSQL via Testcontainers, migrations applied, full endpoint flows |
| Concurrency | `backend/internal/integration/concurrency_test.go` | The scenarios in Section 7.2, including concurrent import |
| Google client | `backend/internal/googlebooks/*_test.go` | Normalization table tests (partial dates, HTML, ISBN-10 conversion, BISAC paths, missing authors, `http` covers, `pageCount` 0); error mapping against an `httptest` fake Google. Tests never call the real Google API |

Setup:

- Use `testcontainers-go` with `postgres:18-alpine`.
- Apply migrations with `golang-migrate` or by executing the migration files.
- Seed deterministic fixtures per test; truncate between tests.
- Skip integration tests with a build tag (`//go:build integration`) when Docker
  is unavailable, but require them in CI.

### 9.2 Database validation cases (20+)

`database/tests/validation_cases.sql` and `docs/evidence/validation.md`:

1. Duplicate email rejected.
2. Email with uppercase/whitespace rejected by the normalized check.
3. Blank display name rejected.
4. Rating 0 rejected.
5. Rating 6 rejected.
6. Rating 5 accepted.
7. Duplicate rating for one user/book rejected.
8. Duplicate review for one user/book rejected.
9. Blank review text rejected.
10. Progress page greater than page count rejected.
11. Negative current page rejected.
12. Percentage outside 0-100 rejected.
13. Invalid reading status rejected.
14. Invalid shelf type rejected.
15. Duplicate status shelf for one user rejected.
16. Duplicate custom shelf name for one user rejected.
17. Foreign-key violation rejected.
18. Review delete updates `review_count`.
19. Rating insert/update/delete updates summary correctly.
20. `reading_history` update and delete are rejected.
21. Moving to a status shelf removes other status memberships.
22. Default shelves are created on registration.
23. Book with two authors and two categories returns both.
24. Cascade delete removes owned shelves, progress, ratings, and reviews.
25. A `google` book without `imported_at` or `google_books_id` is rejected.
26. An unknown `books.source` is rejected.
27. A date without precision, or precision without a date, is rejected.
28. Importing the same volume twice returns the same `book_id`; a second
    volume with the same ISBN-13 raises `books_isbn13_unique`.
29. `UpdateBook` does not change a `google` book.

### 9.3 Frontend end-to-end

Playwright flows:

1. Register, log in, and load the dashboard.
2. Search Google Books (against a stubbed API response in CI), import a
   result, and shelve it; search the library by title, author, and category.
3. Open a book detail page and see authors/categories/summary.
4. Add a book to a shelf and move it between status shelves.
5. Update progress and confirm the Finished transition.
6. Create, edit, and delete a review.
7. Rate a book and see the summary update.
8. Open each report and verify non-empty data.

---

## 10. Frontend Wiring Required to Complete the Project

The UI is built, but it is not connected. These changes are part of the
backend completion milestone because the API cannot be considered done until
the application uses it.

| File / area | Change |
| --- | --- |
| `frontend/src/lib/api-client.ts` | Add `credentials: "include"`, parse the error envelope, surface `requestId`, handle `401` by invalidating auth state, expose pagination headers |
| `frontend/src/providers/QueryClientProvider.tsx` | New TanStack Query provider with sensible retry/stale defaults |
| `frontend/src/app/layout.tsx` | Wrap the application in the query provider |
| `frontend/src/services/auth-service.ts` | `register`, `login`, `logout`, `me` |
| `frontend/src/services/user-service.ts` | Profile read/update, password change |
| `frontend/src/services/book-service.ts` | Add `searchGoogle` (`GET /api/external/books`), `importGoogleBook`, and an `ensureLocalBook(result)` helper that returns `result.bookId` or imports first; keep `search` for library search; detail, create/update, rating summary |
| `frontend/src/components/GoogleAttribution.tsx` | New: "Powered by Google" mark and link, used on search results, Google-sourced book detail, and as a page-level notice |
| `frontend/next.config.*` | Allow `books.google.com` and `*.googleusercontent.com` covers if `next/image` is used |
| `frontend/src/services/reading-service.ts` | Progress CRUD, history, shelf add/move/remove |
| `frontend/src/services/shelf-service.ts` | Shelf CRUD |
| `frontend/src/services/review-service.ts` | Review list/create/update/delete, combined review+rating |
| `frontend/src/services/author-service.ts` | Authors and categories |
| `frontend/src/services/report-service.ts` | All report endpoints |
| `frontend/src/types/domain.ts` | Add summary fields, author/category IDs, version, user/profile shapes, `Book.source`/`infoLink`/`publishedDatePrecision`, and an `ExternalBook` type |
| `frontend/src/components/AuthBoundary.tsx` | Replace `localStorage` with `useQuery(['auth','me'])`; redirect to `/login` on `401` |
| `frontend/src/app/login/page.tsx` | React Hook Form + Zod; call the auth service; show API errors |
| `frontend/src/app/register/page.tsx` | New registration page |
| `frontend/src/app/(workspace)/dashboard/page.tsx` | Real stats, current reading, recent activity, shelves |
| `frontend/src/app/(workspace)/catalog/page.tsx` | Two tabs: "Google Books" (default, `searchGoogle`) and "In the library" (`search`, with author/category filters); pagination, loading/empty/error states; on `UPSTREAM_*` errors show a message and switch to the library tab |
| `frontend/src/app/books/[bookId]/page.tsx` | New book detail page with shelf/progress/review/rating actions; Google attribution and `infoLink` when `source = google`; date shown per `publishedDatePrecision` |
| `frontend/src/app/(workspace)/authors/page.tsx` | Author/category listings and detail navigation |
| `frontend/src/app/(workspace)/shelves/page.tsx` | Shelf list, create/rename/delete, add/move/remove books |
| `frontend/src/app/(workspace)/progress/page.tsx` | Progress list and update form |
| `frontend/src/app/(workspace)/reviews/page.tsx` | User reviews with edit/delete |
| `frontend/src/app/(workspace)/reports/page.tsx` | Report tabs and query links |
| `frontend/src/app/(workspace)/settings/page.tsx` | Profile and password forms |
| `frontend/src/app/explore/page.tsx` | Public Google Books search; actions on a result send anonymous users to `/login` before import |
| `frontend/src/components/PublicHome.tsx` | Replace `featuredBooks` mock data with a public featured query |
| `frontend/src/data/mock-data.ts` | Remove usage; delete or retain only as a storybook fixture |

Use React Hook Form + Zod for register, login, review, rating, shelf, progress,
and profile forms. Use TanStack Query mutations with invalidation for shelves,
progress, reviews, ratings, and reports. Add optimistic updates for rating and
shelf actions.

Never call `googleapis.com` from the browser and never put the key in a
`NEXT_PUBLIC_*` variable.

Contract compatibility: keep `Book.author` as a string, keep `Review.body`,
keep `Rating.value`, keep `ReadingProgress.bookId/status/currentPage/totalPages`,
and keep the `ReadingReport` field names in Section 4.4 unchanged.

---

## 11. Docker, CI, and Deployment

### 11.1 Compose

Add to the `api` service in both Compose files:

```dotenv
APP_ENV=local
ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
JWT_TTL=60m
COOKIE_SECURE=false
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
```

`GOOGLE_BOOKS_API_KEY` and `GOOGLE_BOOKS_COUNTRY` are already passed to the
`api` service in both Compose files (empty key and `US` by default).

For the RDS file use `APP_ENV=production` and `COOKIE_SECURE=true` when the
frontend is served over HTTPS. Never commit a production `JWT_SECRET`.

Add a `web` service after `frontend/Dockerfile` exists, with a health check and
`NEXT_PUBLIC_API_URL` pointing at the API.

### 11.2 CI

`.github/workflows/ci.yml`:

1. **Backend static:** `go vet ./...`, `go test ./...`, `sqlc diff` or
   regenerate-and-compare.
2. **Database integration:** start `postgres:18`, run `migrate up`, apply seeds,
   run `go test -tags=integration ./...`, run the validation SQL suite.
3. **Frontend:** `npm ci`, `npm run lint`, `npm run typecheck`,
   `npm run build`.
4. **Infra:** `terraform fmt -check`, `terraform validate`.

### 11.3 AWS

- RDS remains private; only the API security group connects.
- Load DB credentials from Secrets Manager via `DB_SECRET_ARN` when configured;
  otherwise use injected environment variables for the course deployment.
- Add the API container to ECS Fargate (or the existing EC2 profile), behind an
  ALB, with CloudWatch JSON logs.
- Store the JWT secret in Secrets Manager, not in the task definition.
- Verify a backup/restore into a separate database before the final demo.

---

## 12. Documentation and Evidence Package

Create:

```text
docs/api/
  openapi.yaml          # Authoritative request/response contract for every endpoint
  errors.md             # Error codes, envelope format, and SQLSTATE-to-HTTP mapping
  examples.md           # curl examples for each endpoint group
docs/database/
  schema-dictionary.md  # Table/column meanings, constraints, ownership, and cascade rules
  query-catalog.md      # q01-q10 with purpose, SQL, parameters, and expected output
docs/diagrams/          # Other reviewed diagrams, if any (the ER diagram is at the top level, below)
docs/evidence/
  transactions.md       # Commit and deliberate-rollback output for the three transactions
  concurrency.md        # Multi-session output, locks, isolation levels, and observed results
  validation.md         # 20+ validation cases with setup, expected result, and pass/fail
  performance.md        # Before/after EXPLAIN plans, rows examined, and timings
docs/runbooks/
  local.md              # Local Docker startup, reset, and troubleshooting
  rds.md                # RDS connectivity, migrations, secrets, and backup checks
  reset.md              # Destructive reset procedure with confirmation and target check
```

Also add the course-mandated evidence directories:

```text
database/
  queries/        # q01-q10, compiled by sqlc
  functions/      # documented copies of fn_* DDL
  triggers/       # documented copies of trigger DDL
  tests/          # validation cases
  concurrency/    # multi-session scripts
  performance/    # baseline and optimized EXPLAIN scripts
```

Canonical function/trigger DDL lives in the migrations. The `functions/` and
`triggers/` files are evidence copies and must be checked against the migration
text; note this in their README files.

The ER diagram is `ER_diagram.png` at the top level of `Team_Project/`, with its
reading guide `ER_diagram.md` (entity and relationship tables, a Mermaid
relational equivalent, and a check against the migrations). It already includes
the `000006` columns on `BOOKS`. The old `finalized_er_diagram.png` is
superseded and removed. Reference both files from the root README, and keep
them in step with every migration that changes a table or column; update
`database/schema/schema.sql` at the same time.

Known limitation recorded in `ER_diagram.md` section 8: authors are identified
by name (case-insensitive unique), so two people with the same name share one
row.

Update stale statements:

- Root README: backend is no longer "only a README".
- `PLAN.md`: reflect the current migration, schema, and API work.
- Milestone plan: replace MySQL-specific wording with PostgreSQL equivalents.
- Frontend README: state that the UI is complete but API wiring is in progress.

---

## 13. Phased Execution Checklist

### Phase 1 — Contract and database foundation

- [x] Freeze the API contract in `docs/api/openapi.yaml`.
- [x] Add `docs/api/errors.md` and `docs/api/examples.md`.
- [x] Add `000002_search_and_extensions` up/down migrations.
- [x] Add `000003_domain_rules` up/down migrations.
- [x] Add `000004_reporting_indexes` up/down migrations.
- [x] Write `database/queries/*.sql` for all endpoints.
- [x] Add `database/schema/schema.sql` and `backend/sqlc.yaml`.
- [x] Add `002_reference_data.sql`, `003_demo_activity.sql`, and summary refresh.
- [x] Fix `001_smoke_test.sql` so it does not conflict with default shelves.
- [x] Generate `backend/internal/db` with the Docker sqlc tool service.
- [x] Verify a clean database migrates and seeds without manual edits.

**Status (2026-11-10):** Phase 1 verified. sqlc generated 13 Go files;
`go mod tidy` and `go build ./...` pass in WSL. A clean PostgreSQL 18 database
applied migrations `000001`-`000004` and all seeds. OpenAPI validated with
38 paths, 53 operations, 225 refs, no missing refs, and no duplicate
operationIds. The `.down.sql` files have not been exercised yet.

**Phase 1 addendum (2026-11-10): Google Books design.**

- [x] Add `000006_book_sources` up/down (source, date precision, import time).
- [x] Update the schema mirror, seeds 001/002 (`source = 'seed'`), and book
      queries; add `ListBooksByExternalIDs` and `ImportGoogleBook` (66 queries).
- [x] Add `GET /api/external/books`, `POST /api/books/import`,
      `ExternalBook`, `Book.source`/`infoLink`/`publishedDatePrecision`, and
      the `UPSTREAM_*` errors to the contract.

Verified in a throwaway `postgres:18-alpine`: migrations 1-6 up, seeds twice,
all 66 generated queries `PREPARE`, `000006` down/up on seeded data (backfill
gives 201 `seed` rows with `day` precision), and 10 constraint/import checks.
`sqlc diff`, `go build ./...`, and `go vet ./...` are clean. OpenAPI: 40
paths, 55 operations, 247 refs, no missing refs, no duplicate operationIds.

### 13.1 Phase 1 verification commands

```bash
# From Team_Project/
cp -n .env.example .env
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc

cd backend
go mod tidy
cd ..

./scripts/dev.sh local
```

Then confirm:

```bash
docker compose --profile tools exec postgres \
  psql -U book_app -d book_metaverse -c \
  "SELECT (SELECT count(*) FROM books) AS books,
          (SELECT count(*) FROM authors) AS authors,
          (SELECT count(*) FROM categories) AS categories,
          (SELECT count(*) FROM users) AS users,
          (SELECT count(*) FROM shelves) AS shelves,
          (SELECT count(*) FROM ratings) AS ratings,
          (SELECT count(*) FROM reviews) AS reviews;"
```

Expected: 201 books, 51 authors, 16 categories, 51 users (50 demo + 1 smoke),
301 ratings, 301 reviews, and a non-zero shelf count.

**Exit:** `migrate up` + seed on an empty PostgreSQL 18 database produces a
complete dataset; `sqlc generate` succeeds.

### Phase 2 — Go foundation

- [ ] Replace `database/sql` with `pgxpool`.
- [ ] Add config loading and validation, including the `GOOGLE_BOOKS_*`
      settings (Section 6.1). The Google client itself is Phase 4.
- [ ] Add `httpx` response/error helpers and domain errors.
- [ ] Add request ID, logging, recovery, CORS, timeout, body limit, and rate
      limit middleware.
- [ ] Add `/healthz`, `/readyz`, and `/` with migration-version readiness.
- [ ] Add route skeleton with 501 or 404 for unimplemented endpoints during
      development.

**Exit:** The API starts with a valid pool, readiness reports the schema
version, and CORS works from `http://localhost:3000`.

### Phase 3 — Authentication and users

- [ ] Argon2id hash/verify with PHC encoding.
- [ ] JWT issue/parse with method and expiry validation.
- [ ] Register, login, logout, and me endpoints with cookie + Bearer support.
- [ ] Profile read/update and password change.
- [ ] Authorization middleware and ownership helpers.
- [ ] Unit tests for hashing, JWT, and auth handlers.

**Exit:** Register creates a user, profile, and four shelves; login returns a
working token; protected routes reject anonymous callers.

### Phase 4 — Catalog and search

- [ ] Library search with filters, sort, pagination, and total count.
- [ ] Book detail with authors, categories, summary, user context, `source`,
      and `infoLink`.
- [ ] Book create (`source = user`) and update (`403` for Google rows) with
      author/category upserts.
- [ ] Authors and categories endpoints.
- [ ] `internal/googlebooks` client, normalizer, and cache with table tests
      against an `httptest` fake.
- [ ] `GET /api/external/books` with `bookId` marking and error mapping.
- [ ] `POST /api/books/import` with dedupe by volume ID and ISBN-13, and the
      concurrent-import test.
- [ ] Search integration tests for title, author, category, and keyword.

**Exit:** `q01` and `q02` return validated results; a book with multiple
authors/categories renders completely; a Google result can be imported once
and then shelved through the existing shelf endpoint; Google failures return
`UPSTREAM_*` while library search keeps working.

### Phase 5 — Shelves, progress, and history

- [ ] Shelf list/create/rename/delete.
- [ ] Shelf books list and add/move/remove.
- [ ] `fn_move_book_to_shelf` and `fn_update_reading_progress`.
- [ ] Progress list/get/update/delete and history list.
- [ ] Optimistic version handling and retry helper.
- [ ] Transaction commit and rollback tests.

**Exit:** Moving a book always updates shelf state, progress, and history
consistently; invalid page counts are rejected.

### Phase 6 — Reviews, ratings, and summaries

- [ ] Rating summary endpoint and rating upsert/delete.
- [ ] Review list/create/update/delete with ownership checks.
- [ ] Combined `PUT /api/books/{bookId}/review` transaction.
- [ ] Summary trigger verification for insert/update/delete.
- [ ] Duplicate and invalid-input conflict mapping.
- [ ] Review/rating integration and concurrency tests.

**Exit:** Duplicate reviews and invalid ratings are rejected; summary counts stay
consistent after concurrent writes.

### Phase 7 — Reports

- [ ] `q03`-`q10` repository queries.
- [ ] Reading-history report matching the frontend contract.
- [ ] Popular, top-rated, recommendation, ranking, and activity endpoints.
- [ ] Static report query catalog endpoint.
- [ ] Report tests with deterministic seed data.

**Exit:** Every report returns non-empty, validated results against the seed
dataset and links to its SQL.

### Phase 8 — Frontend integration

- [ ] Query provider and auth service.
- [ ] Replace `localStorage` auth with cookie-backed session.
- [ ] Wire catalog, book detail, shelves, progress, reviews, ratings, reports,
      and settings.
- [ ] Google Books search tab, import-on-action, and Google attribution.
- [ ] Add forms and loading/empty/error states.
- [ ] Remove `mock-data.ts` usage.
- [ ] Playwright happy-path flows.

**Exit:** No workspace page depends on mock data or a fake session.

### Phase 9 — Evidence and quality gates

- [ ] 20+ validation cases with recorded pass/fail.
- [ ] Three transaction demonstrations with commit and rollback.
- [ ] Three concurrency demonstrations with multi-session output.
- [ ] Five index optimizations with before/after plans and timings.
- [ ] `go test ./...`, integration tests, frontend lint/typecheck/build,
      `terraform validate`.
- [ ] Evidence files linked from `docs/README.md`.

**Exit:** Every required database behavior is reproducible from scripts and
documented with actual output.

### Phase 10 — Deployment and final QA

- [ ] Apply migrations and seeds to RDS.
- [ ] Deploy the API and frontend.
- [ ] Verify Secrets Manager access, private RDS connectivity, and logs.
- [ ] Run the complete flow against RDS.
- [ ] Rehearse the final demonstration and freeze features.

**Exit:** The deployed application performs every core operation against
PostgreSQL, and the local and deployed schemas match.

---

## 14. Requirement Traceability

| Course requirement | Implementation |
| --- | --- |
| Database-driven platform | Go API + explicit SQL + seeded PostgreSQL; no ORM |
| Search by title/author/category/keyword | Discovery: `GET /api/external/books` (Google Books). Library: `q01`, `GET /api/books`, FTS + `pg_trgm` + GIN |
| Book metadata from a real source | `POST /api/books/import`, `books.source`, `ImportGoogleBook` with dedupe on `google_books_id`/`isbn13` |
| Book metadata | `books`, `authors`, `categories`, junction tables, detail endpoint |
| Accounts and profiles | `/api/auth/*`, `/api/users/me`, Argon2id + JWT |
| Reading shelves | `/api/shelves`, `fn_move_book_to_shelf`, default shelves |
| Reviews and ratings | `/api/books/{id}/reviews`, `/api/books/{id}/ratings`, ownership checks |
| Average ratings and summaries | `book_rating_summaries` + refresh triggers |
| Progress and history | `/api/reading-progress`, `/api/reading-history`, `fn_update_reading_progress` |
| Many-to-many authors/categories | `book_authors`, `book_categories`, aggregate detail query |
| Managed relationships | FKs, unique/check constraints, composite keys, partial unique index |
| Triggers/procedures | Rating validation, summary maintenance, defaults, history immutability, progress validation |
| Transactions | Registration, shelf move, progress update, combined review+rating |
| Concurrency control | Row locks, optimistic version, unique constraints, retry helper |
| Advanced SQL | `q04`-`q10`: CTEs, window functions, `CASE`, `EXISTS`, aggregates |
| Interface demonstration | Full frontend integration in Phase 8 |

---

## 15. Definition of Done

The project is complete when:

- [ ] A clean PostgreSQL 18 database migrates and seeds with one command.
- [ ] Registration creates a user, profile, and four default shelves.
- [ ] Login works with the seeded demo account and protects workspace routes.
- [ ] Search by title, author, category, and keyword returns paginated results,
      both from Google Books and from the library.
- [ ] A Google result can be imported once and then shelved, rated, reviewed,
      and tracked; Google content is shown with attribution.
- [ ] Book detail shows all authors, categories, summary, and reviews.
- [ ] Shelf add/move/remove updates progress and history atomically.
- [ ] Progress updates validate page counts and move completed books to Finished.
- [ ] Ratings and reviews can be created, edited, and deleted by their owner.
- [ ] Invalid ratings, duplicate reviews, and duplicate ratings are rejected.
- [ ] Rating and review summaries stay correct after concurrent writes.
- [ ] At least eight key queries return validated results.
- [ ] Five indexes show measurable before/after improvement.
- [ ] Twenty or more validation cases pass reproducibly.
- [ ] Three transaction and three concurrency demonstrations are documented.
- [ ] The frontend uses the API for every workspace page.
- [ ] `go test ./...`, integration tests, lint, typecheck, and build pass.
- [ ] The deployed application uses RDS for every core operation.
- [ ] `docs/api`, `docs/database`, `docs/evidence`, and `docs/runbooks` are
      populated and linked from the README.

---

## 16. Risks and Open Decisions

| Risk / decision | Impact | Mitigation |
| --- | --- | --- |
| MySQL wording remains in older docs | Confusing deliverables and grading | Treat PostgreSQL 18 as final; update the Milestone Plan and presentation |
| `sqlc` cannot parse PL/pgSQL functions | Generation failure | Keep functions in separate DDL files; verify on the first generation pass; fall back to hand-written pgx wrappers |
| Adding author uniqueness fails on existing duplicates | Migration failure | Include the dedupe/remap step in `000003` and test on a copy first |
| Default-shelf trigger conflicts with old smoke seed | Seed failure | Change the smoke shelf to `custom` and use `NOT EXISTS` guards |
| Seed volume takes too long | Schedule slip | Generate deterministic rows with `generate_series`; keep one SQL file per concern |
| Cookie auth breaks across ports/domains | Frontend cannot authenticate | Configure exact `ALLOWED_ORIGINS`, `credentials: include`, and `SameSite=Lax`; use Bearer in tests |
| Testcontainers unavailable on a teammate machine | Integration tests skipped locally | Run them in CI and document the Docker requirement |
| Concurrent summary updates look flaky | Weak evidence | Record isolation level, seed state, session order, and final counts; use deterministic fixtures |
| No admin role | Any authenticated user can add catalog records | Document the MVP limitation; add `users.role` in a post-MVP migration if required |
| Authors are identified by name (`authors_name_ci_unique_idx`) | Two different people with the same name share one author row | Accepted for the MVP: Google returns names without IDs and `UpsertAuthor` depends on the unique name. Documented in `ER_diagram.md` section 8 |
| Google Books unavailable or slow during the demo | Discovery search and import fail | 5 s timeout and `UPSTREAM_*` errors; the UI falls back to library search; reports and evidence use seed data only; import the demo books before the presentation |
| Google quota exhausted (about 1,000 requests/day by default; anonymous is lower) | Search returns `503` | Use an API key, 10-minute search cache, 30/min per-IP limit; tests use a fake server, never the real API |
| Google query operators misbehave (`intitle:`/`inauthor:`/`isbn:` returned 0 results in October 2026 until a reported fix on 2026-11-02) | Filtered Google search returns nothing | Rehearse the demo queries; free-text `q` and library search still work |
| Google API terms restrict permanent copies of content | Storing imported books may not comply | Store only books users act on, attribute them, never commit Google data; confirm with the instructor (open decision 4) |
| Imported books have no seed activity | Reports look empty for real books | Expected; reports demonstrate SQL on the seed dataset, and real books gain activity as users use them |

### Open decisions to confirm with the team

1. Whether catalog writes should require a future `admin` role or remain open
   to any authenticated user for the course demo.
2. Whether the deployed frontend and API share a domain (simplifies cookies) or
   are cross-site (requires a CSRF strategy).
3. Whether the combined `PUT /api/books/{bookId}/review` endpoint should be the
   only review form path or coexist with separate review and rating endpoints.
   This plan keeps both.
4. **Decided 2026-11-10:** users search Google Books live and books are
   imported on first use (Section 2, item 11). Still open: whether the
   instructor accepts storing acted-on Google books under Google's API terms.
   If not, the fallback is to store only `google_books_id` plus the user's own
   data and fetch display metadata from Google at read time, which removes
   library search for those books.
5. Whether to keep the library search tab visible to users or use
   `GET /api/books` only for evidence and filters. This plan keeps the tab.
