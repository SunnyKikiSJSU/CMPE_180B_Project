# Book Metaverse Frontend

Next.js 16, TypeScript, Tailwind CSS, and TanStack Query frontend for the Book Metaverse Portal.
Requires Node.js 20.9 or later.

## Local development

```bash
cp .env.local.example .env.local
npm ci
npm run dev
```

Open http://localhost:3000.

The browser calls `/api/...` on the page's own origin. `next.config.ts` forwards
those requests to the Go API at `API_PROXY_TARGET` (default
`http://localhost:8080`), so there is no CORS and no API URL in the browser
bundle. The target is read when `next dev` or `next build` starts, so a
production build points at the API given at build time
(`API_PROXY_TARGET=http://api:8080 npm run build` in Compose); `next start`
does not change it.

The Go API must be running for sign-in (`../backend/README.md`). Seed accounts
such as `reader1@example.com` / `Password123!` work after the seeds are loaded.

## Docker

`Dockerfile` builds a small production image from Next.js's standalone output
(the build sets `NEXT_OUTPUT=standalone`, which `next.config.ts` reads): Node.js 22 on Alpine, runs as the
non-root `node` user, listens on port 3000, and has a health check. Compose
builds it as the `web` service with `API_PROXY_TARGET=http://api:8080`:

```bash
./scripts/dev.sh local          # from Team_Project: database, API, and web
docker build -t book-metaverse-web --build-arg API_PROXY_TARGET=http://api:8080 .
```

## Status

Every page uses the API (FRONTEND_PLAN.md phases F0-F8): sign-in and
registration, the catalog with Open Library search and import, book pages with
shelf, progress, rating, and review actions, shelves and reading progress,
reviews, reports, authors and categories, and settings. No page uses mock
data. The manual UI checklist with screenshots is in
[`../docs/evidence/ui-checklist.md`](../docs/evidence/ui-checklist.md).

## Checks

```bash
npm run lint
npm run typecheck
npm test          # Vitest unit tests (src/**/*.test.ts); fetch is mocked
npm run build
```
