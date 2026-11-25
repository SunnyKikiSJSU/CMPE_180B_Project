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

## Status

Connected to the API so far (FRONTEND_PLAN.md phases F0-F2):

- **Sign-in, registration, and sign-out** against the Go API. The session is
  the `bm_session` HttpOnly cookie; nothing is stored in `localStorage`.
- **Workspace guard:** every page under `(workspace)` checks
  `GET /api/auth/me`. Signed-out visitors go to `/login?returnTo=<page>` and
  come back after signing in; if the API is unreachable, a retry card is shown.
- **App shell:** the signed-in user's name, the current page in the sidebar,
  sign-out, and the header search (to `/catalog?q=`).
- **Foundation for the other pages:** `lib/api-client.ts` (error envelope as
  `ApiError`, `204` handling, pagination headers), `types/domain.ts` (every
  schema the UI uses), `lib/query-keys.ts`, `lib/format.ts`, and the TanStack
  Query provider (no retry on 4xx; a query's 401 re-checks the session).

The public home carousel shows popular books from the API; no page uses mock
data. The remaining work (reviews, reports, authors, settings, and polish) is
planned in [`../FRONTEND_PLAN.md`](../FRONTEND_PLAN.md).

## Checks

```bash
npm run lint
npm run typecheck
npm test          # Vitest unit tests (src/**/*.test.ts); fetch is mocked
npm run build
```
