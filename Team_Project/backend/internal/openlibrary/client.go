// Package openlibrary searches Open Library (openlibrary.org) and reads works
// for import. Its catalog data is openly reusable
// (https://openlibrary.org/developers/licensing); covers are linked from
// covers.openlibrary.org, never copied, as its cover guidelines ask.
package openlibrary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"book-metaverse/backend/internal/domain"
)

// ErrNotFound means Open Library has no such work (or it was deleted).
var ErrNotFound = errors.New("openlibrary: work not found")

// UpstreamError is any other failure: timeout, rate limit, 5xx, bad JSON.
type UpstreamError struct {
	Op     string
	Status int
	Err    error
}

func (e *UpstreamError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("openlibrary %s: HTTP %d", e.Op, e.Status)
	}
	return fmt.Sprintf("openlibrary %s: %v", e.Op, e.Err)
}

func (e *UpstreamError) Unwrap() error { return e.Err }

const (
	defaultCoversURL = "https://covers.openlibrary.org"
	// requestTimeout stays under the API's 10-second request budget.
	requestTimeout = 9 * time.Second
	maxBodyBytes   = 8 << 20
)

type Client struct {
	siteURL   string
	coversURL string
	userAgent string
	http      *http.Client
}

// New returns a client for baseURL (https://openlibrary.org in production).
func New(baseURL, userAgent string) *Client {
	return &Client{
		siteURL:   strings.TrimRight(baseURL, "/"),
		coversURL: defaultCoversURL,
		userAgent: userAgent,
		http:      &http.Client{Timeout: requestTimeout},
	}
}

// SearchParams is one search page. At least one text field is set.
type SearchParams struct {
	Q, Title, Author, Subject, ISBN string
	// Newest sorts by publication date instead of relevance.
	Newest      bool
	Page, Limit int
}

// Search returns the total number of matches and one page of works.
func (c *Client) Search(ctx context.Context, p SearchParams) (int64, []domain.ExternalBook, error) {
	query := url.Values{}
	for key, value := range map[string]string{"q": p.Q, "title": p.Title, "author": p.Author, "subject": p.Subject, "isbn": p.ISBN} {
		if value != "" {
			query.Set(key, value)
		}
	}
	if p.Newest {
		query.Set("sort", "new")
	}
	query.Set("page", strconv.Itoa(p.Page))
	query.Set("limit", strconv.Itoa(p.Limit))
	query.Set("fields", searchFields)
	var res struct {
		NumFound int64 `json:"numFound"`
		Docs     []doc `json:"docs"`
	}
	if err := c.get(ctx, "search", "/search.json", query, &res); err != nil {
		return 0, nil, err
	}
	books := make([]domain.ExternalBook, 0, len(res.Docs))
	for _, d := range res.Docs {
		if b, ok := c.normalize(d); ok {
			books = append(books, b)
		}
	}
	return res.NumFound, books, nil
}

type work struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Subtitle    string   `json:"subtitle"`
	Description any      `json:"description"`
	Covers      []int64  `json:"covers"`
	Subjects    []string `json:"subjects"`
	Type        struct {
		Key string `json:"key"`
	} `json:"type"`
	Location string `json:"location"`
}

// Work reads one work for import: the work record (description, redirects)
// and its search document (authors, subjects, pages, edition ISBN) are
// fetched in parallel. A merged work redirects to its new ID, which is
// followed once; the returned ExternalID is the current one.
func (c *Client) Work(ctx context.Context, id string) (domain.ExternalBook, error) {
	if !WorkIDPattern.MatchString(id) {
		return domain.ExternalBook{}, ErrNotFound
	}
	for attempt := 0; ; attempt++ {
		w, d, found, err := c.fetchWork(ctx, id)
		if err != nil {
			return domain.ExternalBook{}, err
		}
		switch w.Type.Key {
		case "/type/redirect":
			next := strings.TrimPrefix(w.Location, "/works/")
			if attempt > 0 || !WorkIDPattern.MatchString(next) {
				return domain.ExternalBook{}, ErrNotFound
			}
			id = next
			continue
		case "/type/delete":
			return domain.ExternalBook{}, ErrNotFound
		}
		if !found {
			// Not in the search index yet: use what the work record has.
			d = doc{Key: "/works/" + id, Title: w.Title, Subtitle: w.Subtitle, Subject: w.Subjects}
			if len(w.Covers) > 0 {
				d.CoverID = w.Covers[0]
			}
		}
		if d.Title == "" {
			d.Title = w.Title
		}
		b, ok := c.normalize(d)
		if !ok {
			return domain.ExternalBook{}, ErrNotFound
		}
		b.Description = description(w.Description)
		return b, nil
	}
}

func (c *Client) fetchWork(ctx context.Context, id string) (work, doc, bool, error) {
	type searchResult struct {
		docs []doc
		err  error
	}
	ch := make(chan searchResult, 1)
	go func() {
		var res struct {
			Docs []doc `json:"docs"`
		}
		query := url.Values{"q": {"key:/works/" + id}, "limit": {"1"}, "fields": {searchFields}}
		err := c.get(ctx, "work search", "/search.json", query, &res)
		ch <- searchResult{res.Docs, err}
	}()
	var w work
	werr := c.get(ctx, "work", "/works/"+id+".json", nil, &w)
	s := <-ch
	if werr != nil {
		return work{}, doc{}, false, werr
	}
	if w.Type.Key == "/type/redirect" || w.Type.Key == "/type/delete" {
		return w, doc{}, false, nil
	}
	if s.err != nil {
		return work{}, doc{}, false, s.err
	}
	for _, d := range s.docs {
		if strings.TrimPrefix(d.Key, "/works/") == id {
			return w, d, true, nil
		}
	}
	return w, doc{}, false, nil
}

func (c *Client) get(ctx context.Context, op, path string, query url.Values, out any) error {
	u := c.siteURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return &UpstreamError{Op: op, Err: err}
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return &UpstreamError{Op: op, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return &UpstreamError{Op: op, Status: resp.StatusCode}
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(out); err != nil {
		return &UpstreamError{Op: op, Err: fmt.Errorf("decode: %w", err)}
	}
	return nil
}
