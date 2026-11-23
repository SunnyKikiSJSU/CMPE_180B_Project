//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type apiBook struct {
	ID      uuid.UUID
	Title   string
	Author  string
	Authors []struct {
		ID   uuid.UUID
		Name string
	}
	Categories []struct {
		ID   uuid.UUID
		Name string
	}
	Source        string
	InfoLink      *string
	PublishedDate *string
	PageCount     *int
	AverageRating float64
	RatingCount   int
	MyProgress    *struct {
		Status      string
		CurrentPage int
	}
	MyRating *int
}

func decode[T any](t *testing.T, body string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return v
}

func login(t *testing.T, h http.Handler, email, password string) string {
	t.Helper()
	rec := call(h, "POST", "/api/auth/login", `{"email":"`+email+`","password":"`+password+`"}`, "")
	if rec.Code != 200 {
		t.Fatalf("login: %d", rec.Code)
	}
	return decode[struct{ AccessToken string }](t, rec.Body.String()).AccessToken
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLibrarySearch(t *testing.T) {
	h, pool := setup(t)
	total := count(t, pool, "SELECT count(*) FROM books")

	rec := call(h, "GET", "/api/books?pageSize=5", "", "")
	books := decode[[]apiBook](t, rec.Body.String())
	if rec.Code != 200 || len(books) != 5 || rec.Header().Get("X-Total-Count") != strconv.Itoa(total) {
		t.Fatalf("all books: %d, %d rows, total %s (want %d)", rec.Code, len(books), rec.Header().Get("X-Total-Count"), total)
	}
	// No query: newest first.
	for i := 1; i < len(books); i++ {
		if books[i-1].PublishedDate != nil && books[i].PublishedDate != nil && *books[i-1].PublishedDate < *books[i].PublishedDate {
			t.Fatalf("not newest first: %s before %s", *books[i-1].PublishedDate, *books[i].PublishedDate)
		}
	}

	// A page past the end still reports the real total.
	rec = call(h, "GET", "/api/books?page=9999&pageSize=5", "", "")
	if rec.Code != 200 || rec.Body.String() != "[]\n" || rec.Header().Get("X-Total-Count") != strconv.Itoa(total) {
		t.Fatalf("past the end: %d %q total %s", rec.Code, rec.Body.String(), rec.Header().Get("X-Total-Count"))
	}

	// Search by a seeded title word matches that book.
	var title string
	_ = pool.QueryRow(context.Background(), "SELECT title FROM books WHERE source = 'seed' ORDER BY title LIMIT 1").Scan(&title)
	word := strings.Fields(title)[len(strings.Fields(title))-1]
	rec = call(h, "GET", "/api/books?q="+url.QueryEscape(word)+"&pageSize=100", "", "")
	found := false
	for _, b := range decode[[]apiBook](t, rec.Body.String()) {
		found = found || b.Title == title
	}
	if rec.Code != 200 || !found {
		t.Fatalf("search %q did not return %q", word, title)
	}

	// Author and category filters return only matching books, each once.
	var authorID, categoryID uuid.UUID
	_ = pool.QueryRow(context.Background(), "SELECT author_id FROM book_authors GROUP BY author_id ORDER BY count(*) DESC, author_id LIMIT 1").Scan(&authorID)
	_ = pool.QueryRow(context.Background(), "SELECT category_id FROM book_categories GROUP BY category_id ORDER BY count(*) DESC, category_id LIMIT 1").Scan(&categoryID)
	rec = call(h, "GET", "/api/books?authorId="+authorID.String()+"&pageSize=100", "", "")
	byAuthor := decode[[]apiBook](t, rec.Body.String())
	if want := count(t, pool, "SELECT count(*) FROM book_authors WHERE author_id = $1", authorID); len(byAuthor) != want || rec.Header().Get("X-Total-Count") != strconv.Itoa(want) {
		t.Fatalf("author filter: %d rows, want %d", len(byAuthor), want)
	}
	seen := map[uuid.UUID]bool{}
	for _, b := range byAuthor {
		hasAuthor := false
		for _, a := range b.Authors {
			hasAuthor = hasAuthor || a.ID == authorID
		}
		if !hasAuthor || seen[b.ID] {
			t.Fatalf("author filter returned %s (has author %v, duplicate %v)", b.Title, hasAuthor, seen[b.ID])
		}
		seen[b.ID] = true
		if b.Author != b.Authors[0].Name {
			t.Fatalf("author must be the alphabetically first name: %q vs %q", b.Author, b.Authors[0].Name)
		}
	}
	rec = call(h, "GET", "/api/books?categoryId="+categoryID.String()+"&pageSize=100", "", "")
	if want := count(t, pool, "SELECT count(*) FROM book_categories WHERE category_id = $1", categoryID); rec.Header().Get("X-Total-Count") != strconv.Itoa(want) {
		t.Fatalf("category filter total %s, want %d", rec.Header().Get("X-Total-Count"), want)
	}

	// The title filter is a case-insensitive substring match on the title.
	rec = call(h, "GET", "/api/books?title=aNCIENT&pageSize=100", "", "")
	if want := count(t, pool, "SELECT count(*) FROM books WHERE title ILIKE '%ancient%'"); want == 0 || rec.Header().Get("X-Total-Count") != strconv.Itoa(want) {
		t.Fatalf("title filter total %s, want %d", rec.Header().Get("X-Total-Count"), want)
	}

	// Keywords also match author and category names (EXISTS over the links):
	// every book by that author and in that category is found.
	for _, c := range []struct{ word, sql string }{
		{"Bronte", "SELECT DISTINCT ba.book_id FROM book_authors ba JOIN authors a USING (author_id) WHERE a.name ILIKE '%bronte%'"},
		{"Fantasy", "SELECT DISTINCT bc.book_id FROM book_categories bc JOIN categories c USING (category_id) WHERE c.name ILIKE '%fantasy%'"},
	} {
		rec := call(h, "GET", "/api/books?q="+c.word+"&pageSize=100", "", "")
		got := map[uuid.UUID]bool{}
		for _, b := range decode[[]apiBook](t, rec.Body.String()) {
			got[b.ID] = true
		}
		rows, err := pool.Query(context.Background(), c.sql)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		for rows.Next() {
			var id uuid.UUID
			_ = rows.Scan(&id)
			want++
			if !got[id] {
				t.Fatalf("q=%s misses book %s", c.word, id)
			}
		}
		rows.Close()
		if want == 0 {
			t.Fatalf("q=%s: no seed books to check", c.word)
		}
	}

	// LIKE wildcards in user text match literally.
	if rec := call(h, "GET", "/api/books?q=%25%25%25&pageSize=1", "", ""); rec.Code != 200 || rec.Header().Get("X-Total-Count") != "0" {
		t.Fatalf("wildcard: %s", rec.Header().Get("X-Total-Count"))
	}
	for _, sort := range []string{"relevance", "newest", "rating", "title", "popular"} {
		if rec := call(h, "GET", "/api/books?q=the&sort="+sort, "", ""); rec.Code != 200 {
			t.Fatalf("sort %s: %d", sort, rec.Code)
		}
	}
}

func TestBookDetailWithPersonalContext(t *testing.T) {
	h, pool := setup(t)
	var bookID uuid.UUID
	var value int
	err := pool.QueryRow(context.Background(), `
		SELECT r.book_id, r.rating_value FROM ratings r JOIN users u ON u.user_id = r.user_id
		WHERE u.email = 'reader1@example.com' LIMIT 1`).Scan(&bookID, &value)
	if err != nil {
		t.Skip("reader1 has no ratings in this seed")
	}

	anon := decode[apiBook](t, call(h, "GET", "/api/books/"+bookID.String(), "", "").Body.String())
	if anon.MyRating != nil || anon.MyProgress != nil || anon.Source != "seed" || anon.InfoLink != nil || len(anon.Authors) == 0 {
		t.Fatalf("anonymous detail: %+v", anon)
	}
	token := login(t, h, "reader1@example.com", "Password123!")
	mine := decode[apiBook](t, call(h, "GET", "/api/books/"+bookID.String(), "", token).Body.String())
	if mine.MyRating == nil || *mine.MyRating != value {
		t.Fatalf("myRating: %v, want %d", mine.MyRating, value)
	}
	if rec := call(h, "GET", "/api/books/"+uuid.NewString(), "", ""); rec.Code != 404 {
		t.Fatalf("unknown book: %d", rec.Code)
	}
}

func TestCreateAndUpdateBook(t *testing.T) {
	h, pool := setup(t)
	ctx := context.Background()
	token := login(t, h, "reader1@example.com", "Password123!")
	tag := uuid.NewString()[:8]
	isbn := "97" + strconv.FormatInt(int64(uuid.New().ID())%100000000000, 10)
	isbn += strings.Repeat("0", 13-len(isbn))
	var created []uuid.UUID
	t.Cleanup(func() {
		for _, id := range created {
			_, _ = pool.Exec(ctx, "DELETE FROM books WHERE book_id = $1", id)
		}
		_, _ = pool.Exec(ctx, "DELETE FROM authors WHERE name LIKE $1", "IT Author "+tag+"%")
		_, _ = pool.Exec(ctx, "DELETE FROM categories WHERE name LIKE $1", "IT Category "+tag+"%")
	})

	// A book without authors reads back as "Unknown author" (primary_author is
	// coalesced in SQL; before, the NULL failed to scan into a string).
	rec := call(h, "POST", "/api/books", `{"title":"IT no-author `+tag+`"}`, token)
	bare := decode[apiBook](t, rec.Body.String())
	created = append(created, bare.ID)
	if rec.Code != 201 || bare.Author != "Unknown author" || len(bare.Authors) != 0 || bare.Source != "user" {
		t.Fatalf("no-author book: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "GET", "/api/books/"+bare.ID.String(), "", ""); rec.Code != 200 {
		t.Fatalf("no-author detail: %d %s", rec.Code, rec.Body.String())
	}

	body := `{"title":"  IT Book ` + tag + ` ","authors":["IT Author ` + tag + ` B","it author ` + tag + ` b","IT Author ` + tag + ` A"],` +
		`"categories":["IT Category ` + tag + `"],"pageCount":321,"isbn13":"` + isbn + `","publishedDate":"2020-05-17","coverUrl":"https://example.com/c.jpg"}`
	rec = call(h, "POST", "/api/books", body, token)
	book := decode[apiBook](t, rec.Body.String())
	created = append(created, book.ID)
	if rec.Code != 201 || book.Title != "IT Book "+tag || len(book.Authors) != 2 || book.Author != "IT Author "+tag+" A" || len(book.Categories) != 1 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	if book.PublishedDate == nil || *book.PublishedDate != "2020-05-17" || book.PageCount == nil || *book.PageCount != 321 {
		t.Fatalf("fields: %s", rec.Body.String())
	}

	// Same ISBN-13 again: 409.
	if rec := call(h, "POST", "/api/books", `{"title":"dup","isbn13":"`+isbn+`"}`, token); rec.Code != 409 || !strings.Contains(rec.Body.String(), "books_isbn13_unique") {
		t.Fatalf("duplicate ISBN: %d %s", rec.Code, rec.Body.String())
	}
	// A failed create leaves nothing behind (one transaction).
	if n := count(t, pool, "SELECT count(*) FROM books WHERE title = 'dup'"); n != 0 {
		t.Fatalf("rolled-back book persisted: %d", n)
	}

	rec = call(h, "PATCH", "/api/books/"+book.ID.String(), `{"title":"IT Book `+tag+` (2nd ed.)","pageCount":400}`, token)
	updated := decode[apiBook](t, rec.Body.String())
	if rec.Code != 200 || updated.Title != "IT Book "+tag+" (2nd ed.)" || *updated.PageCount != 400 || len(updated.Authors) != 2 {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}

	// Google rows are read-only.
	var googleID uuid.UUID
	err := pool.QueryRow(ctx, `INSERT INTO books (google_books_id, title, source, imported_at) VALUES ($1, $2, 'google', now()) RETURNING book_id`,
		"it-"+tag, "IT Google "+tag).Scan(&googleID)
	if err != nil {
		t.Fatal(err)
	}
	created = append(created, googleID)
	if rec := call(h, "PATCH", "/api/books/"+googleID.String(), `{"title":"x"}`, token); rec.Code != 403 || !strings.Contains(rec.Body.String(), "FORBIDDEN") {
		t.Fatalf("google update: %d %s", rec.Code, rec.Body.String())
	}
	g := decode[apiBook](t, call(h, "GET", "/api/books/"+googleID.String(), "", "").Body.String())
	if g.InfoLink == nil || *g.InfoLink != "https://books.google.com/books?id=it-"+tag {
		t.Fatalf("google infoLink: %v", g.InfoLink)
	}

	if rec := call(h, "POST", "/api/books", `{"title":"x","pageCount":0}`, token); rec.Code != 400 {
		t.Fatalf("invalid page count: %d", rec.Code)
	}
}

func TestAuthorsAndCategories(t *testing.T) {
	h, pool := setup(t)
	rec := call(h, "GET", "/api/authors?pageSize=10", "", "")
	authors := decode[[]struct {
		ID        uuid.UUID
		Name      string
		BookCount int
	}](t, rec.Body.String())
	if rec.Code != 200 || len(authors) != 10 || rec.Header().Get("X-Total-Count") != strconv.Itoa(count(t, pool, "SELECT count(*) FROM authors")) {
		t.Fatalf("authors: %d %d %s", rec.Code, len(authors), rec.Header().Get("X-Total-Count"))
	}
	for i := 1; i < len(authors); i++ {
		if strings.Compare(authors[i-1].Name, authors[i].Name) > 0 && strings.ToLower(authors[i-1].Name) > strings.ToLower(authors[i].Name) {
			t.Fatalf("authors not sorted by name: %q, %q", authors[i-1].Name, authors[i].Name)
		}
	}
	first := authors[0]
	rec = call(h, "GET", "/api/authors/"+first.ID.String()+"?pageSize=100", "", "")
	detail := decode[struct {
		Name      string
		BookCount int
		Books     []apiBook
	}](t, rec.Body.String())
	if rec.Code != 200 || detail.Name != first.Name || len(detail.Books) != first.BookCount || detail.BookCount != first.BookCount {
		t.Fatalf("author detail: %d books %d, list said %d", rec.Code, len(detail.Books), first.BookCount)
	}
	if rec := call(h, "GET", "/api/authors?q="+url.QueryEscape(first.Name[:3]), "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), first.Name) {
		t.Fatalf("author search: %d", rec.Code)
	}

	rec = call(h, "GET", "/api/categories?pageSize=100", "", "")
	categories := decode[[]struct {
		ID        uuid.UUID
		BookCount int
	}](t, rec.Body.String())
	if rec.Code != 200 || len(categories) != count(t, pool, "SELECT count(*) FROM categories") {
		t.Fatalf("categories: %d %d", rec.Code, len(categories))
	}
	rec = call(h, "GET", "/api/categories/"+categories[0].ID.String()+"?pageSize=5", "", "")
	if rec.Code != 200 || rec.Header().Get("X-Total-Count") != strconv.Itoa(categories[0].BookCount) {
		t.Fatalf("category detail total %s, want %d", rec.Header().Get("X-Total-Count"), categories[0].BookCount)
	}
	if rec := call(h, "GET", "/api/categories/"+uuid.NewString(), "", ""); rec.Code != 404 {
		t.Fatalf("unknown category: %d", rec.Code)
	}
}
