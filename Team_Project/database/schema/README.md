# sqlc Schema Mirror

`database/migrations/` is the source of truth for the PostgreSQL schema.

`schema.sql` is a reduced mirror used only by `sqlc`. It contains tables and
columns so sqlc can type-check queries, but it deliberately omits:

- extensions,
- generated-column expressions,
- indexes and constraints that sqlc does not need,
- triggers,
- PL/pgSQL functions,
- backfill statements.

Update this file whenever a migration adds, removes, or renames a table or
column. The next `sqlc generate` run reads this file, so a mismatch will show up
as a generated-code diff or a compile error in the repository layer.

Do not run this file against a database. Apply the versioned migrations instead:

```bash
docker compose --profile tools run --rm migrate
```
