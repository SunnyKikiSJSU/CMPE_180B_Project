# Seed Data

The Compose `seed` service runs every `.sql` file in this directory in lexical
order. All files are idempotent where practical, so the local database can be
refreshed without a manual reset.

## Files

| File | Purpose |
| --- | --- |
| `001_smoke_test.sql` | One connected row set that proves migrations, constraints, triggers, and connectivity work |
| `002_reference_data.sql` | 50 authors, 15 categories, 200 books (`source = 'seed'`), and their many-to-many links |
| `003_demo_activity.sql` | 50 users, default/custom shelves, 500 shelf items, 500 progress rows, 300 ratings, 300 reviews, and 500 history events |
| `004_refresh_summaries.sql` | Rebuilds `book_rating_summaries` after bulk inserts |

## Seed books and Google Books

Seed books are synthetic. Their `google_books_id` values (`seed-book-<n>`,
`smoke-test-google-books-id`) are placeholders, not Google volumes, and the
rows are marked `source = 'seed'` so they are never shown with Google
attribution. Real books come from Google Books search and are imported one at
a time when a user acts on them.

**Never add Google Books data to a seed file.** This repository is public, and
committing Google content would redistribute it. Reports and evidence use the
seed books because their activity is fixed and reproducible.

## Distribution

Each demo user shelves 10 distinct books: three from a popular pool (books
1-20) and seven spread over books 21-200. Every catalog book is on at least one
shelf, popular books have up to 8 readers, and 169 books have ratings with
values 1-5. This gives the popular, top-rated, ranking, and recommendation
reports a realistic long tail instead of a few identical books.

## Fixed timestamps

All seed timestamps derive from one anchor, `2026-10-31 12:00 UTC`. Shelf
activity runs from 2026-09-06 to 2026-10-28, and reviews cover 32 distinct
days. Report output is therefore identical on every run and every day.

Reports with a rolling window (`PopularBooks`, `ReviewActivity`) count back
from `now()`. When capturing evidence, pass a window that reaches back to
September 2026 (for example `days=120` in November 2026).

## Resetting an older seeded database

The distribution and numbering changed on 2026-11-10. A database seeded before
that date would get a mix of old and new pairs because every insert uses
`ON CONFLICT DO NOTHING`. Recreate the local volume before reseeding:

```bash
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse down --volumes
./scripts/dev.sh local
```

Do not run that against RDS without checking the target first; for RDS, drop
and recreate the database deliberately, then migrate and seed.

## Demo login

```text
reader1@example.com
Password123!
```

All demo users share this password.

## Password hashing note

Every password in this project is hashed with bcrypt, cost 12. The demo seed
uses the `pgcrypto` extension:

```sql
crypt('Password123!', gen_salt('bf', 12))
```

The Go auth service hashes new passwords with `golang.org/x/crypto/bcrypt` at
the same cost and verifies seeded and API-created accounts the same way, so no
rehash step is needed. bcrypt only reads the first 72 bytes of a password, so
the API accepts passwords of 8 to 72 bytes.

## Deterministic IDs

Seed records use fixed UUID namespaces so reruns do not create duplicates:

| Namespace | Records |
| --- | --- |
| `...-8001-...` | authors |
| `...-8002-...` | categories |
| `...-8003-...` | books |
| `...-8004-...` | demo users |
| `...-8005-...` | custom shelves |
| `...-8006-...` | ratings |
| `...-8007-...` | reviews |
| `...-8008-...` | reading history |
| `...-8009-...` | reading progress |

Do not change these values after evidence has been captured; doing so changes
the report results and invalidates recorded query output.
