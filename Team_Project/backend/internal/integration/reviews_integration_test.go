//go:build integration

package integration

import (
	"context"
	"fmt"
	"math"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type summary struct {
	AverageRating float64
	RatingCount   int
	ReviewCount   int
}

type ratingSummary struct {
	AverageRating float64
	RatingCount   int
	Distribution  map[string]int
}

type rating struct {
	BookID    uuid.UUID
	UserID    uuid.UUID
	Value     int
	Summary   *summary
	BookTitle string
}

type review struct {
	ID     uuid.UUID
	BookID uuid.UUID
	UserID uuid.UUID
	Title  *string
	Body   string
	Rating    *int
	User      *struct{ DisplayName string }
	BookTitle string
}

type apiError struct {
	Error struct {
		Code    string
		Details map[string]any
	}
}

// newBook inserts a book that no other test touches, so its summary starts at
// zero. Deleting it cascades to its ratings, reviews, and summary.
func newBook(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(),
		"INSERT INTO books (title, source) VALUES ($1, 'user') RETURNING book_id",
		"Review test "+uuid.NewString()[:8]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM books WHERE book_id = $1", id) })
	return id
}

// storedSummary reads the trigger-maintained row next to the real aggregates.
func storedSummary(t *testing.T, pool *pgxpool.Pool, bookID uuid.UUID) (stored, actual summary) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `
		SELECT s.average_rating::float8, s.rating_count, s.review_count,
		       coalesce((SELECT round(avg(rating_value), 2) FROM ratings WHERE book_id = $1), 0)::float8,
		       (SELECT count(*) FROM ratings WHERE book_id = $1),
		       (SELECT count(*) FROM reviews WHERE book_id = $1)
		FROM book_rating_summaries s WHERE s.book_id = $1`, bookID).
		Scan(&stored.AverageRating, &stored.RatingCount, &stored.ReviewCount,
			&actual.AverageRating, &actual.RatingCount, &actual.ReviewCount)
	if err != nil {
		t.Fatal(err)
	}
	return stored, actual
}

func wantSummary(t *testing.T, pool *pgxpool.Pool, bookID uuid.UUID, want summary) {
	t.Helper()
	stored, actual := storedSummary(t, pool, bookID)
	if stored != want || actual != want {
		t.Fatalf("summary: stored %+v, actual %+v, want %+v", stored, actual, want)
	}
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	e := decode[apiError](t, rec.Body.String())
	if e.Error.Code != code || (field != "" && e.Error.Details["field"] != field) {
		t.Fatalf("error %+v, want %s field %q", e.Error, code, field)
	}
}

func TestRatingsAndSummary(t *testing.T) {
	h, pool := setup(t)
	bookID := newBook(t, pool)
	path := "/api/books/" + bookID.String() + "/ratings"
	tokens := make([]string, 3)
	for i := range tokens {
		tokens[i], _ = newReader(t, h, pool)
	}

	for i, v := range []int{5, 4, 1} {
		rec := call(h, "POST", path, fmt.Sprintf(`{"value":%d}`, v), tokens[i])
		if rec.Code != 200 {
			t.Fatalf("rate %d: %d %s", v, rec.Code, rec.Body.String())
		}
		if r := decode[rating](t, rec.Body.String()); r.Value != v || r.Summary == nil || r.Summary.RatingCount != i+1 {
			t.Fatalf("rate %d response: %+v", v, r)
		}
	}
	wantSummary(t, pool, bookID, summary{3.33, 3, 0})
	got := decode[ratingSummary](t, call(h, "GET", path, "", "").Body.String())
	wantDist := map[string]int{"1": 1, "2": 0, "3": 0, "4": 1, "5": 1}
	if got.RatingCount != 3 || got.AverageRating != 3.33 || fmt.Sprint(got.Distribution) != fmt.Sprint(wantDist) {
		t.Fatalf("GET ratings: %+v", got)
	}

	// A second rating by the same user replaces the first (summary UPDATE path).
	if rec := call(h, "POST", path, `{"value":2}`, tokens[0]); rec.Code != 200 {
		t.Fatalf("re-rate: %d", rec.Code)
	}
	wantSummary(t, pool, bookID, summary{2.33, 3, 0})

	// DELETE path; a second delete has nothing to remove.
	if rec := call(h, "DELETE", path, "", tokens[2]); rec.Code != 204 {
		t.Fatalf("delete rating: %d", rec.Code)
	}
	wantSummary(t, pool, bookID, summary{3.00, 2, 0})
	rec := call(h, "DELETE", path, "", tokens[2])
	wantError(t, rec, 404, "NOT_FOUND", "")

	for _, body := range []string{`{"value":0}`, `{"value":6}`, `{}`, `{"value":4.5}`} {
		rec := call(h, "POST", path, body, tokens[0])
		if rec.Code != 400 {
			t.Fatalf("rate %s: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	rec = call(h, "POST", path, `{"value":0}`, tokens[0])
	wantError(t, rec, 400, "VALIDATION_ERROR", "value")
	wantSummary(t, pool, bookID, summary{3.00, 2, 0})

	unknown := "/api/books/" + uuid.NewString() + "/ratings"
	rec = call(h, "POST", unknown, `{"value":3}`, tokens[0])
	wantError(t, rec, 404, "NOT_FOUND", "")
	rec = call(h, "GET", unknown, "", "")
	wantError(t, rec, 404, "NOT_FOUND", "")
	if rec := call(h, "POST", path, `{"value":3}`, ""); rec.Code != 401 {
		t.Fatalf("anonymous rate: %d", rec.Code)
	}

	rec = call(h, "GET", "/api/users/me/ratings", "", tokens[0])
	mine := decode[[]rating](t, rec.Body.String())
	if rec.Code != 200 || len(mine) != 1 || mine[0].BookID != bookID || mine[0].Value != 2 || rec.Header().Get("X-Total-Count") != "1" ||
		!strings.HasPrefix(mine[0].BookTitle, "Review test ") {
		t.Fatalf("my ratings: %d %+v", rec.Code, mine)
	}
}

func TestReviewLifecycleAndOwnership(t *testing.T) {
	h, pool := setup(t)
	bookID := newBook(t, pool)
	reviewsPath := "/api/books/" + bookID.String() + "/reviews"
	owner, ownerID := newReader(t, h, pool)
	other, _ := newReader(t, h, pool)

	rec := call(h, "POST", reviewsPath, `{"title":"  Great  ","body":"  Loved it.  "}`, owner)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	created := decode[review](t, rec.Body.String())
	if created.UserID != ownerID || created.Body != "Loved it." || created.Title == nil || *created.Title != "Great" {
		t.Fatalf("created: %+v", created)
	}
	wantSummary(t, pool, bookID, summary{0, 0, 1})

	rec = call(h, "POST", reviewsPath, `{"body":"Second try"}`, owner)
	wantError(t, rec, 409, "CONFLICT", "")

	for body, field := range map[string]string{
		`{"body":"   "}`:                                "body",
		`{"title":"x","body":""}`:                       "body",
		`{"title":"` + longText(201) + `","body":"ok"}`: "title",
		`{"body":"` + longText(5001) + `"}`:             "body",
	} {
		rec := call(h, "POST", reviewsPath, body, other)
		wantError(t, rec, 400, "VALIDATION_ERROR", field)
	}
	rec = call(h, "POST", "/api/books/"+uuid.NewString()+"/reviews", `{"body":"Nope"}`, other)
	wantError(t, rec, 404, "NOT_FOUND", "")

	// Another user's review behaves as if it did not exist.
	reviewPath := "/api/reviews/" + created.ID.String()
	rec = call(h, "PATCH", reviewPath, `{"body":"Hijacked"}`, other)
	wantError(t, rec, 404, "NOT_FOUND", "")
	rec = call(h, "DELETE", reviewPath, "", other)
	wantError(t, rec, 404, "NOT_FOUND", "")

	rec = call(h, "PATCH", reviewPath, `{"body":"Still great."}`, owner)
	if kept := decode[review](t, rec.Body.String()); rec.Code != 200 || kept.Title == nil || *kept.Title != "Great" || kept.Body != "Still great." {
		t.Fatalf("patch without title: %d %+v", rec.Code, kept)
	}
	rec = call(h, "PATCH", reviewPath, `{"title":"","body":"Even better the second time."}`, owner)
	updated := decode[review](t, rec.Body.String())
	if rec.Code != 200 || updated.ID != created.ID || updated.Title != nil || updated.Body != "Even better the second time." {
		t.Fatalf("update: %d %+v", rec.Code, updated)
	}

	rec = call(h, "GET", "/api/users/me/reviews", "", owner)
	mine := decode[[]review](t, rec.Body.String())
	if rec.Code != 200 || len(mine) != 1 || mine[0].ID != created.ID || !strings.HasPrefix(mine[0].BookTitle, "Review test ") {
		t.Fatalf("my reviews: %d %+v", rec.Code, mine)
	}

	if rec := call(h, "DELETE", reviewPath, "", owner); rec.Code != 204 {
		t.Fatalf("delete: %d", rec.Code)
	}
	wantSummary(t, pool, bookID, summary{0, 0, 0})
	rec = call(h, "DELETE", reviewPath, "", owner)
	wantError(t, rec, 404, "NOT_FOUND", "")
	if rec := call(h, "DELETE", "/api/reviews/not-a-uuid", "", owner); rec.Code != 404 {
		t.Fatalf("malformed id: %d", rec.Code)
	}
}

func longText(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

func TestBookReviewListSortAndPaging(t *testing.T) {
	h, pool := setup(t)
	bookID := newBook(t, pool)
	path := "/api/books/" + bookID.String()
	for _, v := range []int{2, 5, 3} {
		token, _ := newReader(t, h, pool)
		rec := call(h, "PUT", path+"/review", fmt.Sprintf(`{"rating":%d,"body":"Rated %d"}`, v, v), token)
		if rec.Code != 200 {
			t.Fatalf("put %d: %d %s", v, rec.Code, rec.Body.String())
		}
	}

	rec := call(h, "GET", path+"/reviews?sort=rating&pageSize=2", "", "")
	page1 := decode[[]review](t, rec.Body.String())
	if rec.Code != 200 || rec.Header().Get("X-Total-Count") != "3" || len(page1) != 2 ||
		*page1[0].Rating != 5 || *page1[1].Rating != 3 || page1[0].User == nil || page1[0].User.DisplayName != "Reader" {
		t.Fatalf("page 1: %d %s %+v", rec.Code, rec.Header().Get("X-Total-Count"), page1)
	}
	rec = call(h, "GET", path+"/reviews?sort=rating&pageSize=2&page=2", "", "")
	if page2 := decode[[]review](t, rec.Body.String()); len(page2) != 1 || *page2[0].Rating != 2 {
		t.Fatalf("page 2: %+v", page2)
	}
	rec = call(h, "GET", path+"/reviews?page=9", "", "")
	if rec.Code != 200 || rec.Body.String() != "[]\n" || rec.Header().Get("X-Total-Count") != "3" {
		t.Fatalf("past the end: %d %q %s", rec.Code, rec.Body.String(), rec.Header().Get("X-Total-Count"))
	}
	if newest := decode[[]review](t, call(h, "GET", path+"/reviews", "", "").Body.String()); len(newest) != 3 || newest[0].Body != "Rated 3" {
		t.Fatalf("newest first: %+v", newest)
	}
	rec = call(h, "GET", path+"/reviews?sort=title", "", "")
	wantError(t, rec, 400, "VALIDATION_ERROR", "sort")
	rec = call(h, "GET", "/api/books/"+uuid.NewString()+"/reviews", "", "")
	wantError(t, rec, 404, "NOT_FOUND", "")
}

type saveResult struct {
	Review  review
	Rating  *rating
	Summary summary
}

func TestSaveReviewAndRatingInOneTransaction(t *testing.T) {
	h, pool := setup(t)
	bookID := newBook(t, pool)
	path := "/api/books/" + bookID.String() + "/review"
	token, userID := newReader(t, h, pool)

	rec := call(h, "PUT", path, `{"rating":4,"title":"First","body":"Solid."}`, token)
	first := decode[saveResult](t, rec.Body.String())
	if rec.Code != 200 || first.Rating == nil || first.Rating.Value != 4 || first.Review.Rating == nil || *first.Review.Rating != 4 ||
		first.Summary != (summary{4, 1, 1}) || first.Review.UserID != userID {
		t.Fatalf("first PUT: %d %+v", rec.Code, first)
	}

	// Without "rating" the existing rating stays; PUT replaces the title too.
	rec = call(h, "PUT", path, `{"body":"Even more solid."}`, token)
	second := decode[saveResult](t, rec.Body.String())
	if rec.Code != 200 || second.Review.ID != first.Review.ID || second.Review.Title != nil ||
		second.Review.Body != "Even more solid." || second.Rating == nil || second.Rating.Value != 4 {
		t.Fatalf("second PUT: %d %+v", rec.Code, second)
	}
	wantSummary(t, pool, bookID, summary{4, 1, 1})

	rec = call(h, "PUT", path, `{"rating":9,"body":"Bad rating"}`, token)
	wantError(t, rec, 400, "VALIDATION_ERROR", "rating")

	// Force the review step to fail after the rating step already ran inside
	// the transaction: the new rating must roll back with it.
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION test_fail_review() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.review_text = 'force rollback' THEN
				RAISE EXCEPTION 'forced failure' USING ERRCODE = '23514';
			END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER test_fail_review BEFORE INSERT OR UPDATE ON reviews
			FOR EACH ROW EXECUTE FUNCTION test_fail_review();`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DROP TRIGGER IF EXISTS test_fail_review ON reviews; DROP FUNCTION IF EXISTS test_fail_review()")
	})
	rec = call(h, "PUT", path, `{"rating":1,"body":"force rollback"}`, token)
	wantError(t, rec, 400, "VALIDATION_ERROR", "")
	wantSummary(t, pool, bookID, summary{4, 1, 1})
	if n := count(t, pool, "SELECT count(*) FROM ratings WHERE book_id = $1 AND user_id = $2 AND rating_value = 4", bookID, userID); n != 1 {
		t.Fatalf("rating changed despite rollback")
	}
	if n := count(t, pool, "SELECT count(*) FROM reviews WHERE book_id = $1 AND review_text = 'Even more solid.'", bookID); n != 1 {
		t.Fatalf("review changed despite rollback")
	}

	rec = call(h, "PUT", "/api/books/"+uuid.NewString()+"/review", `{"rating":3,"body":"Missing"}`, token)
	wantError(t, rec, 404, "NOT_FOUND", "")
}

// BACKEND_PLAN.md §7.2: one user posts the same review twice at once.
func TestDuplicateReviewRace(t *testing.T) {
	h, pool := setup(t)
	bookID := newBook(t, pool)
	token, _ := newReader(t, h, pool)
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = call(h, "POST", "/api/books/"+bookID.String()+"/reviews", `{"body":"Posted twice"}`, token).Code
		}(i)
	}
	wg.Wait()
	if !((codes[0] == 201 && codes[1] == 409) || (codes[0] == 409 && codes[1] == 201)) {
		t.Fatalf("expected one 201 and one 409, got %v", codes)
	}
	wantSummary(t, pool, bookID, summary{0, 0, 1})
}

// BACKEND_PLAN.md §7.2: many users rate one book at the same time. Before
// migration 000007 the summary lost updates here (see docs/evidence/concurrency.md).
func TestConcurrentRatingsKeepSummaryConsistent(t *testing.T) {
	h, pool := setup(t)
	bookID := newBook(t, pool)
	const readers = 12
	tokens := make([]string, readers)
	for i := range tokens {
		tokens[i], _ = newReader(t, h, pool)
	}
	codes := make([]int, readers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			body := `{"value":` + strconv.Itoa(i%5+1) + `}`
			codes[i] = call(h, "POST", "/api/books/"+bookID.String()+"/ratings", body, tokens[i]).Code
		}(i)
	}
	close(start)
	wg.Wait()
	for i, c := range codes {
		if c != 200 {
			t.Fatalf("reader %d: %d", i, c)
		}
	}
	// Values 1..5 repeating over 12 readers: 1,2,3,4,5,1,2,3,4,5,1,2 = 33.
	want := summary{math.Round(33.0/readers*100) / 100, readers, 0}
	wantSummary(t, pool, bookID, want)
}
