package openlibrary

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestISBN13(t *testing.T) {
	for raw, want := range map[string]string{
		"9780441013593":     "9780441013593",
		"978-0-441-01359-3": "9780441013593",
		"0441013597":        "9780441013593",
		"080442957X":        "9780804429573",
		"9780441013594":     "", // bad check digit
		"0441013598":        "", // bad ISBN-10 check digit
		"97804410135":       "",
		"abcdefghij":        "",
	} {
		got, ok := ISBN13(raw)
		if got != want || ok != (want != "") {
			t.Errorf("ISBN13(%q) = %q, %v; want %q", raw, got, ok, want)
		}
	}
}

func TestCategoriesDropsCatalogNoise(t *testing.T) {
	// Subjects of "Dune" (OL893414W) as Open Library returns them.
	got := Categories([]string{
		"Dune (Imaginary place)", "Fiction", "Fiction, science fiction, general", "Dune (imaginary place), fiction",
		"New York Times reviewed", "Science fiction", "Science-fiction", "American literature",
		"nyt:mass-market-monthly=2021-11-07", "New York Times bestseller", "science fiction", "Accessible book",
		"Reading Level-Grade 7", "Translations into Russian", "Imaginary wars and battles", "Ecology", "Fantasy",
	})
	want := []string{"Fiction", "Science fiction", "American literature", "Imaginary wars and battles", "Ecology"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := Categories([]string{"poetry"}); !reflect.DeepEqual(got, []string{"Poetry"}) {
		t.Fatalf("capitalized: %q", got)
	}
}

func TestDescription(t *testing.T) {
	raw := map[string]any{"type": "/type/text", "value": "Set on Arrakis.\r\n\r\n\r\n\r\nPaul\x07 grows up.\n\n" +
		"([source][1])\n\n[1]: https://example.com/dune\n----------\nAlso contained in:\n - [Dune Saga](https://openlibrary.org/x)"}
	if got := description(raw); got == nil || *got != "Set on Arrakis.\n\nPaul grows up.\n\n([source][1])" {
		t.Fatalf("description: %q", deref(got))
	}
	if got := description("  Plain text.  "); got == nil || *got != "Plain text." {
		t.Fatalf("string description: %q", deref(got))
	}
	if description(nil) != nil || description(map[string]any{"value": "  "}) != nil {
		t.Fatal("empty descriptions must be nil")
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

const duneDoc = `{"key":"/works/OL893414W","title":"  Dune ","author_name":["Frank Herbert","frank herbert",""],
"first_publish_year":1965,"number_of_pages_median":604,"cover_i":11481354,
"subject":["Science fiction","nyt:x=1","Fantasy"],"isbn":["0441013597","9780340960196","bad"],
"editions":{"docs":[{"isbn":["0425038912","9780425038918"],"publisher":["Berkley"]}]}}`

// fake serves /search.json and /works/<id>.json from the given handlers.
func fake(t *testing.T, search, work http.HandlerFunc) (*Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		switch {
		case r.URL.Path == "/search.json" && search != nil:
			search(w, r)
		case strings.HasPrefix(r.URL.Path, "/works/") && work != nil:
			work(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-agent"), &calls
}

func TestSearch(t *testing.T) {
	c, _ := fake(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("title") != "dune" || q.Get("author") != "herbert" || q.Get("subject") != "" || q.Get("sort") != "new" ||
			q.Get("page") != "2" || q.Get("limit") != "5" || !strings.Contains(q.Get("fields"), "editions.isbn") || q.Has("q") {
			t.Errorf("query: %v", q)
		}
		fmt.Fprintf(w, `{"numFound":42,"docs":[%s,{"key":"/books/OL1M","title":"An edition, not a work"},{"key":"/works/OL2W","title":" "}]}`, duneDoc)
	}, nil)
	total, books, err := c.Search(context.Background(), SearchParams{Title: "dune", Author: "herbert", Newest: true, Page: 2, Limit: 5})
	if err != nil || total != 42 || len(books) != 1 {
		t.Fatalf("search: %v %d %d", err, total, len(books))
	}
	b := books[0]
	if b.Source != "openlibrary" || b.ExternalID != "OL893414W" || b.Title != "Dune" || !reflect.DeepEqual(b.Authors, []string{"Frank Herbert"}) ||
		!reflect.DeepEqual(b.Categories, []string{"Science fiction", "Fantasy"}) || *b.PublishedDate != "1965" || *b.PageCount != 604 ||
		*b.ISBN13 != "9780425038918" || *b.Publisher != "Berkley" || *b.CoverURL != "https://covers.openlibrary.org/b/id/11481354-M.jpg" ||
		!strings.HasSuffix(b.InfoLink, "/works/OL893414W") || b.Description != nil {
		t.Fatalf("normalized: %+v", b)
	}
	if want := []string{"9780441013593", "9780340960196", "9780425038918"}; !reflect.DeepEqual(b.ISBN13s, want) {
		t.Fatalf("ISBN13s %q, want %q", b.ISBN13s, want)
	}
}

func TestUpstreamErrors(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"rate limited": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTooManyRequests) },
		"server error": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) },
		"bad JSON":     func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<html>`) },
	} {
		c, _ := fake(t, h, nil)
		var up *UpstreamError
		if _, _, err := c.Search(context.Background(), SearchParams{Q: "x", Page: 1, Limit: 1}); !errors.As(err, &up) {
			t.Errorf("%s: %v", name, err)
		}
	}
	c := New("http://127.0.0.1:9", "test-agent")
	var up *UpstreamError
	if _, _, err := c.Search(context.Background(), SearchParams{Q: "x", Page: 1, Limit: 1}); !errors.As(err, &up) {
		t.Fatalf("unreachable: %v", err)
	}
}

func TestWork(t *testing.T) {
	search := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("q") {
		case "key:/works/OL893414W":
			fmt.Fprintf(w, `{"docs":[%s]}`, duneDoc)
		default:
			fmt.Fprint(w, `{"docs":[]}`)
		}
	}
	work := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/works/OL893414W.json":
			fmt.Fprint(w, `{"key":"/works/OL893414W","title":"Dune","type":{"key":"/type/work"},"description":{"type":"/type/text","value":"Desert planet."}}`)
		case "/works/OL893415W.json":
			fmt.Fprint(w, `{"key":"/works/OL893415W","type":{"key":"/type/redirect"},"location":"/works/OL893414W"}`)
		case "/works/OL1W.json":
			fmt.Fprint(w, `{"key":"/works/OL1W","type":{"key":"/type/redirect"},"location":"/works/OL2W"}`)
		case "/works/OL2W.json":
			fmt.Fprint(w, `{"key":"/works/OL2W","type":{"key":"/type/redirect"},"location":"/works/OL3W"}`)
		case "/works/OL4W.json":
			fmt.Fprint(w, `{"key":"/works/OL4W","type":{"key":"/type/delete"}}`)
		case "/works/OL5W.json":
			fmt.Fprint(w, `{"key":"/works/OL5W","title":"Fresh work","subjects":["Poetry"],"covers":[7],"type":{"key":"/type/work"}}`)
		default:
			http.NotFound(w, r)
		}
	}
	c, calls := fake(t, search, work)
	ctx := context.Background()

	b, err := c.Work(ctx, "OL893414W")
	if err != nil || b.ExternalID != "OL893414W" || b.Description == nil || *b.Description != "Desert planet." || len(b.Authors) != 1 {
		t.Fatalf("work: %v %+v", err, b)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected one work and one search call, got %d", calls.Load())
	}
	// A merged work redirects once to its new ID.
	if b, err := c.Work(ctx, "OL893415W"); err != nil || b.ExternalID != "OL893414W" {
		t.Fatalf("redirect: %v %+v", err, b)
	}
	// Not indexed for search yet: built from the work record alone.
	if b, err := c.Work(ctx, "OL5W"); err != nil || b.Title != "Fresh work" || len(b.Authors) != 0 ||
		!reflect.DeepEqual(b.Categories, []string{"Poetry"}) || b.CoverURL == nil {
		t.Fatalf("unindexed: %v %+v", err, b)
	}
	for _, id := range []string{"OL1W", "OL4W", "OL404W", "OL893414M", "../x"} {
		if _, err := c.Work(ctx, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: %v", id, err)
		}
	}
}
