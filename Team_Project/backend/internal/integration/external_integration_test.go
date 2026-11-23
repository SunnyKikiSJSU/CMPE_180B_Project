//go:build integration

package integration

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/openlibrary"
)

// fakeOpenLibrary serves /search.json and /works/<id>.json from a map of
// search docs (JSON) and work records. Unknown works are 404; ids in fail
// answer 500. Each response is a little slow so concurrent imports overlap.
type fakeOpenLibrary struct {
	mu    sync.Mutex
	docs  map[string]string // work ID -> search doc JSON
	works map[string]string // work ID -> work record JSON
	order []string
	fail  map[string]bool
}

func newFakeOpenLibrary(t *testing.T) (*fakeOpenLibrary, string) {
	f := &fakeOpenLibrary{docs: map[string]string{}, works: map[string]string{}, fail: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv.URL
}

func (f *fakeOpenLibrary) add(id, doc, work string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[id], f.works[id] = doc, work
	f.order = append(f.order, id)
}

func (f *fakeOpenLibrary) serve(w http.ResponseWriter, r *http.Request) {
	time.Sleep(30 * time.Millisecond)
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == "/search.json":
		var docs []string
		if key := strings.TrimPrefix(r.URL.Query().Get("q"), "key:/works/"); key != r.URL.Query().Get("q") {
			if d, ok := f.docs[key]; ok && d != "" {
				docs = append(docs, d)
			}
		} else {
			for _, id := range f.order {
				if f.docs[id] != "" {
					docs = append(docs, f.docs[id])
				}
			}
		}
		fmt.Fprintf(w, `{"numFound":%d,"docs":[%s]}`, 1000+len(docs), strings.Join(docs, ","))
	case strings.HasPrefix(r.URL.Path, "/works/"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/works/"), ".json")
		if f.fail[id] {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if rec, ok := f.works[id]; ok {
			fmt.Fprint(w, rec)
			return
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

// olFixture is one test's set of unique work IDs and names, removed afterwards.
type olFixture struct {
	tag string
	ids []string
}

func newOLFixture(t *testing.T, pool *pgxpool.Pool, n int) olFixture {
	f := olFixture{tag: fmt.Sprintf("%06d", rand.IntN(1_000_000))}
	for i := 1; i <= n; i++ {
		f.ids = append(f.ids, fmt.Sprintf("OL%s%dW", f.tag, i))
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM books WHERE openlibrary_id = ANY($1)", f.ids)
		_, _ = pool.Exec(ctx, "DELETE FROM authors WHERE name LIKE '%' || $1", f.tag)
		_, _ = pool.Exec(ctx, "DELETE FROM categories WHERE name LIKE '%' || $1", f.tag)
	})
	return f
}

func olDoc(id, title, extra string) string {
	return fmt.Sprintf(`{"key":"/works/%s","title":%q%s}`, id, title, extra)
}

func olWork(id, title string) string {
	return fmt.Sprintf(`{"key":"/works/%s","title":%q,"type":{"key":"/type/work"},"description":"About %s."}`, id, title, title)
}

// bookOut is the Book fields these tests check.
type bookOut struct {
	ID                     uuid.UUID
	Title                  string
	Source                 string
	Authors                []struct{ Name string }
	Categories             []struct{ Name string }
	InfoLink               *string
	Description            *string
	PublishedDate          *string
	PublishedDatePrecision *string
	PageCount              *int
	ISBN13                 *string
}

type externalBook struct {
	Source     string
	ExternalID string
	BookID     *uuid.UUID
	Title      string
	ISBN13     *string
	InfoLink   string
}

// validSeedISBN returns a seeded book whose ISBN-13 has a correct check digit
// (most seed ISBNs are synthetic), so the normalizer keeps it.
func validSeedISBN(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, string) {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT book_id, isbn13 FROM books WHERE source = 'seed' AND isbn13 IS NOT NULL ORDER BY isbn13")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var isbn string
		if err := rows.Scan(&id, &isbn); err != nil {
			t.Fatal(err)
		}
		if _, ok := openlibrary.ISBN13(isbn); ok {
			return id, isbn
		}
	}
	t.Fatal("no seed book with a valid ISBN-13")
	return uuid.Nil, ""
}

func TestExternalSearchMarksLibraryBooks(t *testing.T) {
	fake, url := newFakeOpenLibrary(t)
	h, pool := setupWith(t, url)
	fx := newOLFixture(t, pool, 2)
	seedID, seedISBN := validSeedISBN(t, pool)
	fake.add(fx.ids[0], olDoc(fx.ids[0], "New work "+fx.tag, ""), "")
	fake.add(fx.ids[1], olDoc(fx.ids[1], "Seeded work "+fx.tag, `,"isbn":["`+seedISBN+`"]`), "")

	rec := call(h, "GET", "/api/external/books?q=work&pageSize=10", "", "")
	books := decode[[]externalBook](t, rec.Body.String())
	if rec.Code != 200 || rec.Header().Get("X-Total-Count") != "1002" || len(books) != 2 {
		t.Fatalf("search: %d %s %s", rec.Code, rec.Header().Get("X-Total-Count"), rec.Body.String())
	}
	if books[0].Source != "openlibrary" || books[0].ExternalID != fx.ids[0] || books[0].BookID != nil ||
		books[0].InfoLink != url+"/works/"+fx.ids[0] {
		t.Fatalf("new work: %+v", books[0])
	}
	// Same ISBN as a seed book: import would return that book, so mark it.
	if books[1].BookID == nil || *books[1].BookID != seedID {
		t.Fatalf("seeded work not marked: %+v (want %s)", books[1], seedID)
	}

	if rec := call(h, "GET", "/api/external/books", "", ""); rec.Code != 400 {
		t.Fatalf("no query: %d", rec.Code)
	}
	// The default setup points at an unreachable Open Library.
	hDown, _ := setup(t)
	rec = call(hDown, "GET", "/api/external/books?q=dune", "", "")
	if rec.Code != 502 || !strings.Contains(rec.Body.String(), "UPSTREAM_UNAVAILABLE") {
		t.Fatalf("Open Library down: %d %s", rec.Code, rec.Body.String())
	}
}

func TestImportOpenLibraryWork(t *testing.T) {
	fake, url := newFakeOpenLibrary(t)
	h, pool := setupWith(t, url)
	token, _ := newReader(t, h, pool)
	fx := newOLFixture(t, pool, 5)
	_, seedISBN := validSeedISBN(t, pool)
	title := "Imported work " + fx.tag
	fake.add(fx.ids[0], olDoc(fx.ids[0], title, fmt.Sprintf(
		`,"author_name":["Author %[1]s","Second %[1]s"],"subject":["Tagcat %[1]s","nyt:list=2020"],"first_publish_year":1999,`+
			`"number_of_pages_median":321,"cover_i":42,"editions":{"docs":[{"isbn":["9780441013593"],"publisher":["Pub"]}]}`, fx.tag)),
		olWork(fx.ids[0], title))
	// Merged into the first work.
	fake.add(fx.ids[1], "", fmt.Sprintf(`{"key":"/works/%s","type":{"key":"/type/redirect"},"location":"/works/%s"}`, fx.ids[1], fx.ids[0]))
	// Shares an ISBN with a seed book.
	fake.add(fx.ids[2], olDoc(fx.ids[2], "Edition of a seed "+fx.tag, `,"isbn":["`+seedISBN+`"]`), olWork(fx.ids[2], "Edition of a seed"))
	fake.fail[fx.ids[3]] = true

	rec := call(h, "POST", "/api/books/import", `{"source":"openlibrary","externalId":"`+fx.ids[0]+`"}`, token)
	b := decode[bookOut](t, rec.Body.String())
	if rec.Code != 201 || b.Source != "openlibrary" || b.Title != title || b.InfoLink == nil || *b.InfoLink != "https://openlibrary.org/works/"+fx.ids[0] ||
		len(b.Authors) != 2 || len(b.Categories) != 1 || b.Categories[0].Name != "Tagcat "+fx.tag ||
		b.PageCount == nil || *b.PageCount != 321 || b.ISBN13 == nil || *b.ISBN13 != "9780441013593" ||
		b.PublishedDate == nil || *b.PublishedDate != "1999-01-01" || b.PublishedDatePrecision == nil || *b.PublishedDatePrecision != "year" ||
		b.Description == nil || *b.Description != "About "+title+"." {
		t.Fatalf("import: %d %s", rec.Code, rec.Body.String())
	}

	// Idempotent: the same work, or an old ID that redirects to it, is 200.
	for _, id := range []string{fx.ids[0], fx.ids[1]} {
		rec := call(h, "POST", "/api/books/import", `{"externalId":"`+id+`"}`, token)
		if again := decode[bookOut](t, rec.Body.String()); rec.Code != 200 || again.ID != b.ID {
			t.Fatalf("re-import %s: %d %s", id, rec.Code, rec.Body.String())
		}
	}
	if n := count(t, pool, "SELECT count(*) FROM books WHERE openlibrary_id = $1", fx.ids[0]); n != 1 {
		t.Fatalf("%d rows for the work", n)
	}

	// A work whose edition is already in the library returns that book.
	rec = call(h, "POST", "/api/books/import", `{"externalId":"`+fx.ids[2]+`"}`, token)
	if seeded := decode[bookOut](t, rec.Body.String()); rec.Code != 200 || seeded.Source != "seed" {
		t.Fatalf("ISBN dedupe: %d %s", rec.Code, rec.Body.String())
	}

	// The imported book is a normal library book: shelve it, but not edit it.
	shelves := shelvesByType(t, h, token)
	if rec := call(h, "POST", "/api/shelves/"+shelves["want_to_read"].ID.String()+"/books", `{"bookId":"`+b.ID.String()+`"}`, token); rec.Code != 200 && rec.Code != 201 {
		t.Fatalf("shelve imported book: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "PATCH", "/api/books/"+b.ID.String(), `{"title":"Mine now"}`, token); rec.Code != 403 {
		t.Fatalf("edit imported book: %d", rec.Code)
	}
	// Library search now marks the work.
	books := decode[[]externalBook](t, call(h, "GET", "/api/external/books?q=x", "", "").Body.String())
	for _, eb := range books {
		if eb.ExternalID == fx.ids[0] && (eb.BookID == nil || *eb.BookID != b.ID) {
			t.Fatalf("search after import: %+v", eb)
		}
	}

	for _, c := range []struct {
		body string
		code int
	}{
		{`{"externalId":"` + fx.ids[4] + `"}`, 404},                   // unknown work
		{`{"externalId":"` + fx.ids[3] + `"}`, 502},                   // Open Library error
		{`{"externalId":"OL1M"}`, 400},                                // an edition ID
		{`{"source":"google","externalId":"` + fx.ids[0] + `"}`, 400}, // unsupported source
		{`{"externalId":"` + fx.ids[0] + `","title":"Forged"}`, 400},  // client metadata
	} {
		if rec := call(h, "POST", "/api/books/import", c.body, token); rec.Code != c.code {
			t.Errorf("%s: %d, want %d: %s", c.body, rec.Code, c.code, rec.Body.String())
		}
	}
	if rec := call(h, "POST", "/api/books/import", `{"externalId":"`+fx.ids[0]+`"}`, ""); rec.Code != 401 {
		t.Fatalf("anonymous import: %d", rec.Code)
	}
}

// BACKEND_PLAN.md Phase 4: two users import the same work at once.
func TestConcurrentImportReturnsSameBook(t *testing.T) {
	fake, url := newFakeOpenLibrary(t)
	h, pool := setupWith(t, url)
	fx := newOLFixture(t, pool, 1)
	fake.add(fx.ids[0], olDoc(fx.ids[0], "Race "+fx.tag, `,"author_name":["Racer `+fx.tag+`"]`), olWork(fx.ids[0], "Race"))
	tokens := []string{}
	for range 2 {
		tok, _ := newReader(t, h, pool)
		tokens = append(tokens, tok)
	}

	codes := make([]int, 2)
	ids := make([]uuid.UUID, 2)
	var wg sync.WaitGroup
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := call(h, "POST", "/api/books/import", `{"externalId":"`+fx.ids[0]+`"}`, tokens[i])
			codes[i] = rec.Code
			ids[i] = decode[bookOut](t, rec.Body.String()).ID
		}(i)
	}
	wg.Wait()
	if !((codes[0] == 201 && codes[1] == 200) || (codes[0] == 200 && codes[1] == 201)) || ids[0] != ids[1] {
		t.Fatalf("codes %v, ids %v", codes, ids)
	}
	if n := count(t, pool, "SELECT count(*) FROM books WHERE openlibrary_id = $1", fx.ids[0]); n != 1 {
		t.Fatalf("%d rows", n)
	}
	if n := count(t, pool, "SELECT count(*) FROM book_authors WHERE book_id = $1", ids[0]); n != 1 {
		t.Fatalf("%d author links", n)
	}
}
