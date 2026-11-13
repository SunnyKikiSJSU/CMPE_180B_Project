# Book Metaverse Frontend

Next.js, TypeScript, Tailwind CSS, and TanStack Query frontend for the Book Metaverse Portal.

## Local development

```bash
cp .env.local.example .env.local
npm install
npm run dev
```

Open http://localhost:3000.

The browser calls `/api/...` on the page's own origin. `next.config.ts` forwards
those requests to the Go API at `API_PROXY_TARGET` (default
`http://localhost:8080`), so there is no CORS and no API URL in the browser
bundle. The target is read when `next dev` or `next build` starts.

## Status

The UI is complete as a visual shell: 11 routes, the public home, and the
workspace layout. It is not connected to the API yet. Pages show static or empty
data, login writes a demo value to `localStorage`, and the home carousel uses
`src/data/mock-data.ts`.

Integration work (API client, cookie session, Google Books search and import,
book detail, shelves, progress, reviews, reports) is planned in
[`../FRONTEND_PLAN.md`](../FRONTEND_PLAN.md).

## Checks

```bash
npm run lint
npx tsc --noEmit
npm run build
```
