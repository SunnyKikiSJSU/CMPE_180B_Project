# Book Metaverse Portal - Implementation Plan

**Course:** CMPE-180B

**Plan date:** October 31, 2026
**Scope:** The current `Team_Project` implementation and the scripts needed to complete the database MVP

## 1. Purpose

This plan describes the repository as it exists now, identifies the missing
layers between the frontend and the database, and proposes the scripts that
should be added under each part of the project tree.

The main design principle is that PostgreSQL remains the source of truth. The
frontend must use the Go API, the API must use explicit SQL, and database rules
must remain visible through migrations, constraints, functions, triggers,
transactions, indexes, and reproducible tests.

S3, semantic search, advanced 3D features, and real-time social features remain
outside the database MVP.

## 2. Current Implementation Snapshot

Updated 2026-11-12. This document is the original overview from October 31,
2026. The detailed, current plans are [`BACKEND_PLAN.md`](BACKEND_PLAN.md) and
[`FRONTEND_PLAN.md`](FRONTEND_PLAN.md); where they differ from later sections
here, they win.

| Area | Current state | Main gap |
| --- | --- | --- |
| Frontend | Next.js 15, React 19, TypeScript, Tailwind CSS 4 UI across 11 routes; `npm run lint` and `npx tsc --noEmit` pass | Pages use mock or empty-state data; the 8 declared service calls are not used; see `FRONTEND_PLAN.md` |
| Authentication | Login writes a demo value to `localStorage` | No API authentication yet (backend Phase 3) |
| Backend | Go module, minimal server (`/`, `/healthz`, `/readyz`) on `database/sql`, Dockerfile, sqlc-generated query code in `backend/internal/db` | No config, `pgxpool`, router, middleware, auth, or business endpoints (backend Phase 2 onward) |
| Database | Migrations `000001`-`000006` with constraints, triggers, functions, search and reporting indexes, and Google Books source columns; 66 named queries; seeds `001`-`004` (201 books, 51 users); verified on a clean PostgreSQL 18 database | No validation, concurrency, or performance evidence scripts yet |
| API contract | `docs/api/openapi.yaml` 0.2.0 (40 paths, 55 operations), `errors.md`, `examples.md` | Not implemented by the server yet |
| Docker | `postgres`, `api`, and `tools` services (`migrate`, `seed`, `sqlc`) locally; RDS variant | No `web` service because `frontend/Dockerfile` does not exist |
| Infrastructure | Terraform provisions RDS (PostgreSQL 18), a private security group, a Secrets Manager secret for the database credentials, and an EC2 IAM role/profile | No EC2 instance yet ([`AWS_PLAN.md`](AWS_PLAN.md)), custom network, or S3 resources |
| Documentation | `README.md`, `DEVELOPMENT.md`, `BACKEND_PLAN.md`, `FRONTEND_PLAN.md`, `docs/`, database READMEs | No schema dictionary, query catalog, runbooks, or evidence files yet |
| Testing | Frontend lint and type check pass; database checks run in throwaway containers | No Go, frontend unit, end-to-end, concurrency, or performance test suites |
| ER model | `ER_diagram.png` and its guide `ER_diagram.md`, committed, matching migrations `000001`-`000006` | Four cardinality labels at `BOOKS` on the activity relationships still read `1..1` (see `ER_diagram.md` §4) |

### Confirmed dependencies

- The frontend installs and builds with `npm`.
- TanStack Query, React Hook Form, and Zod are installed but not used yet.
- Docker Compose is installed and both Compose files resolve.
- On the Windows host, backend work runs in WSL (Ubuntu). Go 1.27 is installed
  in WSL; `sqlc`, `psql`, and `migrate` run as Docker images. Terraform is not
  on the local `PATH`.
- PostgreSQL 18 is the selected local and cloud database version.

## 3. Current Project Tree

Generated directories such as `frontend/.next/` and `frontend/node_modules/` are
excluded. Individual migration, query, sqlc output, and frontend source files
are summarized.

```text
Team_Project/
|-- .env.example
|-- README.md, PLAN.md, DEVELOPMENT.md
|-- BACKEND_PLAN.md, FRONTEND_PLAN.md, AWS_PLAN.md
|-- ER_diagram.png, ER_diagram.md
|-- docker-compose.yml, docker-compose-rdbs.yml
|
|-- backend/
|   |-- Dockerfile, .dockerignore, go.mod, go.sum, sqlc.yaml, README.md
|   |-- cmd/server/main.go
|   `-- internal/db/            # sqlc-generated code (13 files), README.md
|
|-- database/
|   |-- migrations/             # 000001-000006 up/down, README.md
|   |-- schema/schema.sql       # reduced mirror read by sqlc, README.md
|   |-- queries/                # 10 .sql files, 66 named queries, README.md
|   `-- seeds/                  # 001-004, README.md
|
|-- docs/
|   |-- README.md
|   `-- api/                    # openapi.yaml, errors.md, examples.md
|
|-- frontend/
|   |-- .env.local.example, next.config.ts, package.json, tsconfig.json, README.md
|   `-- src/
|       |-- app/                # 11 routes: /, /explore, /login, (workspace)/*
|       |-- components/         # 14 components (5 unused)
|       |-- data/mock-data.ts
|       |-- lib/api-client.ts
|       |-- services/           # book-service.ts, reading-service.ts
|       `-- types/domain.ts
|
|-- infra/                      # Terraform: RDS, security group, IAM, Secrets Manager
|
`-- scripts/
    |-- dev.sh                  # ./scripts/dev.sh local|rds (run from WSL)
    |-- install-go-wsl.sh
    `-- validate_openapi.py
```

## 4. Target Project Tree

The following additions keep responsibilities separated while allowing each
layer to be tested independently.

```text
Team_Project/
|-- .env.example
|-- README.md
|-- PLAN.md
|-- docker-compose.yml
|-- scripts/
|   |-- check.sh
|   |-- dev.sh
|   `-- reset-db.sh
|
|-- backend/
|   |-- cmd/server/main.go
|   |-- internal/
|   |   |-- auth/
|   |   |-- books/
|   |   |-- database/
|   |   |-- handlers/
|   |   |-- middleware/
|   |   |-- repositories/
|   |   `-- services/
|   |-- .dockerignore
|   |-- Dockerfile
|   |-- go.mod
|   |-- go.sum
|   `-- sqlc.yaml
|
|-- database/
|   |-- migrations/
|   |   |-- 000001_initial_schema.up.sql
|   |   `-- 000001_initial_schema.down.sql
|   |-- seeds/
|   |   |-- 001_reference_data.sql
|   |   `-- 002_demo_activity.sql
|   |-- queries/
|   |-- functions/
|   |-- triggers/
|   |-- tests/
|   |-- concurrency/
|   |-- performance/
|   `-- scripts/
|       |-- migrate.sh
|       `-- seed.sh
|
|-- frontend/
|   |-- src/
|   |   |-- app/
|   |   |-- components/
|   |   |-- hooks/
|   |   |-- lib/
|   |   |-- providers/
|   |   |-- services/
|   |   |-- types/
|   |   `-- validation/
|   |-- tests/
|   |   `-- unit/
|   |-- .dockerignore
|   |-- Dockerfile
|   `-- package.json
|
|-- infra/
|   |-- ec2.tf                 # planned (AWS_PLAN.md)
|   |-- iam.tf
|   |-- network.tf
|   |-- outputs.tf
|   |-- rds.tf
|   |-- secrets.tf
|   |-- variables.tf
|   `-- versions.tf
|
`-- docs/
    |-- README.md
    |-- architecture/
    |-- api/
    |-- database/
    |-- diagrams/
    |-- runbooks/
    `-- evidence/
```

## 5. Shell Automation

The project is Linux-based and will be developed through WSL. It will keep only
five Bash scripts. Other workflows use Go, npm, Docker Compose, `psql`, and
Terraform commands directly.

### 5.1 Root scripts

| Script | Responsibility |
| --- | --- |
| `scripts/dev.sh` | Start PostgreSQL, wait for health, apply migrations and seeds, then start the API and frontend with clear logs |
| `scripts/check.sh` (optional) | Run Compose validation, frontend lint/typecheck/build, Go tests, database validation, and `terraform validate`; return one non-zero status if any check fails |
| `scripts/reset-db.sh` | Recreate the local database volume, run all migrations, and load seeds; require explicit confirmation because it destroys local data |

`dev.sh` and `reset-db.sh` should fail fast if required environment variables
are missing. `reset-db.sh` should require explicit confirmation and must never
print passwords.

### 5.2 Database scripts

| Script | Responsibility |
| --- | --- |
| `database/scripts/migrate.sh` | Run the Compose `migrate` service and verify the resulting schema version |
| `database/scripts/seed.sh` | Load ordered seed files and reject non-idempotent duplicate inserts |

Database implementation order:

1. Translate the ER diagram into normalized `*.up.sql` and `*.down.sql`
   migrations.
2. Add constraints for ratings, progress, unique reviews and ratings, and
   junction-table composite keys.
3. Add idempotent reference and demonstration seed data.
4. Add explicit SQL query files and `sqlc` configuration.
5. Add functions and triggers for rating summaries and protected derived data.
6. Add transaction, concurrency, and rollback evidence.
7. Add indexes and before-and-after performance measurements.

The current ER model includes `users`, `user_profiles`, `books`, `authors`,
`categories`, `book_authors`, `book_categories`, `shelves`,
`user_shelf_items`, `reading_progress`, `reading_history`, `ratings`,
`reviews`, and `book_rating_summaries`.

### 5.3 Backend workflow

The Go backend does not need backend-specific shell scripts. Use direct
commands:

```bash
go run ./cmd/server
go test ./...
go vet ./...
sqlc generate
```

The Go API should expose:

- `/healthz` for process health.
- `/readyz` for PostgreSQL connectivity.
- `/api/auth`, `/api/users`, `/api/books`, `/api/authors`, `/api/categories`,
  `/api/shelves`, `/api/reading-progress`, `/api/reading-history`,
  `/api/reviews`, and `/api/reports` (ratings live under
  `/api/books/{id}/ratings`).
- `/api/external/books` (Google Books search through the server) and
  `/api/books/import` (adds one Google volume to the library when a user acts
  on it). `BACKEND_PLAN.md` is the detailed contract.

Integration tests should use a disposable database and deterministic fixtures.
They should cover constraint errors, rollback behavior, duplicate review and
rating races, and transaction boundaries.

### 5.4 Frontend workflow

Add these commands to `frontend/package.json`:

| Command | Responsibility |
| --- | --- |
| `npm run typecheck` | Run `tsc --noEmit` |
| `npm run test` | Run the Vitest unit tests |
| `npm run check` | Run lint, typecheck, unit tests, and production build in the correct order |

The frontend does not need a separate shell script.

Required frontend integration work:

- Add `src/providers/QueryProvider.tsx` and wire it into the root layout
  (file names and phases: [`FRONTEND_PLAN.md`](FRONTEND_PLAN.md)).
- Replace `mock-data.ts` usage with TanStack Query calls through
  `bookService` and `readingService`.
- Replace the demo `localStorage` session in `LoginPage` and `AuthBoundary`
  with real JWT/session behavior.
- Add React Hook Form and Zod schemas for login, registration, reviews,
  ratings, shelves, and progress.
- Expand `src/types/domain.ts` to match the database and OpenAPI contracts,
  including books, authors, categories, shelves, summaries, and timestamps.
- Move shared loading, error, and empty-state behavior into hooks.
- Verify public browsing, login, catalog search, shelf movement, progress
  updates, and review submission with the manual UI checklist
  (`FRONTEND_PLAN.md` §6.3).

### 5.5 Infrastructure workflow

Infrastructure does not need shell wrappers. Run Terraform directly and
review each plan before applying it:

```bash
terraform fmt -check
terraform init
terraform validate
terraform plan -out=rds-test.tfplan
terraform apply rds-test.tfplan
```

The infrastructure plan should keep development and production settings
separate. RDS should remain private for deployed application use. Temporary
local access should be restricted to one `/32` CIDR, removed after testing, and
never opened to `0.0.0.0/0`.

Application infrastructure still needed after the database MVP:

- One EC2 instance running the Compose stack (API and web), reached over HTTP.
- The instance's security group wired to the RDS security group.
- A backup and restore check.
- Post-MVP S3 bucket and the IAM policy it needs.

Details and tasks: [`AWS_PLAN.md`](AWS_PLAN.md).

### 5.6 Documentation

Restore the missing `docs/` directory and use it for:

- `docs/architecture/`: current system diagrams and decisions.
- `docs/api/`: OpenAPI contract, endpoint examples, and error format.
- `docs/database/`: schema dictionary, query catalog, and migration notes. The
  ER diagram is `ER_diagram.png` with its guide `ER_diagram.md` at the top
  level of `Team_Project/`.
- `docs/diagrams/`: other reviewed diagrams with reliable names.
- `docs/runbooks/`: local setup, RDS connection, migration, seed, backup, and
  restore procedures.
- `docs/evidence/`: transaction, concurrency, validation, and performance
  results.

`ER_diagram.png` and `ER_diagram.md` stay where they are and are linked from the
root README. They still need to be added to Git on purpose.

## 6. Implementation Phases

### Phase 0 - Baseline cleanup

- Restore `docs/README.md` so the current root README link works.
- Update the frontend README and root README status to reflect the implemented
  UI.
- Agree on the OpenAPI endpoint and error-response contract.
- Review uncommitted Compose and package-lock changes before further work.
- Confirm PostgreSQL 18 support in the selected AWS region.

### Phase 1 - Database foundation

- Write the initial schema migration and rollback.
- Add constraints, junction keys, and summary structures from the ER diagram.
- Add deterministic seed data and a database reset workflow.
- Add query files and configure `sqlc`.

Exit criteria: a clean database can be created, seeded, queried, and reset
using scripts without manual SQL.

### Phase 2 - Backend foundation

- Create the Go module, configuration loader, connection pool, and health
  endpoints.
- Add authentication, middleware, repository, service, and handler layers.
- Implement catalog, search, shelf, progress, review, rating, and report
  endpoints through explicit SQL.
- Add unit and integration tests.

Exit criteria: the API health check reports database readiness, and the main
endpoints are covered by `httptest` smoke tests.

### Phase 3 - Frontend integration

- Add TanStack Query and the shared API client provider.
- Replace mock data and the demo login with API-backed behavior.
- Implement forms and entity detail views.
- Add loading, empty, error, and conflict states.

Exit criteria: no workspace page depends on `mock-data.ts` or a fake
`localStorage` session.

### Phase 4 - Database evidence and quality

- Add transaction commit and rollback demonstrations.
- Add multi-session concurrency scripts.
- Add at least eight key queries and all required complex SQL.
- Add indexes and capture before-and-after plans and timings.
- Complete database validation and the manual UI checklist.

Exit criteria: required database behavior can be reproduced from scripts and
the evidence is documented.

### Phase 5 - Container and AWS integration

- Add backend and frontend Dockerfiles and `.dockerignore` files.
- Add `api` and `web` services to Compose with health checks.
- Deploy the API, frontend, and private RDS configuration.
- Keep secrets in the gitignored `.env`, never in committed files.
- Read container logs with `docker compose logs`.

Exit criteria: the deployed application uses RDS for all core operations and
can be demonstrated without local database access.

### Phase 6 - Post-MVP extensions

- Add S3-backed cover and avatar uploads.
- Add optional semantic search, richer reports, or other extensions only after
  the database MVP is stable.

## 7. Script Priority

| Priority | Script | Reason |
| --- | --- | --- |
| P0 | `database/scripts/migrate.sh` | Nothing downstream can be tested without a schema |
| P0 | `database/scripts/seed.sh` | Reproducible data is required for API and SQL evidence |
| P1 | `scripts/dev.sh` | Starts the local development stack consistently |
| P2 | `scripts/reset-db.sh` | Reproduces a clean local database on demand |
| P2 | `scripts/check.sh` | Optional: one consistent local quality gate |

## 8. Script Quality Rules

- Scripts must be non-interactive unless they explicitly require confirmation
  for destructive actions.
- Scripts must run under WSL/Linux and use LF line endings.
- Scripts must stop on the first error and return a non-zero exit code.
- Scripts must read secrets from `.env` or the process environment; secrets
  must never be committed or echoed.
- Destructive database scripts must clearly identify the target before
  proceeding.
- Repeated runs should be safe where practical.
- Every script should have a short usage message and a documented expected
  outcome.
- SQL validation scripts should assert results, not merely execute statements.

## 9. Immediate Next Actions

Done: API contract and error format (`docs/api/`), migrations `000001`-`000006`,
seeds, the Go module with `/healthz` and `/readyz`, the backend Dockerfile, and
the `docs/` tree.

Next:

1. Backend Phase 2: config, `pgxpool`, `httpx`, request ID / logging / recovery
   / timeout / body-limit middleware, router skeleton, and a database-ping
   `/readyz` (`BACKEND_PLAN.md` §13).
2. Frontend F0–F1: `typecheck` script, Vitest, the rewritten API client, and
   expanded types, built against mock responses (`FRONTEND_PLAN.md` §7).
3. Backend Phase 3 and frontend F2 together: real login against the cookie
   session.
4. Add `frontend/Dockerfile` and the Compose `web` service.
5. Add `npm run check`.
6. Add `ec2.tf` and deploy (`AWS_PLAN.md` §4).
