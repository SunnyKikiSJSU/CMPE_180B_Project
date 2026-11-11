# Generated sqlc Code

This directory is the output target for `sqlc generate` (see
`backend/sqlc.yaml`). The generated files are committed so the Docker build and
`go test` do not need a separate sqlc installation.

## Generate with Docker

The repository includes a pinned `sqlc/sqlc:1.27.0` tool service in
`docker-compose.yml`.

```bash
# From Team_Project/, after creating .env
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc
```

The container mounts `backend/` and `database/` under `/src` and runs
`sqlc generate` from `/src/backend`, so `sqlc.yaml` can resolve:

- `../database/schema/schema.sql`
- `../database/queries`

## Generate with a local sqlc install

If sqlc is already installed:

```bash
cd backend
sqlc generate
```

## After generation

The generated code imports `github.com/google/uuid`. Once Go is installed
locally, run:

```bash
cd backend
go mod tidy
```

Do not hand-edit generated files. Change `database/queries/*.sql` or
`database/schema/schema.sql` and regenerate.
