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

| Area | Current state | Main gap |
| --- | --- | --- |
| Frontend | Next.js 15, React, TypeScript, Tailwind CSS UI is implemented across 11 routes | Pages use mock or empty-state data; API services are not called yet |
| Authentication | Login writes a demo value to `localStorage` | No JWT, password hashing, API authentication, or protected server-side session |
| Backend | Only `backend/README.md` exists | No Go module, server, handlers, repositories, services, or tests |
| Database | Only migration and seed directory instructions exist | No schema, migrations, seed data, queries, functions, triggers, or validation tests |
| Docker | PostgreSQL 18, `migrate`, and `seed` are defined | No API or web service because their Dockerfiles do not exist |
| Infrastructure | Terraform provisions RDS, a security group, Secrets Manager, and an EC2 IAM role/profile | No application deployment, custom network, backup policy, alerting, or S3 resources |
| Documentation | Root README and infrastructure README exist | `docs/` is missing, the root README links to `docs/README.md`, and several status statements are stale |
| Testing | Frontend lint and production build pass | No unit, API, database, concurrency, performance, or end-to-end tests |
| ER model | `ER_diagram.png` and its guide `ER_diagram.md` describe the relational model, including the Google Books columns from migration `000006` | The files are untracked in Git; the model is implemented as SQL in migrations `000001`-`000006` |

### Confirmed dependencies

- The frontend installs and builds with `npm`.
- `bookService` and `readingService` exist but are not imported by any page or
  component.
- TanStack Query, React Hook Form, and Zod are installed but not used yet.
- Docker Compose is installed and the Compose file resolves successfully.
- Go, `sqlc`, and Terraform are not currently available in the local `PATH`.
- PostgreSQL 18 is the selected local and cloud database version.

## 3. Current Project Tree

Generated directories such as `frontend/.next/` and `frontend/node_modules/` are
excluded from this tree because they are ignored build output.

```text
Team_Project/
|-- .env.example
|-- README.md
|-- PLAN.md
|-- docker-compose.yml
|-- ER_diagram.png
|-- ER_diagram.md
|
|-- backend/
|   `-- README.md
|
|-- database/
|   |-- migrations/
|   |   `-- README.md
|   `-- seeds/
|       `-- README.md
|
|-- frontend/
|   |-- .env.local.example
|   |-- .gitignore
|   |-- README.md
|   |-- eslint.config.mjs
|   |-- next-env.d.ts
|   |-- next.config.ts
|   |-- package-lock.json
|   |-- package.json
|   |-- postcss.config.mjs
|   |-- tsconfig.json
|   `-- src/
|       |-- app/
|       |   |-- (workspace)/
|       |   |   |-- layout.tsx
|       |   |   |-- authors/page.tsx
|       |   |   |-- catalog/page.tsx
|       |   |   |-- dashboard/page.tsx
|       |   |   |-- progress/page.tsx
|       |   |   |-- reports/page.tsx
|       |   |   |-- reviews/page.tsx
|       |   |   |-- settings/page.tsx
|       |   |   `-- shelves/page.tsx
|       |   |-- explore/page.tsx
|       |   |-- login/page.tsx
|       |   |-- globals.css
|       |   |-- layout.tsx
|       |   `-- page.tsx
|       |-- components/
|       |   |-- AppShell.tsx
|       |   |-- AuthBoundary.tsx
|       |   |-- BookCarousel.tsx
|       |   |-- DashboardView.tsx
|       |   |-- DiscoverySection.tsx
|       |   |-- EmptyState.tsx
|       |   |-- FilterBar.tsx
|       |   |-- Footer.tsx
|       |   |-- HeroSection.tsx
|       |   |-- Navbar.tsx
|       |   |-- PublicHome.tsx
|       |   |-- PublicNav.tsx
|       |   |-- SectionPage.tsx
|       |   `-- ShelvesSection.tsx
|       |-- data/
|       |   `-- mock-data.ts
|       |-- lib/
|       |   `-- api-client.ts
|       |-- services/
|       |   |-- book-service.ts
|       |   `-- reading-service.ts
|       `-- types/
|           `-- domain.ts
|
`-- infra/
    |-- .gitignore
    |-- .terraform.lock.hcl
    |-- README.md
    |-- iam.tf
    |-- network.tf
    |-- outputs.tf
    |-- random.tf
    |-- rds.tf
    |-- secrets.tf
    |-- variables.tf
    `-- versions.tf
```

## 4. Target Project Tree

The following additions keep responsibilities separated while allowing each
layer to be tested independently.

```text
Team_Project/
|-- .env.example
|-- .github/
|   `-- workflows/
|       `-- ci.yml
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
|   |   |-- e2e/
|   |   `-- unit/
|   |-- .dockerignore
|   |-- Dockerfile
|   |-- package.json
|   `-- playwright.config.ts
|
|-- infra/
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
| `scripts/check.sh` | Run Compose validation, frontend lint/typecheck/build, Go tests, database validation, and `terraform validate`; return one non-zero status if any check fails |
| `scripts/reset-db.sh` | Recreate the local database volume, run all migrations, and load seeds; require explicit confirmation because it destroys local data |
| `.github/workflows/ci.yml` | Run the same non-destructive checks as `scripts/check.sh` on pushes and pull requests |

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
- `/readyz` for PostgreSQL connectivity and migration readiness.
- `/api/auth`, `/api/users`, `/api/books`, `/api/search`, `/api/shelves`,
  `/api/progress`, `/api/reviews`, `/api/ratings`, and `/api/reports`.
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
| `npm run test` | Run component and service unit tests |
| `npm run test:e2e` | Run Playwright against a disposable local environment |
| `npm run check` | Run lint, typecheck, unit tests, and production build in the correct order |

The frontend does not need a separate shell script. API contract checks should
be implemented as an npm command and invoked by `scripts/check.sh`.

Required frontend integration work:

- Add `src/providers/QueryClientProvider.tsx` and wire it into the root layout.
- Replace `mock-data.ts` usage with TanStack Query calls through
  `bookService` and `readingService`.
- Replace the demo `localStorage` session in `LoginPage` and `AuthBoundary`
  with real JWT/session behavior.
- Add React Hook Form and Zod schemas for login, registration, reviews,
  ratings, shelves, and progress.
- Expand `src/types/domain.ts` to match the database and OpenAPI contracts,
  including books, authors, categories, shelves, summaries, and timestamps.
- Move shared loading, error, empty, and optimistic-update behavior into hooks.
- Add Playwright coverage for public browsing, login, catalog search, shelf
  movement, progress updates, and review submission.

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

- API container image and ECS service.
- Frontend deployment.
- Load balancer and private backend security-group wiring.
- CloudWatch logs and alarms.
- Backup and restore checks.
- Post-MVP S3 bucket and IAM policy.

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

Exit criteria: the API health check reports database readiness, and every MVP
endpoint is covered by at least one integration test.

### Phase 3 - Frontend integration

- Add TanStack Query and the shared API client provider.
- Replace mock data and the demo login with API-backed behavior.
- Implement forms and entity detail views.
- Add loading, empty, error, conflict, and optimistic-update states.

Exit criteria: no workspace page depends on `mock-data.ts` or a fake
`localStorage` session.

### Phase 4 - Database evidence and quality

- Add transaction commit and rollback demonstrations.
- Add multi-session concurrency scripts.
- Add at least eight key queries and all required complex SQL.
- Add indexes and capture before-and-after plans and timings.
- Complete database validation and Playwright end-to-end tests.

Exit criteria: required database behavior can be reproduced from scripts and
the evidence is documented.

### Phase 5 - Container and AWS integration

- Add backend and frontend Dockerfiles and `.dockerignore` files.
- Add `api` and `web` services to Compose with health checks.
- Deploy the API, frontend, and private RDS configuration.
- Verify Secrets Manager access through IAM rather than committed credentials.
- Add CloudWatch logging and basic alarms.

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
| P1 | `scripts/check.sh` | Gives developers and CI one consistent quality gate |
| P1 | `scripts/dev.sh` | Starts the local development stack consistently |
| P2 | `scripts/reset-db.sh` | Reproduces a clean local database on demand |

## 8. Script Quality Rules

- Scripts must be non-interactive in CI unless they explicitly require
  confirmation for destructive actions.
- Scripts must run under WSL/Linux and use LF line endings.
- Scripts must stop on the first error and return a non-zero exit code.
- Scripts must read secrets from `.env`, the process environment, or AWS
  Secrets Manager; secrets must never be committed or echoed.
- Destructive database scripts must clearly identify the target before
  proceeding.
- Repeated runs should be safe where practical.
- Every script should have a short usage message and a documented expected
  outcome.
- SQL validation scripts should assert results, not merely execute statements.

## 9. Immediate Next Actions

1. Approve the API contract and error format.
2. Write the initial PostgreSQL migration from the ER diagram.
3. Add a minimal seed set that supports local development.
4. Create the Go module and `/healthz` and `/readyz` endpoints.
5. Connect the frontend catalog to the API through TanStack Query.
6. Add `npm run typecheck` and `npm run check`.
7. Restore and organize the `docs/` tree.
8. Add backend and frontend Dockerfiles before extending Compose.
