# Development Setup

This guide explains how to start the Book Metaverse backend with either:

- A local PostgreSQL database in Docker
- An existing Amazon RDS for PostgreSQL database

The workflow is designed for Linux and WSL. Run all commands from the Ubuntu
WSL terminal unless a step says otherwise.

## Prerequisites

1. Docker Desktop is running.
2. Docker Desktop has WSL integration enabled for Ubuntu.
3. The repository is available from WSL.
4. A local `.env` file exists with the required values.

Open the project:

```bash
wsl -d Ubuntu
cd "/mnt/d/SJSU/Database System/project/CMPE_180B_Project/Team_Project"
```

The WSL path may be different on another computer. Use the path where the
repository is mounted.

### Install Go in WSL (one-time)

The backend must be built and tested from WSL on a Windows host. Install a
current Go SDK without sudo:

```bash
bash scripts/install-go-wsl.sh
source ~/.bashrc
go version
```

The script downloads the current Go release to `$HOME/go-sdk` and adds
`GOROOT`, `GOPATH`, and `PATH` entries to `~/.bashrc`. Open a new WSL terminal
if `go` is still not found.

## Environment File

Create `.env` from the example if it does not already exist:

```bash
cp -n .env.example .env
```

Edit `.env` in a text editor. Never commit this file.

### Local database values

These values are required for `./scripts/dev.sh local`:

```dotenv
POSTGRES_DB=book_metaverse
POSTGRES_USER=book_app
POSTGRES_PASSWORD=choose-a-local-password
POSTGRES_PORT=5432
API_PORT=8080
JWT_SECRET=choose-a-long-local-secret
```

The local PostgreSQL data is stored in the `book-metaverse_postgres_data`
Docker volume.

### RDS values

These values are required for `./scripts/dev.sh rds`:

```dotenv
RDS_HOST=your-rds-endpoint.amazonaws.com
RDS_PORT=5432
RDS_DB=your_database_name
RDS_USER=your_database_user
RDS_PASSWORD=your_database_password
RDS_SSLMODE=require
API_PORT=8080
JWT_SECRET=choose-a-long-local-secret
```

Do not place real credentials in `.env.example` or commit `.env`.

### Google Books values (both modes)

Book search calls the Google Books API from the API server. These are
optional for local work:

```dotenv
GOOGLE_BOOKS_API_KEY=your-google-books-api-key
GOOGLE_BOOKS_COUNTRY=US
```

Without a key, Google allows only a small anonymous quota. To create a key,
enable the Books API in a Google Cloud project, create an API key, and restrict
it to the Books API. Keep it in `.env` only; it must never go into a
`NEXT_PUBLIC_*` variable or the frontend. Automated tests use a fake Google
server and need no key.

The RDS security group must allow TCP port `5432` from either:

- The current public IP as a `/32` rule when connecting from a laptop.
- The backend or EC2 security group when running inside the same VPC.

## Generate Database Code (sqlc)

The repository runs sqlc through a pinned Docker container, so a local sqlc
installation is not required.

From the `Team_Project` directory, after creating `.env`:

```bash
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc
```

This writes `backend/internal/db` from `database/queries` and
`database/schema/schema.sql`. The sqlc container needs no database and does not
start PostgreSQL.

After installing Go locally, resolve the generated imports:

```bash
cd backend
go mod tidy
```

## Phase 1 Verification

Run these from WSL in the `Team_Project` directory:

```bash
# 1. Generate backend/internal/db
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse --profile tools run --rm sqlc

# 2. Resolve generated imports
cd backend
go mod tidy
cd ..

# 3. Apply migrations, load seeds, and start the API
./scripts/dev.sh local

# 4. Confirm the seed volume
docker compose --env-file .env --file docker-compose.yml --project-name book-metaverse exec postgres \
  psql -U book_app -d book_metaverse -c \
  "SELECT (SELECT count(*) FROM books) AS books,
          (SELECT count(*) FROM authors) AS authors,
          (SELECT count(*) FROM categories) AS categories,
          (SELECT count(*) FROM users) AS users,
          (SELECT count(*) FROM shelves) AS shelves,
          (SELECT count(*) FROM ratings) AS ratings,
          (SELECT count(*) FROM reviews) AS reviews;"
```

Expected: **201 books, 51 authors, 16 categories, 51 users, 301 ratings,
301 reviews**, and a non-zero shelf count.

Optional OpenAPI structure check:

```bash
python3 -m pip install --user pyyaml
python3 scripts/validate_openapi.py docs/api/openapi.yaml
```

## Start With Local PostgreSQL

Run:

```bash
./scripts/dev.sh local
```

The script performs these steps:

1. Checks that Docker is running.
2. Stops the RDS-backed Compose stack if it is running.
3. Starts local PostgreSQL and waits for it to become healthy.
4. Applies database migrations.
5. Loads the seed data.
6. Builds and starts the Go API container.
7. Waits for `/readyz` to confirm a real database connection.

Successful output includes:

```text
Backend is ready.
Target:   local
Ready:    http://127.0.0.1:8080/readyz
Response: {"database":"connected","status":"ready"}
```

## Start With Amazon RDS

Run:

```bash
./scripts/dev.sh rds
```

The script performs these steps:

1. Checks the required RDS values in `.env`.
2. Tests TCP connectivity to the RDS endpoint and port.
3. Stops the local Compose stack if it is running.
4. Percent-encodes the database credentials for the migration URL.
5. Applies migrations to RDS.
6. Loads the seed data.
7. Builds and starts the RDS-backed Go API container.
8. Waits for `/readyz` to confirm the API can query RDS.

Successful output includes:

```text
Backend is ready.
Target:   rds
Ready:    http://127.0.0.1:8080/readyz
Response: {"database":"connected","status":"ready"}
```

Do not run the local and RDS modes at the same time. Both expose the API on the
same port, and `dev.sh` automatically stops the other mode before starting.

## Verify the Backend

Check process health:

```bash
curl http://127.0.0.1:8080/healthz
```

Expected response:

```json
{"service":"book-metaverse-api","status":"ok"}
```

Check database readiness:

```bash
curl http://127.0.0.1:8080/readyz
```

Expected response:

```json
{"database":"connected","status":"ready"}
```

An unknown route should return `404`:

```bash
curl -i http://127.0.0.1:8080/api/books
```

## View Logs

Local mode:

```bash
docker compose \
  --env-file .env \
  --file docker-compose.yml \
  --project-name book-metaverse \
  logs --follow api
```

RDS mode:

```bash
docker compose \
  --env-file .env \
  --file docker-compose-rdbs.yml \
  --project-name book-metaverse-rds \
  logs --follow api
```

## Stop the Environment

Stop local mode while preserving the database volume:

```bash
docker compose \
  --env-file .env \
  --file docker-compose.yml \
  --project-name book-metaverse \
  down --remove-orphans
```

Stop RDS mode. This removes the API container but does not delete or stop RDS:

```bash
docker compose \
  --env-file .env \
  --file docker-compose-rdbs.yml \
  --project-name book-metaverse-rds \
  down --remove-orphans
```

## Troubleshooting

| Problem | Check |
| --- | --- |
| Docker is not running | Start Docker Desktop and confirm WSL integration is enabled |
| Missing environment values | Compare `.env` with the example values above |
| Local port already in use | Run `./scripts/dev.sh local` again so it stops the other Compose stack |
| RDS TCP connection fails | Check RDS public access, route tables, network ACLs, and the `/32` security-group rule |
| `/healthz` works but `/readyz` returns `503` | Check database host, port, name, username, password, and SSL mode |
| API container exits during build | Run the log command for the selected mode |
| Migration fails | Confirm the database user can create and alter tables |

## Important Files

| File | Purpose |
| --- | --- |
| `.env` | Local credentials and configuration; excluded from Git |
| `.env.example` | Documented environment variable names |
| `scripts/dev.sh` | Linux/WSL startup entry point for both database modes |
| `docker-compose.yml` | Local PostgreSQL and local API stack |
| `docker-compose-rdbs.yml` | RDS-backed API, migration, seed, and client services |
| `backend/` | Go API source and Dockerfile |
| `database/migrations/` | Versioned PostgreSQL schema changes |
| `database/seeds/` | Reproducible development data |

## Security Reminders

- Never commit `.env` or Terraform state files.
- Do not open an RDS security group to `0.0.0.0/0`.
- Remove temporary public database access after testing.
- Use a dedicated application database user instead of the RDS master user
  when the application roles and permissions are finalized.
