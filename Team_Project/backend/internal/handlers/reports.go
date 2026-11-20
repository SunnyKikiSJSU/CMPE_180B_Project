package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
)

type ReportService interface {
	ReadingReport(ctx context.Context, userID uuid.UUID) (domain.ReadingReport, error)
	UserStats(ctx context.Context, userID uuid.UUID) (domain.UserStatsReport, error)
	Popular(ctx context.Context, w domain.Window, limit int) ([]domain.PopularBook, error)
	TopRated(ctx context.Context, minRatings, limit int) ([]domain.TopRatedBook, error)
	Recommendations(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Recommendation, error)
	AuthorRankings(ctx context.Context, minRatings, limit int) ([]domain.AuthorRanking, error)
	CategoryRankings(ctx context.Context, minRatings, limit int) ([]domain.CategoryRanking, error)
	ReviewActivity(ctx context.Context, w domain.Window) ([]domain.ActivityPoint, error)
	Catalog() []domain.QueryCatalogEntry
}

// Reports serves /api/reports/*. Every report response carries X-Query-Id,
// the evidence ID of its SQL in GET /api/reports/queries.
type Reports struct {
	Service ReportService
	// Now is the clock for the default endDate; nil means time.Now.
	Now func() time.Time
}

// intQuery reads an optional whole-number query parameter within [min, max].
func intQuery(r *http.Request, name string, def, min, max int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		return 0, domain.Validation(name, name+" must be a whole number from "+strconv.Itoa(min)+" to "+strconv.Itoa(max)+".")
	}
	return n, nil
}

// window reads ?days (1-365, default 30) and ?endDate (YYYY-MM-DD, default
// today in UTC). endDate may be at most one day ahead of UTC so a caller east
// of UTC can ask for its own "today".
func (h Reports) window(r *http.Request) (domain.Window, error) {
	days, err := intQuery(r, "days", 30, 1, 365)
	if err != nil {
		return domain.Window{}, err
	}
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	today := now().UTC().Truncate(24 * time.Hour)
	end := today
	if raw := r.URL.Query().Get("endDate"); raw != "" {
		end, err = time.Parse(time.DateOnly, raw)
		if err != nil {
			return domain.Window{}, domain.Validation("endDate", "endDate must be a date in YYYY-MM-DD format.")
		}
		if end.After(today.AddDate(0, 0, 1)) {
			return domain.Window{}, domain.Validation("endDate", "endDate cannot be in the future.")
		}
	}
	return domain.Window{Days: days, EndDate: end}, nil
}

// defaultRankingMinRatings keeps authors and categories with only a few
// ratings off the leaderboards unless the caller asks for them.
const defaultRankingMinRatings = 5

func limitQuery(r *http.Request, def int) (int, error) { return intQuery(r, "limit", def, 1, 100) }

func report(w http.ResponseWriter, r *http.Request, queryID string, body any, err error) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.Header().Set("X-Query-Id", queryID)
	httpx.JSON(w, http.StatusOK, body)
}

func (h Reports) ReadingHistory(w http.ResponseWriter, r *http.Request) {
	out, err := h.Service.ReadingReport(r.Context(), caller(r))
	report(w, r, "q07", out, err)
}

func (h Reports) UserStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.Service.UserStats(r.Context(), caller(r))
	report(w, r, "q07", out, err)
}

func (h Reports) Popular(w http.ResponseWriter, r *http.Request) {
	win, err := h.window(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, err := limitQuery(r, 10)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out, err := h.Service.Popular(r.Context(), win, limit)
	report(w, r, "q05", out, err)
}

func (h Reports) TopRated(w http.ResponseWriter, r *http.Request) {
	minRatings, err := intQuery(r, "minRatings", 1, 0, 1000)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, err := limitQuery(r, 10)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out, err := h.Service.TopRated(r.Context(), minRatings, limit)
	report(w, r, "q04", out, err)
}

func (h Reports) Recommendations(w http.ResponseWriter, r *http.Request) {
	limit, err := limitQuery(r, 10)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out, err := h.Service.Recommendations(r.Context(), caller(r), limit)
	report(w, r, "q06", out, err)
}

func (h Reports) Authors(w http.ResponseWriter, r *http.Request) {
	minRatings, err := intQuery(r, "minRatings", defaultRankingMinRatings, 0, 1000)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, err := limitQuery(r, 20)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out, err := h.Service.AuthorRankings(r.Context(), minRatings, limit)
	report(w, r, "q08", out, err)
}

func (h Reports) Categories(w http.ResponseWriter, r *http.Request) {
	minRatings, err := intQuery(r, "minRatings", defaultRankingMinRatings, 0, 1000)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, err := limitQuery(r, 20)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out, err := h.Service.CategoryRankings(r.Context(), minRatings, limit)
	report(w, r, "q08", out, err)
}

func (h Reports) ReviewActivity(w http.ResponseWriter, r *http.Request) {
	win, err := h.window(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out, err := h.Service.ReviewActivity(r.Context(), win)
	report(w, r, "q09", out, err)
}

func (h Reports) Queries(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, h.Service.Catalog())
}
