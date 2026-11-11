# Backend

Go API for the Book Metaverse Portal.

## Requirements

- Docker Desktop for PostgreSQL, migrations, seeds, and the sqlc tool.
- Go 1.25+ for local backend builds, tests, and `go mod tidy`.

## Endpoints

- `/healthz` checks that the API process is running.
- `/readyz` checks that the API can connect to PostgreSQL.

The API reads database connection details from `DB_HOST`, `DB_PORT`, `DB_NAME`,
`DB_USER`, `DB_PASSWORD`, and `DB_SSLMODE`. It also accepts a complete
`DATABASE_URL`.

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
