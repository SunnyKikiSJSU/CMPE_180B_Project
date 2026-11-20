//go:build integration

package integration

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedEnd is the seed clock (database/seeds/003_demo_activity.sql); windows
// ending on it are deterministic whatever today's date is.
const seedEnd = "2026-10-31"

type readingReport struct {
	BooksRead          int
	PagesRead          int
	FavoriteCategories []struct {
		ID   uuid.UUID
		Name string
	}
	RecentActivity []struct {
		BookID      uuid.UUID
		Status      string
		CurrentPage int
		TotalPages  int
	}
}

type userStats struct {
	BooksCompleted  int
	BooksInProgress int
	PagesRead       int
	RatingsGiven    int
	AverageRating   float64
	ReviewsWritten  int
}

type popularBook struct {
	BookID        uuid.UUID
	Title         string
	PrimaryAuthor string
	ShelfAdds     int
	Ratings       int
	AverageRating float64
	Score         int
}

type ranking struct {
	AuthorID      uuid.UUID
	CategoryID    uuid.UUID
	Name          string
	BookCount     int
	RatingCount   int
	AverageRating float64
	Ranking       int
}

type activityPoint struct {
	Date    string
	Reviews int
	Ratings int
}

// report GETs path and checks the status and the X-Query-Id header.
func report[T any](t *testing.T, rec *httptest.ResponseRecorder, queryID string) T {
	t.Helper()
	if rec.Code != 200 || rec.Header().Get("X-Query-Id") != queryID {
		t.Fatalf("status %d, X-Query-Id %q (want %s): %s", rec.Code, rec.Header().Get("X-Query-Id"), queryID, rec.Body.String())
	}
	return decode[T](t, rec.Body.String())
}

func TestPublicReportsOnSeedData(t *testing.T) {
	h, pool := setup(t)

	t.Run("popular", func(t *testing.T) {
		books := report[[]popularBook](t, call(h, "GET", "/api/reports/popular?days=30&limit=100&endDate="+seedEnd, "", ""), "q05")
		if len(books) == 0 {
			t.Fatal("no popular books in the seed window")
		}
		for i, b := range books {
			if b.Score != 2*b.ShelfAdds+b.Ratings || (i > 0 && b.Score > books[i-1].Score) {
				t.Fatalf("row %d score/order: %+v", i, b)
			}
		}
		// Recount the first book independently over the same UTC window.
		top := books[0]
		adds := count(t, pool, `SELECT count(*) FROM user_shelf_items WHERE book_id = $1
			AND added_at >= TIMESTAMPTZ '2026-10-02 00:00:00+00' AND added_at < TIMESTAMPTZ '2026-11-01 00:00:00+00'`, top.BookID)
		ratings := count(t, pool, `SELECT count(*) FROM ratings WHERE book_id = $1
			AND created_at >= TIMESTAMPTZ '2026-10-02 00:00:00+00' AND created_at < TIMESTAMPTZ '2026-11-01 00:00:00+00'`, top.BookID)
		if top.ShelfAdds != adds || top.Ratings != ratings {
			t.Fatalf("top book %s: report %d adds/%d ratings, SQL %d/%d", top.Title, top.ShelfAdds, top.Ratings, adds, ratings)
		}
		// A window before any activity is empty, not null.
		if rec := call(h, "GET", "/api/reports/popular?days=1&endDate=2020-01-01", "", ""); rec.Body.String() != "[]\n" {
			t.Fatalf("empty window: %q", rec.Body.String())
		}
	})

	t.Run("review activity", func(t *testing.T) {
		points := report[[]activityPoint](t, call(h, "GET", "/api/reports/review-activity?days=30&endDate="+seedEnd, "", ""), "q09")
		if len(points) != 30 || points[0].Date != "2026-10-02" || points[29].Date != seedEnd {
			t.Fatalf("days: %d, %+v ... %+v", len(points), points[0], points[len(points)-1])
		}
		reviews, ratings := 0, 0
		for i, p := range points {
			if i > 0 {
				prev, _ := time.Parse(time.DateOnly, points[i-1].Date)
				if cur, _ := time.Parse(time.DateOnly, p.Date); !cur.Equal(prev.AddDate(0, 0, 1)) {
					t.Fatalf("gap between %s and %s", points[i-1].Date, p.Date)
				}
			}
			reviews += p.Reviews
			ratings += p.Ratings
		}
		wantReviews := count(t, pool, `SELECT count(*) FROM reviews
			WHERE created_at >= TIMESTAMPTZ '2026-10-02 00:00:00+00' AND created_at < TIMESTAMPTZ '2026-11-01 00:00:00+00'`)
		wantRatings := count(t, pool, `SELECT count(*) FROM ratings
			WHERE created_at >= TIMESTAMPTZ '2026-10-02 00:00:00+00' AND created_at < TIMESTAMPTZ '2026-11-01 00:00:00+00'`)
		if reviews == 0 || reviews != wantReviews || ratings != wantRatings {
			t.Fatalf("totals: %d reviews/%d ratings, SQL %d/%d", reviews, ratings, wantReviews, wantRatings)
		}
	})

	t.Run("top rated", func(t *testing.T) {
		books := report[[]struct {
			BookID        uuid.UUID
			AverageRating float64
			RatingCount   int
		}](t, call(h, "GET", "/api/reports/top-rated?minRatings=5&limit=20", "", ""), "q04")
		if len(books) == 0 {
			t.Fatal("no top-rated books")
		}
		for i, b := range books {
			if b.RatingCount < 5 || (i > 0 && b.AverageRating > books[i-1].AverageRating) {
				t.Fatalf("row %d: %+v", i, b)
			}
			if n := count(t, pool, "SELECT count(*) FROM ratings WHERE book_id = $1", b.BookID); n != b.RatingCount {
				t.Fatalf("book %s: %d ratings, SQL %d", b.BookID, b.RatingCount, n)
			}
		}
	})

	for _, kind := range []string{"authors", "categories"} {
		t.Run(kind+" ranking", func(t *testing.T) {
			rows := report[[]ranking](t, call(h, "GET", "/api/reports/"+kind, "", ""), "q08")
			if len(rows) == 0 || rows[0].Ranking != 1 {
				t.Fatalf("%s: %+v", kind, rows)
			}
			for i, r := range rows {
				if r.RatingCount < 5 || strings.HasPrefix(r.Name, "Smoke Test") {
					t.Fatalf("default minRatings=5 should hide %+v", r)
				}
				if i > 0 && (r.AverageRating > rows[i-1].AverageRating || r.Ranking < rows[i-1].Ranking) {
					t.Fatalf("order at %d: %+v after %+v", i, r, rows[i-1])
				}
			}
			// minRatings=0 brings back entries with few ratings.
			all := report[[]ranking](t, call(h, "GET", "/api/reports/"+kind+"?minRatings=0&limit=100", "", ""), "q08")
			if len(all) <= len(rows) {
				t.Fatalf("minRatings=0: %d rows, default %d", len(all), len(rows))
			}
		})
	}
	// Independent check of the top author's numbers.
	top := report[[]ranking](t, call(h, "GET", "/api/reports/authors?limit=1", "", ""), "q08")[0]
	var avg float64
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT round(avg(rt.rating_value), 2)::float8, count(*)
		FROM book_authors ba JOIN ratings rt ON rt.book_id = ba.book_id
		WHERE ba.author_id = $1`, top.AuthorID).Scan(&avg, &n); err != nil {
		t.Fatal(err)
	}
	if avg != top.AverageRating || n != top.RatingCount {
		t.Fatalf("top author %s: report %.2f/%d, SQL %.2f/%d", top.Name, top.AverageRating, top.RatingCount, avg, n)
	}

	t.Run("query catalog", func(t *testing.T) {
		rec := call(h, "GET", "/api/reports/queries", "", "")
		entries := decode[[]struct {
			ID        string
			Endpoints []string
			Queries   []string
			SQL       string
		}](t, rec.Body.String())
		if rec.Code != 200 || len(entries) != 10 {
			t.Fatalf("catalog: %d, %d entries", rec.Code, len(entries))
		}
		for i, e := range entries {
			if want := "q" + leftPad(i+1); e.ID != want || len(e.Endpoints) == 0 {
				t.Fatalf("entry %d: %+v", i, e)
			}
			for _, q := range e.Queries {
				if !strings.Contains(e.SQL, "-- name: "+q+" ") {
					t.Fatalf("%s: SQL lacks %s", e.ID, q)
				}
			}
		}
	})
}

func leftPad(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func TestPersonalReportsOnSeedUser(t *testing.T) {
	h, pool := setup(t)
	token := login(t, h, "reader1@example.com", "Password123!")
	var userID uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT user_id FROM users WHERE email = 'reader1@example.com'").Scan(&userID); err != nil {
		t.Fatal(err)
	}

	rep := report[readingReport](t, call(h, "GET", "/api/reports/reading-history", "", token), "q07")
	// Independent pages-read rule: page_count for completed books,
	// current_page for reading/paused, nothing for want_to_read.
	wantPages := count(t, pool, `
		SELECT coalesce(sum(CASE rp.status WHEN 'completed' THEN coalesce(b.page_count, 0)
		                                   WHEN 'want_to_read' THEN 0
		                                   ELSE rp.current_page END), 0)
		FROM reading_progress rp JOIN books b USING (book_id) WHERE rp.user_id = $1`, userID)
	wantRead := count(t, pool, "SELECT count(*) FROM reading_progress WHERE user_id = $1 AND status = 'completed'", userID)
	if rep.BooksRead == 0 || rep.BooksRead != wantRead || rep.PagesRead != wantPages ||
		len(rep.FavoriteCategories) == 0 || len(rep.FavoriteCategories) > 5 || len(rep.RecentActivity) == 0 {
		t.Fatalf("reading history %+v, want %d books / %d pages", rep, wantRead, wantPages)
	}

	stats := report[userStats](t, call(h, "GET", "/api/reports/user-stats", "", token), "q07")
	if stats.BooksCompleted != rep.BooksRead || stats.PagesRead != rep.PagesRead || stats.RatingsGiven == 0 ||
		stats.RatingsGiven != count(t, pool, "SELECT count(*) FROM ratings WHERE user_id = $1", userID) {
		t.Fatalf("user stats %+v disagree with reading history %+v", stats, rep)
	}

	recs := report[[]popularBook](t, call(h, "GET", "/api/reports/recommendations?limit=20", "", token), "q06")
	if len(recs) == 0 {
		t.Fatal("no recommendations for a seeded reader")
	}
	for _, b := range recs {
		if n := count(t, pool, `SELECT
			(SELECT count(*) FROM reading_progress WHERE user_id = $1 AND book_id = $2) +
			(SELECT count(*) FROM ratings WHERE user_id = $1 AND book_id = $2) +
			(SELECT count(*) FROM reviews WHERE user_id = $1 AND book_id = $2) +
			(SELECT count(*) FROM user_shelf_items i JOIN shelves s USING (shelf_id) WHERE s.user_id = $1 AND i.book_id = $2)`,
			userID, b.BookID); n != 0 {
			t.Fatalf("recommended a book the reader already has: %s", b.Title)
		}
	}
}

// newCategorizedBook inserts a throwaway book in one category.
func newCategorizedBook(t *testing.T, pool *pgxpool.Pool, categoryID uuid.UUID, pages *int) uuid.UUID {
	t.Helper()
	id := newBook(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE books SET page_count = $2 WHERE book_id = $1", id, pages); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO book_categories (book_id, category_id) VALUES ($1, $2)", id, categoryID); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPersonalReportsFollowActivity(t *testing.T) {
	h, pool := setup(t)
	token, _ := newReader(t, h, pool)

	// A new reader: zeros and empty lists (never null).
	rec := call(h, "GET", "/api/reports/reading-history", "", token)
	if rec.Code != 200 || rec.Body.String() != `{"booksRead":0,"pagesRead":0,"favoriteCategories":[],"recentActivity":[]}`+"\n" {
		t.Fatalf("empty report: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "GET", "/api/reports/recommendations", "", token); rec.Code != 200 || rec.Body.String() != "[]\n" {
		t.Fatalf("cold-start recommendations: %d %s", rec.Code, rec.Body.String())
	}

	var cats []uuid.UUID
	rows, err := pool.Query(context.Background(), `
		SELECT category_id FROM book_categories GROUP BY category_id ORDER BY count(*) DESC, category_id LIMIT 3`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id uuid.UUID
		_ = rows.Scan(&id)
		cats = append(cats, id)
	}
	rows.Close()
	p300, p200 := 300, 200
	finishedBook := newCategorizedBook(t, pool, cats[0], &p300)
	readingBook := newCategorizedBook(t, pool, cats[1], &p200)
	ratedBook := newCategorizedBook(t, pool, cats[2], nil)

	shelves := shelvesByType(t, h, token)
	if rec := call(h, "POST", "/api/shelves/"+shelves["finished"].ID.String()+"/books", `{"bookId":"`+finishedBook.String()+`"}`, token); rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("finish: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "POST", "/api/shelves/"+shelves["currently_reading"].ID.String()+"/books", `{"bookId":"`+readingBook.String()+`"}`, token); rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("start reading: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "PATCH", "/api/reading-progress/"+readingBook.String(), `{"currentPage":50}`, token); rec.Code != 200 {
		t.Fatalf("progress: %d %s", rec.Code, rec.Body.String())
	}
	// Rated 5 without being on any shelf: still a liked book.
	if rec := call(h, "POST", "/api/books/"+ratedBook.String()+"/ratings", `{"value":5}`, token); rec.Code != 200 {
		t.Fatalf("rate: %d %s", rec.Code, rec.Body.String())
	}

	rep := report[readingReport](t, call(h, "GET", "/api/reports/reading-history", "", token), "q07")
	favorites := map[uuid.UUID]bool{}
	for _, c := range rep.FavoriteCategories {
		favorites[c.ID] = true
	}
	if rep.BooksRead != 1 || rep.PagesRead != 300+50 || !favorites[cats[0]] || !favorites[cats[2]] || favorites[cats[1]] ||
		len(rep.RecentActivity) != 2 || rep.RecentActivity[0].BookID != readingBook || rep.RecentActivity[0].CurrentPage != 50 {
		t.Fatalf("after activity: %+v", rep)
	}
	stats := report[userStats](t, call(h, "GET", "/api/reports/user-stats", "", token), "q07")
	if stats != (userStats{BooksCompleted: 1, BooksInProgress: 1, PagesRead: 350, RatingsGiven: 1, AverageRating: 5}) {
		t.Fatalf("user stats: %+v", stats)
	}

	recs := report[[]popularBook](t, call(h, "GET", "/api/reports/recommendations?limit=100", "", token), "q06")
	if len(recs) == 0 {
		t.Fatal("no recommendations after liking two categories")
	}
	for _, b := range recs {
		if b.BookID == finishedBook || b.BookID == readingBook || b.BookID == ratedBook {
			t.Fatalf("recommended own book %s", b.BookID)
		}
		if n := count(t, pool, "SELECT count(*) FROM book_categories WHERE book_id = $1 AND category_id = ANY($2)", b.BookID, []uuid.UUID{cats[0], cats[2]}); n == 0 {
			t.Fatalf("recommendation %s is outside the liked categories", b.Title)
		}
	}

	// Default window ends today (UTC), so today's activity shows up.
	popular := report[[]popularBook](t, call(h, "GET", "/api/reports/popular?days=1&limit=100", "", ""), "q05")
	seen := map[uuid.UUID]popularBook{}
	for _, b := range popular {
		seen[b.BookID] = b
	}
	if seen[finishedBook].ShelfAdds != 1 || seen[ratedBook].Ratings != 1 || seen[ratedBook].AverageRating != 5 {
		t.Fatalf("today's popular: finished %+v, rated %+v", seen[finishedBook], seen[ratedBook])
	}
	points := report[[]activityPoint](t, call(h, "GET", "/api/reports/review-activity?days=1", "", ""), "q09")
	if len(points) != 1 || points[0].Date != time.Now().UTC().Format(time.DateOnly) || points[0].Ratings < 1 {
		t.Fatalf("today's activity: %+v", points)
	}
}
