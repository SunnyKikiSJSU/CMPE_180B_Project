# Backend

Go API for the Book Metaverse Portal.

## Requirements

- Docker Desktop for PostgreSQL, migrations, seeds, and the sqlc tool.
- Go 1.26+ for local backend builds, tests, and `go mod tidy`.

## Layout

`handler -> service -> repository -> pgxpool -> PostgreSQL` (BACKEND_PLAN §3).

| Package | Purpose |
| --- | --- |
| `cmd/server` | Loads config, resolves database credentials, opens the pool, serves HTTP, shuts down gracefully |
| `internal/config` | Reads and validates environment variables; never logs secret values |
| `internal/database` | `pgxpool` (10 connections max) and the Secrets Manager credential reader |
| `internal/auth` | bcrypt (cost 12) and HS256 session tokens (60 minutes, issuer `book-metaverse`) |
| `internal/middleware` | Request ID, panic recovery, request log, 10 s timeout, 1 MiB body limit, `no-store`, authentication |
| `internal/httpx` | Bounded JSON decoding (unknown fields rejected), JSON responses, the error envelope and SQLSTATE mapping |
| `internal/repository` | sqlc rows to domain types |
| `internal/services` | Validation, business rules, transactions |
| `internal/handlers` | HTTP layer only; no SQL |
| `internal/router` | Every contract route, its auth group, and middleware order |

## Endpoints

Implemented (Phases 2-4): `/`, `/healthz`, `/readyz` (database ping, `503`
when PostgreSQL is unreachable), `POST /api/auth/register`, `POST /api/auth/login`,
`POST /api/auth/logout`, `GET /api/auth/me`, `GET|PATCH|DELETE /api/users/me`,
`PUT /api/users/me/password`, `GET|POST /api/books`, `GET|PATCH /api/books/{bookId}`,
`GET /api/authors`, `GET /api/authors/{authorId}`, `GET /api/categories`, and
`GET /api/categories/{categoryId}`. Every other route in `docs/api/openapi.yaml` is
registered with its auth requirement and returns `501 NOT_IMPLEMENTED` until its
phase is built. Unknown routes return `404 NOT_FOUND`.

Sessions: the `bm_session` cookie (`HttpOnly; SameSite=Lax; Path=/`, plus
`Secure` when `COOKIE_SECURE=true`) or `Authorization: ******` for curl
and tests. Logout clears the cookie; a copied Bearer token stays valid until it
expires.

`PATCH /api/users/me` and `PATCH /api/books/{bookId}`: a missing or `null` field
is left unchanged (the queries use `coalesce`), so fields cannot be cleared to
`null` yet.

Catalog notes:

- Lists are raw JSON arrays with `X-Total-Count`, `X-Page`, and `X-Page-Size`
  (`page` 1-10000, `pageSize` 1-100, default 20). A page past the end returns
  `[]` with the real total.
- `GET /api/books` without `q` sorts newest first; with `q`, by relevance
  (full-text rank, then title). `q` and `title` are at most 200 characters.
- Author and category detail return one page of `books`; the pagination
  headers page those books.
- `GET /api/books/{bookId}` adds `myProgress` and `myRating` for a signed-in
  caller (with `Cache-Control: no-store`).
- `POST /api/books` creates `source = user` books; author and category names
  are trimmed and de-duplicated case-insensitively (at most 20 each).
  A duplicate ISBN-13 is `409`. `PATCH` on a `source = google` book is `403`.

## Configuration

| Variable | Notes |
| --- | --- |
| `JWT_SECRET` | Required. Outside `APP_ENV=local` the API refuses the example values from `.env.example` and Compose |
| `DATABASE_URL` | Complete connection string; wins over everything below |
| `DB_SECRET_ARN` | Secrets Manager secret from `infra/` (`terraform output -raw db_secret_arn`). Read once at startup with the default AWS credential chain (on EC2, the instance profile); `DB_SSLMODE` defaults to `require`. Region from `AWS_REGION` or the ARN |
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE` | Used when neither of the above is set; `DB_PASSWORD` is required |
| `PORT`, `APP_ENV`, `COOKIE_SECURE`, `GOOGLE_BOOKS_API_KEY`, `GOOGLE_BOOKS_COUNTRY` | See `.env.example` |

The startup log records which database source was used, never the values.

## Tests

```bash
cd backend
go test ./...          # unit and handler tests, no database needed
go vet ./...

# Integration tests against the Compose database (migrations and seeds applied):
DATABASE_URL='postgres://book_app:<password>@localhost:5432/book_metaverse?sslmode=disable' \
  go test -tags integration ./internal/integration/
```

## Generate sqlc code

`sqlc` is provided as a pinned Docker tool service; a local sqlc installation
is not required.

```bash
# From Team_Project/, after creating .env
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc
```

The container mounts `backend/` and `database/` under `/src`, runs from
`/src/backend`, and writes `backend/internal/db` from `database/queries` and
`database/schema/schema.sql`.

After Go is installed locally, resolve the generated imports:

```bash
cd backend
go mod tidy
```

Do not edit `internal/db` by hand; change the query/schema files and rerun the
container.

Run the complete environment from WSL:

```bash
./scripts/dev.sh local
./scripts/dev.sh rds
```

See [Development Setup](../DEVELOPMENT.md) for environment variables,
troubleshooting, logs, and shutdown commands.
