# Manual UI checklist (FRONTEND_PLAN.md §6.3)

Run on 2026-11-25 against the **Docker Compose stack** (`./scripts/dev.sh local`:
PostgreSQL 18 with migrations `000001`-`000008` and all seeds, the Go API
container, and the Next.js `web` container built from `frontend/Dockerfile`),
in Chrome through the `web` container only. The API was reached through the
web app's `/api/*` proxy, as it will be on EC2. A new account was registered
for the run; Open Library was live.

| # | Flow | Result | Screenshots |
| --- | --- | --- | --- |
| 1 | Register, sign in, dashboard loads | Pass. "Welcome back, Demo Reader."; four empty status shelves | [register](ui/01a-register.jpg), [dashboard](ui/01b-dashboard-new-user.jpg) |
| 2 | Search Open Library, import a result, shelve it; search the library by title, author, and category | Pass. *The Left Hand of Darkness* imported (`POST /api/books/import`) and shelved as Want to read; library search: title 1 book, "tolkien" 4 books, category Poetry 13 books | [Open Library](ui/02a-openlibrary-search.jpg), [by title](ui/02b-library-search-title.jpg), [by category](ui/02c-library-search-category.jpg) |
| 3 | Book detail shows authors, categories, summary | Pass. Author link, 5 categories, Open Library description, rating summary, attribution link | [book detail](ui/03-book-detail-shelved.jpg) |
| 4 | Add to a shelf and move between status shelves | Pass. Want to read → Currently reading from `/shelves`; counts 0 / 1 | [shelves](ui/04-shelves-moved.jpg) |
| 5 | Update progress; confirm the Finished transition | Pass. Page 152 of 304, then 304 of 304: status became Finished (database trigger); history shows "Finished the book" | [halfway](ui/05a-progress-halfway.jpg), [finished](ui/05b-progress-finished.jpg) |
| 6 | Create, edit, delete a review | Pass. Posted (summary "1 review"), edited, deleted ("0 reviews") | [created](ui/06a-review-created.jpg), [edited](ui/06b-review-edited.jpg) |
| 7 | Rate a book and see the summary update | Pass. "No ratings yet" → "★ 5.0 (1 rating)" | [rating](ui/07-rating-summary.jpg) |
| 8 | Open each report with non-empty data | Pass. All seven tabs loaded with data and no errors | [my reading](ui/08a-report-my-reading.jpg), [popular](ui/08b-report-popular.jpg), [top rated](ui/08c-report-top-rated.jpg), [authors](ui/08d-report-authors.jpg), [categories](ui/08e-report-categories.jpg), [activity](ui/08f-report-activity.jpg), [SQL](ui/08g-report-sql.jpg) |

After the run, the dashboard showed 1 book read, 304 pages, average rating
5.00, and the history timeline ([dashboard](ui/09-dashboard-after.jpg)).

## Session cookie through the proxy

`POST /api/auth/login` through the web container (port 3100 in this run)
returned:

```text
set-cookie: bm_session=…; Path=/; Max-Age=3599; HttpOnly; SameSite=Lax
```

`HttpOnly` and `SameSite=Lax` as designed, and no `Secure` flag because
`COOKIE_SECURE=false` for plain HTTP. The same check repeats on the EC2
address in the deployment phase (and with `COOKIE_SECURE=true` once HTTPS is
in front).

## Checks before the run

`npm run lint`, `npm run typecheck`, `npm test` (51 tests), `npm run build`;
`go test ./...` and `go test -tags integration ./internal/integration/`.
