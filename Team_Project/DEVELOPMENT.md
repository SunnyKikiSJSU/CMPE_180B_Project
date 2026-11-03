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

The RDS security group must allow TCP port `5432` from either:

- The current public IP as a `/32` rule when connecting from a laptop.
- The backend or EC2 security group when running inside the same VPC.

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
