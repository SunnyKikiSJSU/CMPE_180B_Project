# Book Metaverse Frontend

Next.js, TypeScript, Tailwind CSS, and TanStack Query frontend for the Book Metaverse Portal.

## Local development

```bash
cp .env.local.example .env.local
npm install
npm run dev
```

Open http://localhost:3000.

The API base URL is configured with `NEXT_PUBLIC_API_URL`. The current home page is a starter UI; catalog, authentication, shelves, reading progress, ratings, and reviews will be added as the backend contract is implemented.
