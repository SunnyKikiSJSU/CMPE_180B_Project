# Frontend Implementation Plan — Book Metaverse Portal

Companion to [`BACKEND_PLAN.md`](BACKEND_PLAN.md). The backend contract changed on
2026-11-10 to route Google Books through the API (`GET /api/external/books`,
`POST /api/books/import`). This document records what the existing Next.js app
already has, what must change because of that design, and the order to do it in.

- **Contract source:** [`docs/api/openapi.yaml`](docs/api/openapi.yaml) (v0.2.0)
- **Backend intent source:** [`BACKEND_PLAN.md`](BACKEND_PLAN.md) §10, §13 Phase 8.
  For frontend file names and phases, this document is authoritative;
  `BACKEND_PLAN.md` §10 points here.
- **Data model reference:** [`ER_diagram.md`](ER_diagram.md)
- **Last updated:** 2026-11-12

### Status legend

| Mark | Meaning |
| --- | --- |
| `[x]` | Present in source and verified by reading it; `npx tsc --noEmit` and `npm run lint` pass (2026-11-12) |
| `[~]` | Partially present; usable shell but missing behavior |
| `[ ]` | Not started |

No frontend unit or E2E test suite exists yet, so `[x]` does **not** mean
"verified by an automated test".

---

## 1. Current State (Verified)

The app is a **UI-first scaffold**: 11 routes and a complete visual system, with
almost no data flow. It renders static content and empty states.

### 1.1 Routes

| Route | File | Current state |
| --- | --- | --- |
| `/` | [`src/app/page.tsx`](frontend/src/app/page.tsx) | `[x]` Public home; hero, carousel, mood grid, footer. Featured books come from `mock-data.ts`. |
| `/explore` | [`src/app/explore/page.tsx`](frontend/src/app/explore/page.tsx) | `[~]` Public catalog shell, filter bar, empty state. No search. |
| `/login` | [`src/app/login/page.tsx`](frontend/src/app/login/page.tsx) | `[~]` Client form; writes `localStorage` demo session, no API call. |
| `/dashboard` | [`src/app/(workspace)/dashboard/page.tsx`](frontend/src/app/(workspace)/dashboard/page.tsx) | `[~]` Full `DashboardView` layout; all values are zero/static. |
| `/catalog` | [`src/app/(workspace)/catalog/page.tsx`](frontend/src/app/(workspace)/catalog/page.tsx) | `[~]` `SectionPage` placeholder. |
| `/authors` | [`src/app/(workspace)/authors/page.tsx`](frontend/src/app/(workspace)/authors/page.tsx) | `[~]` Placeholder. |
| `/shelves` | [`src/app/(workspace)/shelves/page.tsx`](frontend/src/app/(workspace)/shelves/page.tsx) | `[~]` Placeholder. |
| `/progress` | [`src/app/(workspace)/progress/page.tsx`](frontend/src/app/(workspace)/progress/page.tsx) | `[~]` Placeholder. |
| `/reviews` | [`src/app/(workspace)/reviews/page.tsx`](frontend/src/app/(workspace)/reviews/page.tsx) | `[~]` Placeholder. |
| `/reports` | [`src/app/(workspace)/reports/page.tsx`](frontend/src/app/(workspace)/reports/page.tsx) | `[~]` Placeholder. |
| `/settings` | [`src/app/(workspace)/settings/page.tsx`](frontend/src/app/(workspace)/settings/page.tsx) | `[~]` Placeholder. |

Missing routes: **`/register`**, **`/books/[bookId]`**, and author/category
detail routes.

### 1.2 Shared components

| Component | State | Notes |
| --- | --- | --- |
| `AppShell` | `[x]` | Sidebar, header search input, static profile block, workspace chrome. |
| `AuthBoundary` | `[~]` | Guards the workspace by reading `localStorage`; must become a session query. |
| `DashboardView` | `[~]` | Complete visual layout, hardcoded stats, date "Sunday · October 25, 2026", empty states. |
| `PublicHome`, `PublicNav` | `[x]` | Public shell and mobile drawer. |
| `BookCarousel` | `[~]` | Working client carousel over a `Book[]`, but it reads `books[active].coverUrl` with no guard, so an empty or still-loading list crashes it. Add an empty guard before feeding it API data. |
| `EmptyState`, `FilterBar`, `SectionPage` | `[x]` | Presentation only; `FilterBar` has no state or handlers. |
| `Navbar`, `HeroSection`, `Footer`, `ShelvesSection`, `DiscoverySection` | `[x]` | **Unused dead code**; not imported by any route. |
| `GoogleAttribution`, `BookCard`, `BookGrid`, `Pagination`, `LoadingState`, `ErrorState`, `QueryProvider` | `[ ]` | Missing. |

### 1.3 Data layer

| File | State |
| --- | --- |
| [`src/lib/api-client.ts`](frontend/src/lib/api-client.ts) | `[~]` Base URL from `NEXT_PUBLIC_API_URL`; sends JSON; throws a generic `Error` on non-2xx. No cookies, error envelope, `requestId`, 401 handling, or pagination headers. |
| [`src/services/book-service.ts`](frontend/src/services/book-service.ts) | `[~]` 4 calls: library search, get by ID, list reviews, rate. No Google search, import, `ensureLocalBook`, create/update. |
| [`src/services/reading-service.ts`](frontend/src/services/reading-service.ts) | `[~]` 4 calls: shelves, progress list, progress patch, reading report. No shelf CRUD, history, delete, or reports. |
| [`src/types/domain.ts`](frontend/src/types/domain.ts) | `[~]` Minimal types. No `ExternalBook`, `Book.source`/`infoLink`/`publishedDatePrecision`, summary fields, `Shelf.type`, `ReadingProgress.version`, user/profile/report types. |
| [`src/data/mock-data.ts`](frontend/src/data/mock-data.ts) | `[x]` Imported only by `PublicHome`, which passes `featuredBooks` to `BookCarousel` as props; must be removed from runtime. |

Installed but **unused**: `@tanstack/react-query`, `react-hook-form`,
`@hookform/resolvers`, `zod`. There is no query provider anywhere in the tree.

### 1.4 Tooling

| Item | State |
| --- | --- |
| Next.js 15.5, React 19, TypeScript 5.9, Tailwind 4 | `[x]` |
| `npm run lint` | `[x]` Passes |
| `npx tsc --noEmit` | `[x]` Passes |
| `typecheck` script | `[ ]` Missing from `package.json` |
| Unit test runner (Vitest/RTL/MSW) | `[ ]` Missing |
| Playwright | `[ ]` Missing |
| `web` Dockerfile / Compose service | `[ ]` Missing |
| `next.config.ts` image domains | `[ ]` Missing (only `reactStrictMode`) |

---

## 2. Decisions to Freeze

1. **Server state lives in TanStack Query.** No Redux/Zustand. Local form state
   stays in React Hook Form.
2. **Forms use React Hook Form + Zod**, matching the installed dependencies.
3. **The browser never calls Google.** All Google access goes through
   `GET /api/external/books` and `POST /api/books/import`. No Google API key in
   the frontend, never in a `NEXT_PUBLIC_*` variable.
4. **Google results are not local books.** An `ExternalBook` becomes actionable
   only after import. Every activity call takes the local `bookId` UUID, never
   `googleBooksId`.
5. **Import on first action.** A single `ensureLocalBook(external)` helper
   returns `external.bookId` when present, otherwise imports and returns the new
   local ID. All shelf/rating/review/progress actions go through it.
6. **Auth is cookie-based.** `credentials: "include"` on every request; the
   `bm_session` HttpOnly cookie is the session. Do not write tokens to
   `localStorage`. The `accessToken` in `AuthResponse` is for tests/CLI only.
7. **JSON conventions:** camelCase, UUID strings, RFC 3339 UTC. Lists are raw
   arrays plus `X-Total-Count` / `X-Page` / `X-Page-Size`.
8. **Errors** use `{ "error": { code, message, details, requestId } }`. The
   client surfaces `code` and `requestId`, not raw stack traces.
9. **Compatibility shapes must not break:** `Book.author` stays a string,
   `Review.body`, `Rating.value`, and `ReadingProgress.bookId/status/currentPage/totalPages`
   keep their names.
10. **Google attribution is mandatory** on Google search results and on
    Google-sourced book detail: a visible "Powered by Google" mark plus an
    `infoLink` to the Google Books page. Stored-library lists get a page-level
    notice:
    - Library search, shelf books, author and category books return
      `source`, so show the notice when any row has `source = "google"`.
    - Progress, history, reports, and "my reviews/ratings" rows are small cards
      without `source`. Show the notice on those pages always ("Some book
      information is provided by Google Books").
11. **Google-sourced catalog rows are read-only.** Hide metadata editing when
    `source = "google"`; the API returns `403` for `PATCH /api/books/{id}`.
12. **Dates render by precision.** `publishedDate` may be year/month/day
    precision; use `publishedDatePrecision` to display `2004`, `May 2004`, or
    `12 May 2004`.
13. **No mock data at runtime.** `mock-data.ts` is deleted (or moved to a test
    fixture) before the frontend milestone is done.
14. **Covers** are plain `<img>` until sizing/optimization is settled; if
    `next/image` is adopted, allow `books.google.com` and
    `*.googleusercontent.com` in `next.config.ts` and always render an
    attribution link for Google-sourced images.
15. **Public browsing is allowed; actions require login.** `/explore` calls the
    external search anonymously (the operation has no `security` block, and
    `BACKEND_PLAN.md` §4.2 lists Google search as public). Clicking
    shelf/rate/review redirects to `/login` with a return path.
16. **API data is fetched in the browser.** Pages that need API data are client
    components using TanStack Query, so the `bm_session` cookie travels with
    each request. Server components render layout and static content only. If
    a server-side public fetch is added later, it must use a separate
    server-only URL (for example `API_INTERNAL_URL=http://api:8080` in
    Compose), because `NEXT_PUBLIC_API_URL=http://localhost:8080` does not
    reach the API from inside the `web` container.
17. **One hostname for the web app and the API.** Open the app at
    `http://localhost:3000` with `NEXT_PUBLIC_API_URL=http://localhost:8080`.
    `localhost:3000` → `localhost:8080` is same-site, so the `SameSite=Lax`
    cookie is sent. `127.0.0.1:3000` → `localhost:8080` is cross-site and the
    cookie is dropped, even though CORS allows the origin.
18. **Google search has a small budget.** The backend allows 30 requests per
    minute per IP for `GET /api/external/books` and `POST /api/books/import`
    combined. Search runs on submit (or after a 3-character minimum and a
    700 ms pause), keeps the previous results while a new page loads, and never
    fires on every keystroke.

---

## 3. Target Architecture

```text
app/ route (server component: layout and static content only)
      │
      ▼
client page component ("use client")
      │
      ▼
feature components (books, shelves, progress, reviews, reports)
      │  useQuery / useMutation
      ▼
services/*.ts            one module per API group
      │
      ▼
lib/api-client.ts        envelope, credentials, headers, 401 event
      │
      ▼
Go API (/api/...)        Google is called only by the server
```

### 3.1 Directory additions

```text
frontend/src/
├── app/
│   ├── register/page.tsx                 # new
│   └── books/[bookId]/page.tsx           # new
├── components/
│   ├── GoogleAttribution.tsx             # new
│   ├── books/BookCard.tsx                # new
│   ├── books/BookGrid.tsx                # new
│   ├── books/CoverImage.tsx              # new
│   ├── ui/Pagination.tsx                 # new
│   ├── ui/LoadingState.tsx               # new
│   ├── ui/ErrorState.tsx                 # new
│   └── ui/ConfirmDialog.tsx              # new
├── providers/QueryProvider.tsx           # new (this name is canonical; BACKEND_PLAN.md and PLAN.md refer here)
├── hooks/
│   ├── use-auth.ts                       # me/logout helpers
│   ├── use-google-books.ts               # external search + import mutation
│   ├── use-books.ts
│   ├── use-shelves.ts
│   ├── use-progress.ts
│   ├── use-reviews.ts
│   └── use-reports.ts
├── lib/
│   ├── api-client.ts                     # rewrite
│   ├── query-keys.ts                     # new
│   ├── format.ts                         # dates by precision, ratings
│   └── errors.ts                         # ApiError + code → message map
├── services/
│   ├── auth-service.ts                   # new
│   ├── user-service.ts                   # new
│   ├── book-service.ts                   # extend
│   ├── shelf-service.ts                  # new
│   ├── reading-service.ts                # extend
│   ├── review-service.ts                 # new
│   ├── author-service.ts                 # new
│   └── report-service.ts                 # new
└── types/domain.ts                       # expand
```

### 3.2 Query key conventions

```ts
["auth", "me"]
["books", "search", filters]
["books", id]
["books", id, "reviews"]
["external", "google", filters]
["shelves"]
["shelves", id, "books"]
["progress"]
["progress", bookId]
["history"]
["reviews", "me"]           ["ratings", "me"]
["authors", filters]        ["categories", filters]
["reports", type, params]
```

Invalidate narrowly after mutations: rating → `["books", id]` and its reviews;
shelf mutation → `["shelves"]` and the affected `["shelves", id, "books"]`;
progress → `["progress"]`, `["books", id]`, `["history"]`; import → external
search keys plus `["books"]`.

### 3.3 Query defaults

- `staleTime: 30_000`
- Do not retry `4xx`; retry at most once on network/`5xx`
- `refetchOnWindowFocus: false` (course demo, avoids surprise calls)
- `retry: false` for mutations; show the error envelope instead

---

## 4. Contract Additions Caused by the Google Books Design

Full definitions: [`docs/api/openapi.yaml`](docs/api/openapi.yaml) §`/api/external/books`,
`/api/books/import`, `Book`, `ExternalBook`, `ErrorEnvelope`.

### 4.1 `GET /api/external/books`

| Parameter | Notes |
| --- | --- |
| `q`, `title`, `author`, `category`, `isbn` | At least one required. Mapped to Google operators server-side. |
| `sort` | `relevance` or `newest` only (not the library sort enum). |
| `page`, `pageSize` | `pageSize` capped at **40**, default 20. |

Response is `ExternalBook[]` plus `X-Total-Count` (Google's estimate, can shift
between pages). Errors: `400`, `429 RATE_LIMITED`, `502 UPSTREAM_UNAVAILABLE`,
`503 UPSTREAM_RATE_LIMITED` (with `Retry-After`).

### 4.2 `POST /api/books/import`

Body: `{ "googleBooksId": "..." }`. Returns `Book` with **`201`** when imported,
**`200`** when the volume or its ISBN-13 was already in the library. Both are
success for the UI. Never send client-supplied metadata.

### 4.3 Type changes

| Current | New / changed |
| --- | --- |
| `Book { id,title,author,authors?,categories?,coverUrl? }` | Add required `source: "seed" \| "google" \| "user"`; `infoLink?`, `description?`, `publishedDate?`, `publishedDatePrecision?`, `publisher?`, `pageCount?`, `isbn13?`, `averageRating`, `ratingCount`, `reviewCount`, `myProgress?`, `myRating?` |
| `Shelf { id,name,count }` | Add required `type: ShelfType`, optional `createdAt` |
| `ReadingProgress` | Add `percentComplete`, `version`, optional `book: BookSummary` (`id`, `title`, `author`, `coverUrl`, not a full `Book`); keep existing required fields; `totalPages` 0 means unknown |
| `Review` | Add `title?`, `user?`, `updatedAt?`; keep `body` |
| `Rating` | Add `summary?`, timestamps |
| `Author` / `Category` | Add `bookCount?` |
| — | New: `ExternalBook`, `User`, `UserProfile`, `UserStats`, `ReadingHistoryEvent`, `RatingSummary`, `ReviewRatingResult`, `ShelfBookResult`, `ErrorEnvelope`, report shapes |

### 4.4 Google row rules the UI must respect

- `source = "google"` → show attribution and `infoLink`; hide catalog edit.
- `source = "seed" | "user"` → no Google attribution.
- Google's own ratings are never stored; only local `averageRating` /
  `ratingCount` / `reviewCount` are displayed.
- An external result may already be local: `bookId` is non-null. Do **not**
  import again; use that ID.
- ISBN collisions can map several Google volumes to one local book; the import
  response is the single source of truth for which local ID to use.

### 4.5 Service-to-endpoint map

Every API call the UI makes, by service. Each row is an operation in
`openapi.yaml`; the response type is the schema named there. "List" means raw
array plus pagination headers (read with `apiClientPage`).

| Service | Method | Endpoint | Used by |
| --- | --- | --- | --- |
| `auth-service` | `register` | `POST /api/auth/register` | `/register` |
| | `login` | `POST /api/auth/login` | `/login` |
| | `logout` | `POST /api/auth/logout` (204) | `AppShell` |
| | `me` | `GET /api/auth/me` (unwrap `{ user }`; 401 → `null`) | `AuthBoundary`, `AppShell`, `PublicNav` |
| `user-service` | `getMe` | `GET /api/users/me` | `/settings`, dashboard greeting |
| | `updateMe` | `PATCH /api/users/me` | `/settings` profile form |
| | `changePassword` | `PUT /api/users/me/password` (204) | `/settings` password form |
| | `deleteMe` | `DELETE /api/users/me` body `{"confirm": true}` (204) | `/settings` |
| `book-service` | `searchGoogle` | `GET /api/external/books` (list) | `/catalog` Google tab, `/explore` |
| | `importGoogleBook` | `POST /api/books/import` (200 or 201) | `ensureLocalBook` |
| | `search` | `GET /api/books` (list) | `/catalog` library tab, `/` featured |
| | `getById` | `GET /api/books/{bookId}` | `/books/[bookId]` |
| | `create` | `POST /api/books` | "Add a book by hand" (optional) |
| | `update` | `PATCH /api/books/{bookId}` (403 for Google rows) | book detail edit (non-Google only) |
| | `getRatingSummary` | `GET /api/books/{bookId}/ratings` | book detail |
| | `rate` | `POST /api/books/{bookId}/ratings` | `RatingControl` |
| | `deleteRating` | `DELETE /api/books/{bookId}/ratings` (204) | `RatingControl` |
| `review-service` | `listForBook` | `GET /api/books/{bookId}/reviews` (list) | book detail |
| | `saveWithRating` | `PUT /api/books/{bookId}/review` | `ReviewForm` (create and edit) |
| | `update` | `PATCH /api/reviews/{reviewId}` | `/reviews` inline edit |
| | `remove` | `DELETE /api/reviews/{reviewId}` (204) | book detail, `/reviews` |
| | `listMine` | `GET /api/users/me/reviews` (list) | `/reviews` |
| | `listMyRatings` | `GET /api/users/me/ratings` (list) | `/reviews` |
| `shelf-service` | `list` | `GET /api/shelves` | `/shelves`, dashboard, shelf picker |
| | `create` | `POST /api/shelves` | `/shelves` |
| | `rename` | `PATCH /api/shelves/{shelfId}` (custom only) | `/shelves` |
| | `remove` | `DELETE /api/shelves/{shelfId}` (204, custom only) | `/shelves` |
| | `listBooks` | `GET /api/shelves/{shelfId}/books` (list) | `/shelves` |
| | `addBook` | `POST /api/shelves/{shelfId}/books` | shelf picker |
| | `moveBook` | `PATCH /api/shelves/{shelfId}/books/{bookId}` | `/shelves`, shelf picker |
| | `removeBook` | `DELETE /api/shelves/{shelfId}/books/{bookId}` (204) | `/shelves` |
| `reading-service` | `listProgress` | `GET /api/reading-progress` (list) | `/progress`, dashboard |
| | `getProgress` | `GET /api/reading-progress/{bookId}` | book detail (when not in `myProgress`) |
| | `updateProgress` | `PATCH /api/reading-progress/{bookId}` (send `version`) | `ProgressForm` |
| | `deleteProgress` | `DELETE /api/reading-progress/{bookId}` (204) | `/progress` |
| | `listHistory` | `GET /api/reading-history` | `/progress`, dashboard |
| `author-service` | `listAuthors` | `GET /api/authors` (list) | `/authors`, library filters |
| | `getAuthor` | `GET /api/authors/{authorId}` | author detail |
| | `listCategories` | `GET /api/categories` (list) | `/authors`, library filters |
| | `getCategory` | `GET /api/categories/{categoryId}` | category detail |
| `report-service` | `readingHistory` | `GET /api/reports/reading-history` | dashboard, `/reports` |
| | `userStats` | `GET /api/reports/user-stats` | dashboard, `/reports` |
| | `popular` | `GET /api/reports/popular` | `/reports` |
| | `topRated` | `GET /api/reports/top-rated` | `/reports` |
| | `recommendations` | `GET /api/reports/recommendations` | `/reports`, dashboard |
| | `authorRankings` | `GET /api/reports/authors` | `/reports` |
| | `categoryRankings` | `GET /api/reports/categories` | `/reports` |
| | `reviewActivity` | `GET /api/reports/review-activity` | `/reports` |
| | `queryCatalog` | `GET /api/reports/queries` | `/reports` "View SQL" |

Deliberately unused by the UI: `GET /` (service metadata), `/healthz`,
`/readyz`, `GET /api/search` (alias of `GET /api/books`), `PUT
/api/reading-progress/{bookId}` (full replace; the form uses `PATCH`), and
`POST /api/books/{bookId}/reviews` (the form uses the combined `PUT`). The
existing `bookService.getReviews` and `readingService.getShelves`/`getReport`
move into `review-service`, `shelf-service`, and `report-service`.

---

## 5. Implementation Detail

### 5.1 `lib/api-client.ts`

```ts
export class ApiError extends Error {
  status: number;
  code: string;       // VALIDATION_ERROR, UNAUTHENTICATED, UPSTREAM_RATE_LIMITED, ...
  details?: unknown;
  requestId?: string;
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

apiClient<T>(path, init?): Promise<T>          // JSON body when provided
apiClientPage<T>(path, init?): Promise<Page<T>> // reads X-Total-Count/X-Page/X-Page-Size
```

Requirements:

- `credentials: "include"` by default.
- Send `Content-Type: application/json` only when a body exists. `DELETE`
  may have a body: `DELETE /api/users/me` requires `{"confirm": true}`.
- **Empty responses:** `204` (logout, password change, every `DELETE`) returns
  `undefined` without calling `response.json()`. Any other 2xx with an empty
  body is treated the same way.
- On non-2xx, parse the envelope; if parsing fails, synthesize `INTERNAL` with
  `requestId` from the `X-Request-ID` response header if present.
- Read `Retry-After` on `429` and `503` and put it on `ApiError.retryAfter`
  (seconds). The backend exposes `Retry-After` and `X-Request-ID` through CORS
  (`BACKEND_PLAN.md` §6.2).
- **401 handling:** on `401`, dispatch a browser event
  (`auth:unauthorized`) that clears `["auth","me"]`; `AuthBoundary` then
  redirects to `/login?returnTo=...`. Do **not** dispatch the event for:
  - `POST /api/auth/login` and `POST /api/auth/register`: a 401 there means
    wrong credentials, shown on the form;
  - `GET /api/auth/me`: a 401 there is the normal "not signed in" answer on
    public pages.
- Accept an `AbortSignal`; use a 15 s timeout.
- Never log tokens or cookies.

`ApiError` also carries `retryAfter?: number`.

### 5.2 Auth

- `auth-service.ts`: `register`, `login`, `logout`, `me`. `me` unwraps the
  response: the API returns `{ "user": User }`, the service returns `User`.
  A 401 from `me` resolves to `null` (signed out), not an error.
- `QueryProvider` wraps the root layout.
- `AuthBoundary` calls `useQuery({ queryKey: ["auth","me"] })`; while loading
  render a skeleton; when the result is `null` redirect to `/login` and preserve
  the path.
- `login/page.tsx`: RHF + Zod (`email`, `password`), calls `login`, invalidates
  `["auth","me"]`, then routes to `returnTo ?? "/dashboard"`; shows field errors
  and envelope errors.
- `register/page.tsx`: `email`, `displayName`, `password` (min 8), confirm
  password. Registration already starts a session (`201` with the cookie and
  `{ user, accessToken }`), so do not call `login` afterwards: write `user` into
  `["auth","me"]` and route to `/dashboard`. Show `409 CONFLICT` as "This email
  is already registered".
- Password change (`/settings`): a wrong current password returns
  `400 VALIDATION_ERROR` with `details.field = "currentPassword"`; show it on
  that field. It never signs the user out.
- Delete account (`/settings`): `DELETE /api/users/me` with body
  `{"confirm": true}` after a `ConfirmDialog`; on `204` clear the query cache
  and route to `/`.
- `AppShell`: profile name/avatar from `me`, logout action, active nav state,
  header search submits to `/catalog?q=...`.

### 5.3 Catalog and Google Books flow

`catalog/page.tsx` becomes a client page with two tabs:

1. **Google Books (default)** — calls `searchGoogle(filters)`.
   - Search input that runs on submit (Enter or the search button); optional
     title/author/isbn/category fields. If live search is wanted, require 3
     characters and a 700 ms pause. Use `placeholderData: keepPreviousData` so
     paging does not blank the grid. See decision 18 for the request budget.
   - Result card: cover, title, subtitle, authors, categories, year, publisher,
     page count, attribution link.
   - CTA label depends on `bookId`:
     - `bookId` present → **"Open in library"** (navigate to `/books/{bookId}`).
     - `bookId` null → **"Add to shelf"** (import first).
   - On `UPSTREAM_UNAVAILABLE` / `UPSTREAM_RATE_LIMITED`, show a banner: Google
     is unavailable, with **"Search the library instead"** switching tabs.
   - On `429 RATE_LIMITED` (this browser's per-IP budget), show "Too many
     searches, try again in N seconds" using `ApiError.retryAfter`, and offer
     the same library fallback.
   - Respect `pageSize ≤ 40`; show Google's estimated total, not an exact count.
2. **In the library** — calls `searchBooks(filters)`.
   - Filters: `q`, `authorId`, `categoryId`, `sort`
     (`relevance|newest|rating|title|popular`), page/pageSize.
   - Rows are normal `Book` objects; show local rating summary.
   - Page-level Google attribution notice if any result has `source = "google"`.

Public `/explore` gets a real Google search using the same result component but
no shelf/rate CTAs; instead, "Sign in to save" → `/login?returnTo=/explore`.

### 5.4 Import-on-action

```ts
async function ensureLocalBook(external: ExternalBook): Promise<string> {
  if (external.bookId) return external.bookId;
  const book = await bookService.importGoogleBook(external.googleBooksId);
  queryClient.invalidateQueries({ queryKey: ["books"] });
  return book.id;
}
```

Use it only where the user acts on an `ExternalBook` (a Google search result):
"Add to shelf", "Rate", "Start reading", "Open in library". Once the book is
local, every later action (book detail, shelves, progress, reviews) already has
the `bookId` and calls the activity endpoints directly. A Google ID must never
reach an activity endpoint.

After an import, also invalidate `["external", "google"]` so the result shows
its new `bookId`.

### 5.5 Book detail (`/books/[bookId]`)

- Fetch `GET /api/books/{bookId}` (authenticated, so `myProgress`/`myRating`
  arrive).
- Show title, cover, primary author plus `authors[]`, `categories[]`,
  description, publisher, page count, ISBN, rating summary.
- Date: format by `publishedDatePrecision`.
- `source = "google"`: Google attribution + `infoLink`; hide edit controls.
- Actions: add/move shelf, update progress, set rating, write/edit review.
- **Review form saves rating and review together** with
  `PUT /api/books/{bookId}/review` (`{ rating?, title?, body }` →
  `{ review, rating, summary }`). This is the backend's primary transaction
  demonstration (`BACKEND_PLAN.md` §6.6), so the UI must use it rather than two
  separate calls. The standalone star control may still use
  `POST /api/books/{bookId}/ratings` for a rating without text.
- **Unknown page count:** Google volumes often have no `pageCount`. The API then
  returns `totalPages: 0`. The progress form hides the page field, asks for a
  percentage instead (`percentComplete`), and never divides by `totalPages`.
- Handle `404` (book missing or not visible) and `401`.

### 5.6 Page-by-page changes

| Page | Change |
| --- | --- |
| `/` | Replace `featuredBooks` with a public library query (`GET /api/books?sort=popular&pageSize=3`). Render `BookCarousel` only when the list is non-empty; show the static hero while loading, on error, or when the list is empty. Remove `mock-data` import. |
| `/explore` | Real external search; login gate on actions; empty/loading/error states. |
| `/catalog` | Two tabs as in §5.3; pagination; upstream fallback. |
| `/dashboard` | `userStatsReport` + `readingReport` + `listReadingProgress` + `listShelves`; real values, greeting from `me`, remove the hardcoded date. |
| `/shelves` | List/create/rename/delete shelves; open a shelf's books; add/move/remove books. Disable rename/delete for non-`custom` shelves. |
| `/progress` | List progress, update page/status; send `version`; on `409 STALE_VERSION` show "This book changed elsewhere" and refresh. When `totalPages` is 0, edit `percentComplete` instead of the page. |
| `/reviews` | `listMyReviews` + `listMyRatings`; edit/delete; link to book detail. |
| `/reports` | Tabs for reading history, user stats, popular, top-rated, recommendations, author/category rankings, review activity; query catalog viewer for `GET /api/reports/queries`. |
| `/authors` | Authors and categories lists with counts; navigate to detail. |
| `/settings` | Profile form (displayName, avatarUrl, bio, favoriteGenre), password change (wrong current password is a field error, §5.2), delete account with `ConfirmDialog` and body `{"confirm": true}`. |
| `/login` | Real auth (see §5.2). |
| `/register` | New page (see §5.2). |
| `AppShell` | Dynamic profile, logout, active link, header search navigation. |

### 5.7 Components to build

- `GoogleAttribution` — small "Powered by Google" mark + `infoLink`; two modes:
  per-item (results, detail) and page-level notice (library lists).
- `BookCard` / `BookGrid` — shared between Google results and library results;
  accepts a `variant` for CTA and attribution.
- `CoverImage` — renders `coverUrl`; placeholder when null/broken; never blocks
  layout.
- `Pagination` — page, pageSize, total; emits changes; hides when total unknown.
- `LoadingState`, `ErrorState` — consistent skeletons and envelope-aware errors
  (`requestId` shown small).
- `ConfirmDialog` — delete shelf, review, account.
- Forms: `LoginForm`, `RegisterForm`, `ShelfForm`, `ProgressForm`,
  `ReviewForm` (rating + review through `PUT /api/books/{bookId}/review`),
  `RatingControl`, `ProfileForm`, `PasswordForm` with Zod schemas.

### 5.8 UX rules

- Every list page has distinct **loading**, **empty**, **error**, and
  **unauthorized** states.
- Optimistic updates for rating and shelf membership; roll back on error.
- Disable submit buttons while mutations are pending.
- Show `STALE_VERSION` as a recoverable conflict, never a generic failure.
- Do not show raw SQL errors, cookies, or tokens.
- Keyboard-accessible dialogs, focus rings, and labeled inputs.
- Empty states should keep their current copy unless the API supplies better
  context.

---

## 6. Testing Plan

### 6.1 Tooling to add

- `typecheck` script: `tsc --noEmit`
- Vitest + React Testing Library + jsdom + MSW (mock Service Worker)
- Playwright for E2E
- `test`, `test:watch`, `test:e2e` scripts

### 6.2 Unit / integration (MSW)

- `api-client`: envelope parsing, `requestId`, credentials, header parsing,
  401 event, timeout.
- `ensureLocalBook`: returns existing `bookId`; imports when null; never double
  imports.
- Date precision formatter; rating formatter.
- Auth: login success/failure, register validation, `me` 401 redirect.
- Catalog: Google error → library fallback; pagination.
- Forms: Zod messages, stale-version conflict, delete confirmations.

### 6.3 Playwright flows (from BACKEND_PLAN §9.3)

1. Register, log in, dashboard loads.
2. Search Google (stubbed), import a result, shelve it; search library by
   title/author/category.
3. Book detail shows authors, categories, summaries.
4. Add to shelf and move between status shelves.
5. Update progress; confirm the Finished transition.
6. Create, edit, delete a review.
7. Rate a book and see the summary update.
8. Open each report with non-empty data.

CI gate: `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`,
`npm run test:e2e`.

---

## 7. Phased Execution Checklist

### Phase F0 — Baseline cleanup and tooling

- [x] Next.js 15 App Router, React 19, TypeScript, Tailwind 4
- [x] 11 routes and shared layout
- [x] Visual system in `globals.css`; responsive public drawer
- [x] `npx tsc --noEmit` and `npm run lint` pass (2026-11-12)
- [ ] Add `typecheck` script
- [ ] Add Vitest/RTL/MSW and Playwright dependencies
- [ ] Delete unused `Navbar`, `HeroSection`, `Footer`, `ShelvesSection`,
      `DiscoverySection` (or adopt them intentionally)
- [ ] Remove `mock-data.ts` from runtime; keep only as a test fixture if useful
- [ ] Add `next.config.ts` `images.remotePatterns` if `next/image` is adopted
- [ ] Refresh `frontend/README.md` (currently says the UI is a "starter")

**Exit:** tooling scripts exist; no dead runtime mock import.

### Phase F1 — Contract, types, and API client

- [x] Base `api-client` with `NEXT_PUBLIC_API_URL` and JSON
- [x] 8 initial service methods
- [~] Baseline domain types
- [ ] `ApiError` + `ErrorEnvelope` parsing; `requestId` and `retryAfter` surfaced
- [ ] `credentials: "include"`, timeout, 401 event with the login/register/`me`
      exclusions
- [ ] `204` / empty-body responses return `undefined`
- [ ] `apiClientPage` reading pagination headers
- [ ] Expand `Book`, `Shelf`, `ReadingProgress`, `Review`, `Rating`,
      `Author`, `Category`; add `ExternalBook`, `User`, `UserProfile`,
      `UserStats`, `ReadingHistoryEvent`, report types
- [ ] `query-keys.ts`, `format.ts`, `errors.ts`
- [ ] Unit tests for `api-client`

**Exit:** every OpenAPI schema used by the UI has a TypeScript type; compile and
client tests pass.

### Phase F2 — Providers and authentication

- [x] Login page visual shell
- [x] `AuthBoundary` guard shell
- [ ] `QueryProvider` in the root layout
- [ ] `auth-service.ts` (register/login/logout/me)
- [ ] Login page: RHF + Zod, API errors, `returnTo`
- [ ] `/register` page
- [ ] `AuthBoundary` via `["auth","me"]`; remove `localStorage`
- [ ] `AppShell` profile, logout, active nav, header search
- [ ] 401 redirect handling

**Exit:** demo login works against the Go API; no `localStorage` session.

### Phase F3 — Catalog and Google Books

- [x] Catalog route shell, filter bar, empty state
- [x] Public explore shell
- [ ] `searchGoogle` service + external query hook
- [ ] `importGoogleBook` + `ensureLocalBook`
- [ ] Two-tab catalog (Google default, Library)
- [ ] Google result cards with attribution + `infoLink`
- [ ] Library search with `q`, `authorId`, `categoryId`, `sort`, pagination
- [ ] Upstream error fallback ("Search the library instead")
- [ ] Public explore search + login gate on actions
- [ ] `Pagination`, `BookCard`, `BookGrid`, `CoverImage`

**Exit:** a Google result can be found, imported idempotently, and shelved; the
library tab finds the imported book.

### Phase F4 — Book detail and activity

- [ ] `/books/[bookId]` route
- [ ] Detail fetch with authors, categories, summary, description
- [ ] Attribution, `infoLink`, and date-by-precision rendering
- [ ] Shelf picker (import-on-action)
- [ ] Progress form (page + status + version)
- [ ] Rating control (optimistic, summary refresh)
- [ ] Review form via `PUT /api/books/{bookId}/review` (rating + review in one
      transaction); edit and delete
- [ ] Percentage-only progress when `totalPages` is 0

**Exit:** every activity promised by the report/readme can be performed from the
detail page.

### Phase F5 — Shelves and progress

- [x] Page shells and empty states
- [ ] Shelf list/create/rename/delete
- [ ] Shelf books list, add/move/remove
- [ ] Status-shelf protections in the UI
- [ ] Progress list/update; `STALE_VERSION` recovery
- [ ] Reading history list

**Exit:** shelf and progress flows pass manually against the API.

### Phase F6 — Reviews, ratings, reports, authors, settings

- [x] Page shells
- [ ] My reviews and ratings; edit/delete
- [ ] Reports tabs and data tables
- [ ] Report query catalog viewer (SQL shown as text)
- [ ] Authors/categories listings with counts
- [ ] Author/category detail navigation
- [ ] Settings: profile, password, delete account

**Exit:** all sidebar routes render real data, none use placeholders.

### Phase F7 — Public pages and polish

- [x] Public home hero, carousel, mood grid, footer
- [x] Public mobile drawer
- [ ] Featured books from the API instead of `mock-data`; `BookCarousel` empty
      guard
- [ ] Page-level Google attribution notices
- [ ] Loading/empty/error/unauthorized states on every route
- [ ] Optimistic updates and rollback
- [ ] Accessibility pass (focus, labels, dialogs, contrast)
- [ ] Responsive QA at 360/768/1280 px
- [ ] Empty-state copy consistency

**Exit:** public and workspace pages feel complete and fail gracefully.

### Phase F8 — Quality gates and deployment

- [x] Lint and type-check
- [ ] Vitest/RTL/MSW unit suite
- [ ] Playwright 8 flows
- [ ] CI job (lint → typecheck → unit → build → e2e)
- [ ] `frontend/Dockerfile` + Compose `web` service
- [ ] Production `NEXT_PUBLIC_API_URL` and CORS/cookie verification
- [ ] README/DEVELOPMENT updates

**Exit:** CI is green; the app runs from Compose against the API.

### 7.1 Backend dependencies

A frontend phase can be built against MSW mocks of `openapi.yaml` at any time,
but it is only **done** when it passes against the real API. That needs the
backend phase on the right.

| Frontend phase | Needs backend (`BACKEND_PLAN.md` §13) | Endpoints first used |
| --- | --- | --- |
| F0 Baseline | none | none |
| F1 Contract and client | none (mocks only) | none |
| F2 Auth | Phase 2 (CORS, cookies, error envelope) and Phase 3 (auth) | `/api/auth/*`, `/api/users/me` |
| F3 Catalog and Google | Phase 4 | `/api/books`, `/api/external/books`, `/api/books/import`, `/api/authors`, `/api/categories` |
| F4 Book detail and activity | Phases 4, 5, 6 | `/api/books/{id}`, `/api/shelves/*`, `/api/reading-progress/*`, `/api/books/{id}/ratings`, `/api/books/{id}/review(s)` |
| F5 Shelves and progress | Phase 5 | `/api/shelves*`, `/api/reading-progress*`, `/api/reading-history` |
| F6 Reviews, reports, settings | Phases 6 and 7 (and 3 for settings) | `/api/users/me/*`, `/api/reviews/*`, `/api/reports/*` |
| F7 Polish | as above | — |
| F8 Quality and deployment | Phase 10 (Compose `web`, deployed API) | all |

Backend Phase 8 ("Frontend integration") is this plan's phases F2–F7.

---

## 8. Done vs. To Do — Summary

### 8.1 Reusable as-is (little or no change)

- App Router project, TypeScript, Tailwind 4, global styles.
- Public shell: `PublicHome`, `PublicNav`, mobile drawer, hero, footer.
- `BookCarousel` (already accepts `Book[]`).
- `EmptyState`, `SectionPage`, `FilterBar` visuals.
- `DashboardView` layout (data-free, but structurally complete).
- `AppShell` layout (needs dynamic profile/nav only).
- Base `api-client` URL handling and the 8 existing service calls (still valid).

### 8.2 Implemented but must change

| Area | Why | Required change |
| --- | --- | --- |
| `login/page.tsx` | Fake session | Real auth service + RHF/Zod; remove `localStorage` |
| `AuthBoundary.tsx` | Fake session | `useQuery(["auth","me"])`; redirect on 401 |
| `api-client.ts` | Contract v0.2.0 | Cookies, error envelope, `requestId`, pagination headers, 401 event |
| `types/domain.ts` | Google + full contract | Add `source`, `infoLink`, precision, `ExternalBook`, summaries, user/report types |
| `book-service.ts` | Google design | Add `searchGoogle`, `importGoogleBook`, `ensureLocalBook`; keep library search; add detail/create/update/ratings summary |
| `reading-service.ts` | Full contract | Add history, delete, shelf CRUD (or split into `shelf-service`) |
| `catalog/page.tsx` | Google design | Two tabs, search, pagination, states, upstream fallback |
| `explore/page.tsx` | Google design | Real external search; login gate on actions |
| `PublicHome.tsx` | Mock data | Featured query; remove `mock-data` import |
| `AppShell.tsx` | Auth | Dynamic profile, logout, active nav, header search |
| `DashboardView.tsx` | Static | Real stats, progress, history, shelves; remove hardcoded date |
| Workspace pages | Placeholders | Replace `SectionPage` with feature pages |
| `FilterBar` | Presentation only | Controlled state, real filters |

### 8.3 Missing (new work)

- `/register`, `/books/[bookId]`, author/category detail routes.
- `QueryProvider`, TanStack Query hooks, query keys.
- `auth-service`, `user-service`, `shelf-service`, `review-service`,
  `author-service`, `report-service`.
- `GoogleAttribution`, `BookCard`, `BookGrid`, `CoverImage`, `Pagination`,
  `LoadingState`, `ErrorState`, `ConfirmDialog`.
- All forms and Zod schemas.
- Test suite (Vitest/RTL/MSW, Playwright), `typecheck` script, CI.
- `frontend/Dockerfile` and Compose `web` service.
- Google image domain config, if `next/image` is used.

---

## 9. Risks and Open Decisions

### 9.1 Risks

| Risk | Mitigation |
| --- | --- |
| Passing `googleBooksId` to an activity endpoint | Activity APIs accept only local UUIDs; all actions go through `ensureLocalBook` |
| Double import or duplicate volume | Import is idempotent (`200`/`201`); treat both as success; trust returned `bookId` |
| Google outage or quota exhaustion | `UPSTREAM_*` → banner + library tab; respect `Retry-After`; pageSize ≤ 40 |
| Google total count is an estimate | Label pagination as approximate on the Google tab |
| Hotlinked covers break or are blocked | `CoverImage` fallback; keep attribution and `infoLink`; no S3 dependency |
| Attribution missed on stored lists | Page-level notice wherever `source = "google"` can appear; per-item on search/detail |
| Google rows edited by mistake | Hide edit when `source = "google"`; API returns `403` |
| Cookie/CORS misconfiguration | `credentials: "include"`, `ALLOWED_ORIGINS`, verify both ports (3000 → 8080) |
| `401` mid-session | Global 401 event; query cache clear; redirect with `returnTo` |
| Stale progress write | Send `version`; on `409 STALE_VERSION` prompt refresh, never retry blindly. (The database raises SQLSTATE `BM001`; the browser only ever sees `STALE_VERSION`.) |
| Google search budget (30/min per IP) exhausted by typing or paging | Search on submit (decision 18); keep previous data; show `429 RATE_LIMITED` with `retryAfter` and offer library search |
| Session cookie silently missing | Same hostname for app and API (decision 17); browser-side data fetching (decision 16) |
| Global 401 handler fires on a wrong password | Exclude login, register, and `me` from the 401 event (§5.1); wrong current password on password change is `400`, not `401` |
| `response.json()` on a `204` | `api-client` returns `undefined` for empty responses (§5.1) |
| Empty featured list crashes `BookCarousel` | Render the carousel only for a non-empty list (§5.6) |
| Book without page count | `totalPages: 0`; progress form switches to percentage (§5.5) |
| Fake seed dates | Remove the dashboard's hardcoded date; render timestamps from data |
| Dead components drift | Delete unused files in F0, or adopt them intentionally |

### 9.2 Open decisions

Settled (2026-11-12):

- **Library tab:** always visible next to the Google tab, Google first.
  `BACKEND_PLAN.md` §16, open decision 5, keeps the tab.
- **Anonymous Google search:** allowed. `GET /api/external/books` has no
  `security` block in `openapi.yaml`, and `BACKEND_PLAN.md` §4.2 lists it as a
  public read endpoint. Import (`POST /api/books/import`) requires login.

Still open:

1. Author/category detail as routes (`/authors/[authorId]`) or as drawers on
   `/authors`?
2. `next/image` for Google covers, or plain `<img>` for the MVP?
3. Optimistic updates for rating/shelf only, or also progress?
4. Full report suite on one page with tabs, or separate report routes?
5. Deployment: same domain for web and API (simplest cookies) or cross-site
   (needs `SameSite=None; Secure` and a CSRF strategy). Shared with
   `BACKEND_PLAN.md` §16, open decision 2.

---

## 10. Requirement Traceability

| Requirement | Frontend evidence |
| --- | --- |
| Google Books discovery through the server, key never in the browser | Catalog/explore use `/api/external/books`; no Google URL or key in `src/` |
| Google content attribution | `GoogleAttribution` on every Google result/detail; `infoLink` |
| Local book identity for activity | `ensureLocalBook`; all activity calls use local UUIDs |
| Explicit SQL-backed features demonstrated in UI | Catalog search, shelf joins, progress, reviews, reports all render API data |
| Optimistic concurrency | Progress form sends `version`; stale-version recovery UI |
| Transaction/conflict behavior visible | Duplicate shelf/review (`409 CONFLICT`) and stale progress (`409 STALE_VERSION`) messages |
| Report evidence | Reports page renders every report endpoint and the query catalog |
| Auth/session | Cookie session, login/register/logout, guarded workspace, 401 redirect |
| No mock data | `mock-data.ts` removed from runtime; all pages query the API |

---

## 11. Definition of Done

The frontend milestone is complete when:

1. `npx tsc --noEmit`, `npm run lint`, and `npm run build` pass.
2. No route imports `mock-data.ts`; no code reads or writes
   `book-metaverse-session`.
3. Login/register/logout/me work against the Go API with the cookie session.
4. Catalog offers Google search and library search; a Google result can be
   imported once and then shelved, rated, reviewed, and tracked.
5. Every Google-derived surface shows attribution and `infoLink`; Google rows
   cannot be edited.
6. Every page has loading, empty, error, and unauthorized states.
7. The eight Playwright flows pass.
8. The app runs from the Compose `web` service against the API.
