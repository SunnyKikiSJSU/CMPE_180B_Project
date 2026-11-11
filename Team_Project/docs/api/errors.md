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
field name or database constraint. `requestId` matches the request log entry.

## Codes

| Code | HTTP | Meaning |
| --- | --- | --- |
| `VALIDATION_ERROR` | 400 | Malformed JSON, unknown field, missing field, invalid range or enum |
| `UNAUTHENTICATED` | 401 | Missing, malformed, or expired session |
| `FORBIDDEN` | 403 | Authenticated but the action is not allowed |
| `NOT_FOUND` | 404 | Resource missing or owned by another user |
| `CONFLICT` | 409 | Duplicate resource, duplicate review/rating, duplicate shelf name |
| `STALE_VERSION` | 409 | `reading_progress.version` does not match the current row |
| `RATE_LIMITED` | 429 | Authentication or general rate limit exceeded |
| `DATABASE_UNAVAILABLE` | 503 | The pool cannot reach PostgreSQL |
| `INTERNAL` | 500 | Unexpected failure; details are logged, not returned |

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
