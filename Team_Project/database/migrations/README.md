# Database Migrations

Place versioned `golang-migrate` migration pairs in this directory:

```text
000001_initial_schema.up.sql
000001_initial_schema.down.sql
000002_search_and_extensions.up.sql
000002_search_and_extensions.down.sql
000003_domain_rules.up.sql
000003_domain_rules.down.sql
000004_reporting_indexes.up.sql
000004_reporting_indexes.down.sql
000005_progress_and_history_fixes.up.sql
000005_progress_and_history_fixes.down.sql
```

The Compose `migrate` service mounts this directory at `/migrations` and runs
all pending migrations against the local PostgreSQL database.

## What each migration owns

| Migration | Responsibility |
| --- | --- |
| `000001_initial_schema` | Tables, constraints, foreign keys, initial indexes |
| `000002_search_and_extensions` | `pg_trgm`, `pgcrypto`, generated `books.search_vector`, GIN/trigram indexes |
| `000003_domain_rules` | `updated_at` triggers, default shelves/profile, status-shelf uniqueness, rating validation, rating/review summaries, progress validation, immutable history, `version`, shelf/progress functions, backfills |
| `000004_reporting_indexes` | Composite/partial indexes for dashboards and reports |
| `000005_progress_and_history_fixes` | Replaces three `000003` functions: page/status transitions in `fn_validate_reading_progress` (a finished book can be re-read, paused, or re-queued), `BM001` for stale versions in `fn_update_reading_progress`, and cascade-safe `fn_block_history_mutation`. The down file restores the `000003` definitions exactly. |

### Reading-progress rules (after `000005`)

The validation trigger owns the page/status relationship for every write path:

| Change | Result |
| --- | --- |
| Status becomes `completed` | `current_page` jumps to the last page |
| Status leaves `completed` and the page is still the last page | `current_page` restarts at 0 |
| Page lowered on a `completed` book with no status change | Status becomes `reading` |
| Page reaches the last page | Status becomes `completed` |
| Any change | `percent_complete` is recomputed when `page_count` is known |

`reading_history` rejects every `UPDATE` and every direct `DELETE`. Deletes that
arrive through the `users`/`books` foreign-key cascade are allowed, so account
and book deletion need no session setting.

These migrations are the schema source of truth. `database/schema/schema.sql`
is a reduced mirror used only by sqlc and must be kept in sync with table and
column changes.

The `migrate` service needs `MIGRATE_DATABASE_URL`. `./scripts/dev.sh local`
sets it for you. To run it directly from the repository root:

```bash
export MIGRATE_DATABASE_URL="postgres://book_app:<password>@postgres:5432/book_metaverse?sslmode=disable"
docker compose --profile tools run --rm migrate
```

Apply seeds after migrations:

```bash
docker compose --profile tools run --rm seed
```
