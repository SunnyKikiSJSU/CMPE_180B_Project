//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type shelf struct {
	ID    uuid.UUID
	Name  string
	Type  string
	Count int
}

type progress struct {
	BookID          uuid.UUID
	Status          string
	CurrentPage     int
	TotalPages      int
	PercentComplete float64
	Version         int
}

type shelfResult struct {
	ShelfID      uuid.UUID
	BookID       uuid.UUID
	Progress     progress
	HistoryEvent string
}

// newReader registers a fresh user and deletes it (and everything it owns)
// after the test.
func newReader(t *testing.T, h http.Handler, pool *pgxpool.Pool) (string, uuid.UUID) {
	t.Helper()
	email := "reader-" + uuid.NewString()[:8] + "@example.com"
	rec := call(h, "POST", "/api/auth/register", `{"email":"`+email+`","password":"reading-pass","displayName":"Reader"}`, "")
	if rec.Code != 201 {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	reg := decode[struct {
		User        struct{ ID uuid.UUID }
		AccessToken string
	}](t, rec.Body.String())
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE user_id = $1", reg.User.ID) })
	return reg.AccessToken, reg.User.ID
}

func shelvesByType(t *testing.T, h http.Handler, token string) map[string]shelf {
	t.Helper()
	out := map[string]shelf{}
	for _, s := range decode[[]shelf](t, call(h, "GET", "/api/shelves", "", token).Body.String()) {
		if s.Type != "custom" {
			out[s.Type] = s
		} else {
			out["custom:"+s.Name] = s
		}
	}
	return out
}

func seedBook(t *testing.T, pool *pgxpool.Pool, pageCount bool) (uuid.UUID, int) {
	t.Helper()
	var id uuid.UUID
	var pages *int
	cond := "page_count IS NOT NULL AND page_count > 50"
	if !pageCount {
		cond = "page_count IS NULL"
	}
	if err := pool.QueryRow(context.Background(), "SELECT book_id, page_count FROM books WHERE "+cond+" ORDER BY title LIMIT 1").Scan(&id, &pages); err != nil {
		t.Fatalf("no seed book (%s): %v", cond, err)
	}
	if pages == nil {
		return id, 0
	}
	return id, *pages
}

func historyCount(t *testing.T, pool *pgxpool.Pool, userID, bookID uuid.UUID) int {
	return count(t, pool, "SELECT count(*) FROM reading_history WHERE user_id = $1 AND book_id = $2", userID, bookID)
}

func TestShelfManagement(t *testing.T) {
	h, pool := setup(t)
	token, _ := newReader(t, h, pool)
	other, _ := newReader(t, h, pool)

	shelves := shelvesByType(t, h, token)
	if len(shelves) != 4 {
		t.Fatalf("default shelves: %v", shelves)
	}
	rec := call(h, "POST", "/api/shelves", `{"name":"  Summer 2026 "}`, token)
	custom := decode[shelf](t, rec.Body.String())
	if rec.Code != 201 || custom.Name != "Summer 2026" || custom.Type != "custom" {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "POST", "/api/shelves", `{"name":"Summer 2026"}`, token); rec.Code != 409 || !strings.Contains(rec.Body.String(), "shelves_user_name_unique") {
		t.Fatalf("duplicate name: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "POST", "/api/shelves", `{"name":"x","type":"finished"}`, token); rec.Code != 400 {
		t.Fatalf("status type: %d", rec.Code)
	}
	if rec := call(h, "PATCH", "/api/shelves/"+custom.ID.String(), `{"name":"Beach reads"}`, token); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"Beach reads"`) {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body.String())
	}
	status := shelves["finished"].ID.String()
	if rec := call(h, "PATCH", "/api/shelves/"+status, `{"name":"Done"}`, token); rec.Code != 403 {
		t.Fatalf("rename status shelf: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/api/shelves/"+status, "", token); rec.Code != 403 {
		t.Fatalf("delete status shelf: %d", rec.Code)
	}
	// Another user's shelf does not exist for this caller.
	for _, rec := range []*struct{ method, body string }{{"PATCH", `{"name":"mine"}`}, {"DELETE", ""}} {
		if r := call(h, rec.method, "/api/shelves/"+custom.ID.String(), rec.body, other); r.Code != 404 {
			t.Fatalf("other user %s: %d", rec.method, r.Code)
		}
	}
	if rec := call(h, "GET", "/api/shelves/"+custom.ID.String()+"/books", "", other); rec.Code != 404 {
		t.Fatalf("other user list: %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/api/shelves/"+custom.ID.String(), "", token); rec.Code != 204 {
		t.Fatalf("delete: %d", rec.Code)
	}

	// At most 50 custom shelves.
	for i := 0; i < 50; i++ {
		if rec := call(h, "POST", "/api/shelves", fmt.Sprintf(`{"name":"Shelf %02d"}`, i), token); rec.Code != 201 {
			t.Fatalf("shelf %d: %d", i, rec.Code)
		}
	}
	if rec := call(h, "POST", "/api/shelves", `{"name":"One too many"}`, token); rec.Code != 409 {
		t.Fatalf("51st shelf: %d", rec.Code)
	}
}

func TestShelfMovesKeepProgressAndHistoryInStep(t *testing.T) {
	h, pool := setup(t)
	token, userID := newReader(t, h, pool)
	other, _ := newReader(t, h, pool)
	bookID, pages := seedBook(t, pool, true)
	shelves := shelvesByType(t, h, token)
	want, reading, finished := shelves["want_to_read"].ID, shelves["currently_reading"].ID, shelves["finished"].ID

	// Add to Want to read: progress is created, history written.
	rec := call(h, "POST", "/api/shelves/"+want.String()+"/books", `{"bookId":"`+bookID.String()+`"}`, token)
	res := decode[shelfResult](t, rec.Body.String())
	if rec.Code != 200 || res.Progress.Status != "want_to_read" || res.HistoryEvent != "shelf_moved" || historyCount(t, pool, userID, bookID) != 1 {
		t.Fatalf("add: %d %s", rec.Code, rec.Body.String())
	}
	// Move to Currently reading: status follows, old status shelf is left.
	rec = call(h, "PATCH", "/api/shelves/"+want.String()+"/books/"+bookID.String(), `{"targetShelfId":"`+reading.String()+`"}`, token)
	res = decode[shelfResult](t, rec.Body.String())
	if rec.Code != 200 || res.Progress.Status != "reading" {
		t.Fatalf("move: %d %s", rec.Code, rec.Body.String())
	}
	if n := count(t, pool, "SELECT count(*) FROM user_shelf_items WHERE book_id = $1 AND shelf_id = ANY($2)", bookID, []uuid.UUID{want, reading, finished}); n != 1 {
		t.Fatalf("book on %d status shelves", n)
	}
	if rec := call(h, "GET", "/api/shelves/"+want.String()+"/books", "", token); rec.Header().Get("X-Total-Count") != "0" {
		t.Fatalf("still on want to read: %s", rec.Body.String())
	}
	// Moving from a shelf the book is not on is a 404 and changes nothing.
	before := historyCount(t, pool, userID, bookID)
	if rec := call(h, "PATCH", "/api/shelves/"+want.String()+"/books/"+bookID.String(), `{"targetShelfId":"`+finished.String()+`"}`, token); rec.Code != 404 {
		t.Fatalf("move from wrong shelf: %d", rec.Code)
	}
	if historyCount(t, pool, userID, bookID) != before {
		t.Fatal("failed move wrote history")
	}
	// Finished: completed at the last page.
	rec = call(h, "POST", "/api/shelves/"+finished.String()+"/books", `{"bookId":"`+bookID.String()+`"}`, token)
	res = decode[shelfResult](t, rec.Body.String())
	if rec.Code != 200 || res.Progress.Status != "completed" || res.Progress.CurrentPage != pages || res.Progress.PercentComplete != 100 {
		t.Fatalf("finish: %d %s", rec.Code, rec.Body.String())
	}

	// Shelf book list carries the caller's progress.
	rec = call(h, "GET", "/api/shelves/"+finished.String()+"/books?sort=title", "", token)
	books := decode[[]apiBook](t, rec.Body.String())
	if rec.Code != 200 || len(books) != 1 || books[0].MyProgress == nil || books[0].MyProgress.Status != "completed" {
		t.Fatalf("shelf books: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "GET", "/api/shelves/"+finished.String()+"/books?sort=oldest", "", token); rec.Code != 400 {
		t.Fatalf("bad sort: %d", rec.Code)
	}

	// Custom shelves tag without changing status; moving between custom
	// shelves leaves the first one.
	tagA := decode[shelf](t, call(h, "POST", "/api/shelves", `{"name":"Tag A"}`, token).Body.String())
	tagB := decode[shelf](t, call(h, "POST", "/api/shelves", `{"name":"Tag B"}`, token).Body.String())
	res = decode[shelfResult](t, call(h, "POST", "/api/shelves/"+tagA.ID.String()+"/books", `{"bookId":"`+bookID.String()+`"}`, token).Body.String())
	if res.Progress.Status != "completed" {
		t.Fatalf("custom shelf changed status: %+v", res.Progress)
	}
	if rec := call(h, "PATCH", "/api/shelves/"+tagA.ID.String()+"/books/"+bookID.String(), `{"targetShelfId":"`+tagB.ID.String()+`"}`, token); rec.Code != 200 {
		t.Fatalf("custom move: %d", rec.Code)
	}
	s := shelvesByType(t, h, token)
	if s["custom:Tag A"].Count != 0 || s["custom:Tag B"].Count != 1 || s["finished"].Count != 1 {
		t.Fatalf("counts after custom move: %+v", s)
	}

	// Another user cannot use this user's shelf.
	if rec := call(h, "POST", "/api/shelves/"+finished.String()+"/books", `{"bookId":"`+bookID.String()+`"}`, other); rec.Code != 404 {
		t.Fatalf("foreign shelf: %d", rec.Code)
	}
	if rec := call(h, "POST", "/api/shelves/"+finished.String()+"/books", `{"bookId":"`+uuid.NewString()+`"}`, token); rec.Code != 404 || !strings.Contains(rec.Body.String(), "Book not found") {
		t.Fatalf("unknown book: %d %s", rec.Code, rec.Body.String())
	}

	// Removing from a status shelf removes the progress; custom tags stay.
	if rec := call(h, "DELETE", "/api/shelves/"+finished.String()+"/books/"+bookID.String(), "", token); rec.Code != 204 {
		t.Fatalf("remove: %d", rec.Code)
	}
	if rec := call(h, "GET", "/api/reading-progress/"+bookID.String(), "", token); rec.Code != 404 {
		t.Fatalf("progress after removal: %d", rec.Code)
	}
	if s := shelvesByType(t, h, token); s["custom:Tag B"].Count != 1 || s["finished"].Count != 0 {
		t.Fatalf("after removal: %+v", s)
	}
	if rec := call(h, "DELETE", "/api/shelves/"+finished.String()+"/books/"+bookID.String(), "", token); rec.Code != 404 {
		t.Fatalf("remove again: %d", rec.Code)
	}
	// History is append-only and survives.
	events := decode[[]struct{ EventType, ToStatus string }](t, call(h, "GET", "/api/reading-history?bookId="+bookID.String()+"&limit=100", "", token).Body.String())
	if len(events) < 4 || events[0].EventType != "shelf_moved" {
		t.Fatalf("history: %+v", events)
	}
}

func TestProgressUpdates(t *testing.T) {
	h, pool := setup(t)
	token, userID := newReader(t, h, pool)
	bookID, pages := seedBook(t, pool, true)
	path := "/api/reading-progress/" + bookID.String()

	rec := call(h, "PATCH", path, `{"currentPage":10,"status":"reading"}`, token)
	p := decode[progress](t, rec.Body.String())
	if rec.Code != 200 || p.Status != "reading" || p.CurrentPage != 10 || p.TotalPages != pages || p.Version != 1 {
		t.Fatalf("start: %d %s", rec.Code, rec.Body.String())
	}
	if s := shelvesByType(t, h, token); s["currently_reading"].Count != 1 {
		t.Fatalf("status shelf not set: %+v", s)
	}

	// Rejected updates change nothing (one transaction).
	before := historyCount(t, pool, userID, bookID)
	rec = call(h, "PATCH", path, fmt.Sprintf(`{"currentPage":%d}`, pages+1), token)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "current_page must be between 0 and") || !strings.Contains(rec.Body.String(), `"field":"currentPage"`) {
		t.Fatalf("beyond last page: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "PATCH", path, `{"percentComplete":50}`, token); rec.Code != 400 || !strings.Contains(rec.Body.String(), `"field":"percentComplete"`) {
		t.Fatalf("percent on a book with pages: %d %s", rec.Code, rec.Body.String())
	}
	if got := decode[progress](t, call(h, "GET", path, "", token).Body.String()); got.Version != p.Version || got.CurrentPage != 10 || historyCount(t, pool, userID, bookID) != before {
		t.Fatalf("rolled-back update left changes: %+v", got)
	}

	// Optimistic versions: a stale version is 409 STALE_VERSION.
	rec = call(h, "PATCH", path, fmt.Sprintf(`{"currentPage":20,"version":%d}`, p.Version), token)
	p2 := decode[progress](t, rec.Body.String())
	if rec.Code != 200 || p2.Version != p.Version+1 {
		t.Fatalf("versioned update: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "PATCH", path, fmt.Sprintf(`{"currentPage":30,"version":%d}`, p.Version), token); rec.Code != 409 || !strings.Contains(rec.Body.String(), "STALE_VERSION") {
		t.Fatalf("stale version: %d %s", rec.Code, rec.Body.String())
	}

	// Reaching the last page completes the book and moves it to Finished.
	rec = call(h, "PATCH", path, fmt.Sprintf(`{"currentPage":%d}`, pages), token)
	if done := decode[progress](t, rec.Body.String()); done.Status != "completed" || done.PercentComplete != 100 {
		t.Fatalf("completion: %s", rec.Body.String())
	}
	if s := shelvesByType(t, h, token); s["finished"].Count != 1 || s["currently_reading"].Count != 0 {
		t.Fatalf("finished shelf: %+v", s)
	}
	events := decode[[]struct{ EventType string }](t, call(h, "GET", "/api/reading-history?bookId="+bookID.String(), "", token).Body.String())
	if len(events) == 0 || events[0].EventType != "book_completed" {
		t.Fatalf("completion event: %+v", events)
	}

	list := call(h, "GET", "/api/reading-progress?status=completed", "", token)
	if list.Code != 200 || list.Header().Get("X-Total-Count") != "1" || !strings.Contains(list.Body.String(), `"book":{`) {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	if rec := call(h, "GET", "/api/reading-progress?status=done", "", token); rec.Code != 400 {
		t.Fatalf("bad status filter: %d", rec.Code)
	}
	if rec := call(h, "PATCH", path, `{}`, token); rec.Code != 400 {
		t.Fatalf("empty update: %d", rec.Code)
	}

	if rec := call(h, "DELETE", path, "", token); rec.Code != 204 {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := call(h, "DELETE", path, "", token); rec.Code != 404 {
		t.Fatalf("delete again: %d", rec.Code)
	}
	if s := shelvesByType(t, h, token); s["finished"].Count != 0 {
		t.Fatalf("status shelf kept after delete: %+v", s)
	}
}

func TestProgressWithoutPageCount(t *testing.T) {
	h, pool := setup(t)
	token, _ := newReader(t, h, pool)
	rec := call(h, "POST", "/api/books", `{"title":"IT no pages `+uuid.NewString()[:6]+`"}`, token)
	book := decode[apiBook](t, rec.Body.String())
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM books WHERE book_id = $1", book.ID) })

	rec = call(h, "PATCH", "/api/reading-progress/"+book.ID.String(), `{"status":"reading","percentComplete":37.5}`, token)
	p := decode[progress](t, rec.Body.String())
	if rec.Code != 200 || p.PercentComplete != 37.5 || p.TotalPages != 0 || p.Status != "reading" {
		t.Fatalf("percent without pages: %d %s", rec.Code, rec.Body.String())
	}
}

// Two clients update with the same version at once: exactly one wins.
func TestConcurrentProgressUpdates(t *testing.T) {
	h, pool := setup(t)
	token, _ := newReader(t, h, pool)
	bookID, _ := seedBook(t, pool, true)
	path := "/api/reading-progress/" + bookID.String()
	start := decode[progress](t, call(h, "PATCH", path, `{"currentPage":1,"status":"reading"}`, token).Body.String())

	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = call(h, "PATCH", path, fmt.Sprintf(`{"currentPage":%d,"version":%d}`, 5+i, start.Version), token).Code
		}(i)
	}
	wg.Wait()
	if !((codes[0] == 200 && codes[1] == 409) || (codes[0] == 409 && codes[1] == 200)) {
		t.Fatalf("expected one 200 and one 409, got %v", codes)
	}
	if got := decode[progress](t, call(h, "GET", path, "", token).Body.String()); got.Version != start.Version+1 {
		t.Fatalf("version after race: %d", got.Version)
	}
}
