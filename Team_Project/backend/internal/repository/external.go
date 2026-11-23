package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

// ExternalMatches maps outside search results to local books.
type ExternalMatches struct {
	ByWorkID map[string]uuid.UUID
	ByISBN13 map[string]uuid.UUID
}

// MatchExternal finds local books for a page of Open Library works, by work ID
// or by any ISBN-13 of the work (ListBooksByExternalIDs).
func (c *Catalog) MatchExternal(ctx context.Context, workIDs, isbn13s []string) (ExternalMatches, error) {
	out := ExternalMatches{ByWorkID: map[string]uuid.UUID{}, ByISBN13: map[string]uuid.UUID{}}
	rows, err := c.q.ListBooksByExternalIDs(ctx, db.ListBooksByExternalIDsParams{OpenlibraryIds: workIDs, Isbn13s: isbn13s})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.OpenlibraryID.Valid {
			out.ByWorkID[r.OpenlibraryID.String] = r.BookID
		}
		if r.Isbn13.Valid {
			out.ByISBN13[r.Isbn13.String] = r.BookID
		}
	}
	return out, nil
}

// FindImported returns the local book for a work (by work ID, then ISBN-13);
// ErrNoRows when there is none.
func (c *Catalog) FindImported(ctx context.Context, workID string, isbn13s []string) (uuid.UUID, error) {
	if isbn13s == nil {
		isbn13s = []string{}
	}
	r, err := c.q.FindImportedBook(ctx, db.FindImportedBookParams{OpenlibraryID: workID, Isbn13s: isbn13s})
	if err != nil {
		return uuid.Nil, noRows(err)
	}
	return r.BookID, nil
}

// ImportOpenLibrary inserts the work as a source = 'openlibrary' row, or
// returns the existing row for the same work (inserted = false). Call inside
// InTx together with SetAuthors and SetCategories.
func (c *Catalog) ImportOpenLibrary(ctx context.Context, b domain.ExternalBook) (uuid.UUID, bool, error) {
	params := db.ImportOpenLibraryBookParams{
		OpenlibraryID: b.ExternalID, Isbn13: pgText(b.ISBN13), Title: b.Title, Description: pgText(b.Description),
		Publisher: pgText(b.Publisher), PageCount: pgInt4(b.PageCount), CoverUrl: pgText(b.CoverURL),
	}
	// Open Library gives the first publication year only.
	if b.PublishedDate != nil {
		if year, err := strconv.Atoi(*b.PublishedDate); err == nil {
			params.PublishedDate = pgtype.Date{Time: time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
			params.PublishedDatePrecision = pgtype.Text{String: "year", Valid: true}
		}
	}
	r, err := c.q.ImportOpenLibraryBook(ctx, params)
	return r.BookID, r.Inserted, err
}
