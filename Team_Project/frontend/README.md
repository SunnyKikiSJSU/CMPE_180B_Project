# Book Metaverse Frontend

Next.js, TypeScript, Tailwind CSS, and TanStack Query frontend for the Book Metaverse Portal.

## Local development

```bash
cp .env.local.example .env.local
npm install
npm run dev
```

Open http://localhost:3000.

The API base URL is configured with `NEXT_PUBLIC_API_URL` (default
`http://localhost:8080`). Open the app at `http://localhost:3000`, not
`127.0.0.1:3000`: the session cookie is only sent when the app and the API use
the same hostname.

## Status

The UI is complete as a visual shell: 11 routes, the public home, and the
workspace layout. It is not connected to the API yet. Pages show static or empty
data, login writes a demo value to `localStorage`, and the home carousel uses
`src/data/mock-data.ts`.

Integration work (API client, cookie session, Google Books search and import,
book detail, shelves, progress, reviews, reports, tests) is planned in
[`../FRONTEND_PLAN.md`](../FRONTEND_PLAN.md).

## Checks

```bash
npm run lint
npx tsc --noEmit
npm run build
```
