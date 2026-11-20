# Query catalog (q01-q10)

The ten evidence queries from `BACKEND_PLAN.md` §5.4. Each is a named query in
`database/queries/`, compiled by sqlc, and run by the API with bound
parameters only. `GET /api/reports/queries` returns the same list with the
exact SQL text the API runs (taken from the generated sqlc code, so it cannot
drift from `database/queries/`). Every report response carries an
`X-Query-Id` header with its ID.

| ID | Purpose | Named queries (file) | Endpoint | Parameters | SQL features |
| --- | --- | --- | --- | --- | --- |
| q01 | Library search | `SearchBooks` (`books.sql`) | `GET /api/books` | `q`, `title`, `authorId`, `categoryId`, `sort`, `page`, `pageSize` | CTE, `EXISTS` over the author/category links, full-text search (`websearch_to_tsquery`, `ts_rank`), escaped `ILIKE` served by the `pg_trgm` GIN indexes, `json_agg` |
| q02 | Book detail | `GetBookByID` (`books.sql`) | `GET /api/books/{bookId}` | `bookId` | Outer join to the rating summary, `json_agg` subqueries for authors and categories |
| q03 | Shelf contents | `ListShelfBooks` (`shelves.sql`) | `GET /api/shelves/{shelfId}/books` | `shelfId`, `sort`, paging | Inner and outer joins, correlated `json_agg` and `max()` subqueries (latest activity), `count(*) OVER ()` for the total |
| q04 | Average ratings by book, category, author | `TopRatedBooks`, `RatingSummaryByCategory`, `RatingSummaryByAuthor` (`reports.sql`) | `GET /api/reports/top-rated` | `minRatings` (0-1000, default 1), `limit` (1-100, default 10) | `GROUP BY`, `HAVING`, aggregates |
| q05 | Popular books | `PopularBooks` (`reports.sql`) | `GET /api/reports/popular` | `days` (1-365, default 30), `endDate` (default today UTC), `limit` | CTEs that pre-aggregate each activity table (no join fan-out), weighted score |
| q06 | Recommendations | `RecommendedBooks` (`reports.sql`) | `GET /api/reports/recommendations` | `limit` | CTEs, `UNION`, `NOT EXISTS`, weighted ranking |
| q07 | Reading statistics | `UserReadingStats`, `ReadingReportSummary`, `FavoriteCategories`, `RecentReadingActivity` (`reports.sql`) | `GET /api/reports/user-stats`, `GET /api/reports/reading-history` | caller | CTEs per status, `CROSS JOIN` of one-row aggregates, `SUM`, `AVG`, `UNION` |
| q08 | Author and category rankings | `AuthorRankings`, `CategoryRankings` (`reports.sql`) | `GET /api/reports/authors`, `GET /api/reports/categories` | `minRatings` (0-1000, default 5), `limit` (default 20) | `RANK() OVER`, `HAVING`, `count(DISTINCT ...)` |
| q09 | Review activity per day | `ReviewActivity` (`reports.sql`) | `GET /api/reports/review-activity` | `days`, `endDate` | `generate_series`, per-day pre-aggregation, outer joins |
| q10 | Rating distribution | `GetRatingDistribution` (`ratings.sql`) | `GET /api/books/{bookId}/ratings` | `bookId` | `GROUP BY` on the rating value |

## Definitions

- **Pages read** (q07): a `completed` book counts `books.page_count`; a
  `reading` or `paused` book counts `reading_progress.current_page`;
  `want_to_read` counts nothing. Each user/book pair has one progress row, so
  no book is counted twice.
- **Liked book** (q06, q07 favorite categories): a book the user completed or
  rated 4 or 5. A rating counts even if the book is on no shelf.
- **Already known book** (q06): tracked in `reading_progress`, on any of the
  user's shelves, rated, or reviewed. These are never recommended.
- **Time window** (q05, q09): `days` whole UTC days ending on `endDate`,
  that is `[endDate - days + 1, endDate]`. The seed data ends on
  `2026-10-31`; pass `endDate=2026-10-31` for a stable demo or test window.
- **Rankings** (q08): `HAVING count(ratings) >= minRatings` is applied before
  `RANK()`, so one 5-star rating cannot top the leaderboard. Ties share a rank.
- **Consistency**: the reading-history report runs its three queries in one
  read-only `REPEATABLE READ` transaction, so totals and lists come from the
  same snapshot.

## Expected output on the seed data

Migrations `000001`-`000007` and seeds `001`-`004`. Personal reports are for
`reader1@example.com`; windows end on `2026-10-31`. These values are checked
independently in `backend/internal/integration/reports_integration_test.go`.

**q07** `GET /api/reports/user-stats`

```json
{"booksCompleted":3,"booksInProgress":5,"pagesRead":1783,"ratingsGiven":6,"averageRating":3.33,"reviewsWritten":6}
```

Favorite categories (reading-history): Fantasy (5 liked books), Romance (2),
Biography (1), then two more.

**q05** `GET /api/reports/popular?days=30&endDate=2026-10-31&limit=3`

| title | primaryAuthor | shelfAdds | ratings | averageRating | score |
| --- | --- | --- | --- | --- | --- |
| Forgotten Memory | Isaac Asimov | 3 | 2 | 4.00 | 8 |
| Radiant Season | Fyodor Dostoevsky | 3 | 1 | 3.00 | 7 |
| Wandering City | Flannery O'Connor | 2 | 2 | 2.00 | 6 |

**q04** `GET /api/reports/top-rated?minRatings=5&limit=3`

| title | primaryAuthor | averageRating | ratingCount |
| --- | --- | --- | --- |
| Fragile Garden | Gabriel Garcia Marquez | 4.86 | 7 |
| Ancient Garden | Franz Kafka | 4.71 | 7 |
| Golden Garden | George Orwell | 4.63 | 8 |

**q06** `GET /api/reports/recommendations?limit=3` (reader 1)

| title | primaryAuthor | score | averageRating |
| --- | --- | --- | --- |
| Golden Empire | Margaret Atwood | 7 | 4.00 |
| Golden Season | Doris Lessing | 7 | 4.00 |
| Hopeful Garden | Leo Tolstoy | 7 | 3.88 |

**q08** `GET /api/reports/authors?limit=3` and `GET /api/reports/categories?limit=3`

| name | bookCount | ratingCount | averageRating | ranking |
| --- | --- | --- | --- | --- |
| Franz Kafka | 8 | 13 | 4.08 | 1 |
| Chimamanda Ngozi Adichie | 8 | 20 | 4.00 | 2 |
| Neil Gaiman | 8 | 13 | 4.00 | 3 |
| Business (category) | 13 | 17 | 4.41 | 1 |
| Young Adult (category) | 13 | 18 | 4.22 | 2 |
| Literary Fiction (category) | 13 | 17 | 4.06 | 3 |

**q09** `GET /api/reports/review-activity?days=30&endDate=2026-10-31`: 30
points from `2026-10-02` to `2026-10-31`. Seeded reviews and ratings stop on
`2026-10-14`, so later days are 0:

| date | reviews | ratings |
| --- | --- | --- |
| 2026-10-13 | 10 | 10 |
| 2026-10-14 | 6 | 6 |
| 2026-10-15 | 0 | 0 |
