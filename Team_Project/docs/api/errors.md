# API Errors

Every non-2xx response uses one envelope:

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

`message` is safe to show to a user. `details` is optional and may contain a
field name or database constraint. `requestId` matches the request log entry
and is also sent in the `X-Request-ID` response header.

## Browser client rules

- The browser calls `/api/...` on the web app's own origin and Next.js forwards
  the request to the Go API, so there is no CORS and every response header
  (including `X-Total-Count`, `X-Page`, `X-Page-Size`, and `X-Request-ID`) is
  readable.
- `204` responses (logout, password change, every `DELETE`) have no body.
- `401 UNAUTHENTICATED` means the session is missing or expired, except on
  `POST /api/auth/login`, where it means wrong credentials.
- A wrong `currentPassword` on `PUT /api/users/me/password` is
  `400 VALIDATION_ERROR` with `details.field = "currentPassword"`, never `401`.
- Login and sign-up are rate-limited per client address: `POST /api/auth/login`
  allows 10 attempts at once and then one every 6 seconds;
  `POST /api/auth/register` allows 5 and then one a minute. Over the limit is
  `429 RATE_LIMITED` with a `Retry-After` header (seconds) and
  `details.retryAfterSeconds`. No other endpoint is limited.

## Codes

| Code | HTTP | Meaning |
| --- | --- | --- |
| `VALIDATION_ERROR` | 400 | Malformed JSON, unknown field, missing field, invalid range or enum |
| `UNAUTHENTICATED` | 401 | Missing, malformed, or expired session |
| `FORBIDDEN` | 403 | Authenticated but the action is not allowed |
| `NOT_FOUND` | 404 | Resource missing or owned by another user |
| `CONFLICT` | 409 | Duplicate resource, duplicate review/rating, duplicate shelf name |
| `STALE_VERSION` | 409 | `reading_progress.version` does not match the current row |
| `RATE_LIMITED` | 429 | Too many login or sign-up attempts from one client; retry after `Retry-After` seconds |
| `UPSTREAM_UNAVAILABLE` | 502 | Open Library failed, timed out, is rate limiting, or returned an unusable response |
| `DATABASE_UNAVAILABLE` | 503 | The pool cannot reach PostgreSQL |
| `INTERNAL` | 500 | Unexpected failure; details are logged, not returned |

## Open Library mapping

Only `GET /api/external/books` and `POST /api/books/import` call Open Library.
Every other endpoint reads PostgreSQL only, so an Open Library outage never
breaks library search, shelves, progress, reviews, or reports. Calls have a
9-second timeout (inside the API's 10-second request limit).

| Open Library response | API response |
| --- | --- |
| `200` | Normalized result; `200` for search, `201` or `200` for import |
| `404`, a deleted work, or a work with no title | `404 NOT_FOUND` ("Book not found in Open Library.") |
| A merged work (`/type/redirect`) | Followed once; the import uses the new work ID |
| `429`, any other non-`200`, timeout, or malformed JSON | `502 UPSTREAM_UNAVAILABLE` |

Invalid parameters never reach Open Library: they are `400 VALIDATION_ERROR`
(for example an `externalId` that is not a work ID such as `OL893414W`).
Open Library's status is logged with the request ID; it is never returned to
the client.

Import race: two users importing the same work at once both get the same
book (one `201`, one `200`). A work with an ISBN-13 that is already in the
library returns the existing book with `200`, including when the other row
appears between the check and the insert (`23505` on `books_isbn13_unique`).

`PATCH /api/books/{bookId}` on an imported book (`source` `openlibrary` or
`google`) returns `403 FORBIDDEN`.

## PostgreSQL mapping

| SQLSTATE | Meaning | API response |
| --- | --- | --- |
| `23505` | Unique violation | `409 CONFLICT`; include the constraint in `details` |
| `23503` | Foreign-key violation | `400 VALIDATION_ERROR` |
| `23514` | Check violation | `400 VALIDATION_ERROR` |
| `22023` | Invalid parameter raised by a trigger | `400 VALIDATION_ERROR` |
| `P0002` | Book or shelf not found inside a function | `404 NOT_FOUND` |
| `BM001` | Stale `reading_progress.version` (project-specific code, migration `000005`) | `409 STALE_VERSION`; never retried |
| `40001` | Serialization failure | Retry once, then `409 CONFLICT` |
| `40P01` | Deadlock detected | Retry once, then `409 CONFLICT` |
| `55000` | Direct `reading_history` update/delete attempted | `409 CONFLICT` |

Deleting a user or a book still removes its history: the append-only trigger
allows deletes that arrive through a foreign-key cascade.

## Trigger messages

| Trigger | Message | Code |
| --- | --- | --- |
| `ratings_validate_value` | `rating_value must be between 1 and 5` | `VALIDATION_ERROR` |
| `reading_progress_validate` | `current_page must be between 0 and <n>` | `VALIDATION_ERROR` |
| `reading_progress_validate` | `status must be want_to_read, reading, completed, or paused` | `VALIDATION_ERROR` |
| `reading_history_immutable` | `reading_history is append-only` | `CONFLICT` |
| `fn_move_book_to_shelf` | `shelf not found` | `NOT_FOUND` |
| `fn_update_reading_progress` | `book not found` | `NOT_FOUND` |
| `fn_update_reading_progress` | `stale reading progress version` | `STALE_VERSION` |

## Examples

Duplicate review:

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "You already reviewed this book.",
    "details": { "constraint": "reviews_user_book_unique" },
    "requestId": "01J9Z8R6K3W4T2M1"
  }
}
```

Invalid page number:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "current_page must be between 0 and 200",
    "requestId": "01J9Z8R6K3W4T2M1"
  }
}
```
