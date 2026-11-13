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

- CORS exposes `X-Total-Count`, `X-Page`, `X-Page-Size`, `X-Request-ID`, and
  `Retry-After`; other response headers are invisible to frontend code.
- `204` responses (logout, password change, every `DELETE`) have no body.
- `401 UNAUTHENTICATED` means the session is missing or expired, except on
  `POST /api/auth/login`, where it means wrong credentials.
- A wrong `currentPassword` on `PUT /api/users/me/password` is
  `400 VALIDATION_ERROR` with `details.field = "currentPassword"`, never `401`.
- `429` and `503` may carry `Retry-After` (seconds).

## Codes

| Code | HTTP | Meaning |
| --- | --- | --- |
| `VALIDATION_ERROR` | 400 | Malformed JSON, unknown field, missing field, invalid range or enum |
| `UNAUTHENTICATED` | 401 | Missing, malformed, or expired session |
| `FORBIDDEN` | 403 | Authenticated but the action is not allowed |
| `NOT_FOUND` | 404 | Resource missing or owned by another user |
| `CONFLICT` | 409 | Duplicate resource, duplicate review/rating, duplicate shelf name |
| `STALE_VERSION` | 409 | `reading_progress.version` does not match the current row |
| `RATE_LIMITED` | 429 | Authentication, general, or Google-search rate limit exceeded for this client |
| `UPSTREAM_UNAVAILABLE` | 502 | Google Books failed, timed out, or returned an unusable response |
| `UPSTREAM_RATE_LIMITED` | 503 | The server's Google Books quota is exhausted; includes `Retry-After` |
| `DATABASE_UNAVAILABLE` | 503 | The pool cannot reach PostgreSQL |
| `INTERNAL` | 500 | Unexpected failure; details are logged, not returned |

## Google Books mapping

Only `GET /api/external/books` and `POST /api/books/import` call Google. Every
other endpoint reads PostgreSQL only, so a Google outage never breaks library
search, shelves, progress, reviews, or reports.

| Google response | API response |
| --- | --- |
| `200` | Normalized result; `200` for search, `201` or `200` for import |
| `404` for a volume ID | `404 NOT_FOUND` with `details.googleBooksId` |
| `400` (bad query) | `400 VALIDATION_ERROR` |
| `403` with reason `unknownLocation` | `502 UPSTREAM_UNAVAILABLE`; set `GOOGLE_BOOKS_COUNTRY` |
| `403` or `429` with a quota/rate reason | `503 UPSTREAM_RATE_LIMITED` with `Retry-After` |
| `5xx`, timeout, or malformed JSON | `502 UPSTREAM_UNAVAILABLE` |
| Volume has no title | `400 VALIDATION_ERROR` with `details.reason = "volume has no title"` |

Google's status and body are logged with the request ID. They are never
returned to the client, and the API key never appears in logs.

Import race: two users importing the same volume at once both get the same
book. A different volume with an ISBN-13 that is already in the library raises
`23505` on `books_isbn13_unique`; the service returns the existing book with
`200` instead of `409`.

`PATCH /api/books/{bookId}` on a book with `source = "google"` returns
`403 FORBIDDEN`.

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
