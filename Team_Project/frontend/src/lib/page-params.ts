// URL state for the /reviews, /authors, and /reports pages, so tabs, searches,
// and pages survive a reload and work with the back button.

interface ParamReader { get(name: string): string | null; }

/** The API allows pages 1..10000 (handlers/params.go). */
const MAX_PAGE = 10000;

export function pageParam(raw: string | null): number {
  const n = Number(raw);
  return Number.isInteger(n) && n >= 1 && n <= MAX_PAGE ? n : 1;
}

function oneOf<T extends string | number>(raw: string | null, options: readonly T[], fallback: T): T {
  return options.find((o) => String(o) === raw) ?? fallback;
}

/** `path` with the non-default values as a query string. */
export function hrefWith(path: string, params: Record<string, string | number | undefined>, defaults: Record<string, string | number | undefined> = {}): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === "" || value === defaults[key]) continue;
    search.set(key, String(value));
  }
  const s = search.toString();
  return s ? `${path}?${s}` : path;
}

// --- /reviews ---------------------------------------------------------------

export type ReviewsTab = "reviews" | "ratings";
export interface ReviewsState { tab: ReviewsTab; page: number; }

export function parseReviewsParams(params: ParamReader): ReviewsState {
  return { tab: oneOf<ReviewsTab>(params.get("tab"), ["reviews", "ratings"], "reviews"), page: pageParam(params.get("page")) };
}

export const reviewsHref = (state: Partial<ReviewsState>) => hrefWith("/reviews", { tab: state.tab, page: state.page }, { tab: "reviews", page: 1 });

// --- /authors ---------------------------------------------------------------

export type NamesTab = "authors" | "categories";
export interface NamesState { tab: NamesTab; q: string; page: number; }

export function parseNamesParams(params: ParamReader): NamesState {
  return {
    tab: oneOf<NamesTab>(params.get("tab"), ["authors", "categories"], "authors"),
    q: (params.get("q") ?? "").trim().slice(0, 200),
    page: pageParam(params.get("page")),
  };
}

export const namesHref = (state: Partial<NamesState>) => hrefWith("/authors", { tab: state.tab, q: state.q, page: state.page }, { tab: "authors", page: 1 });

/** An author's or category's own page (its books). */
export const authorHref = (id: string) => `/authors/${encodeURIComponent(id)}`;
export const categoryHref = (id: string) => `/categories/${encodeURIComponent(id)}`;

// --- /reports ---------------------------------------------------------------

export const reportTabs = [
  { id: "mine", label: "My reading" },
  { id: "popular", label: "Popular" },
  { id: "top-rated", label: "Top rated" },
  { id: "authors", label: "Author ranking" },
  { id: "categories", label: "Category ranking" },
  { id: "activity", label: "Review activity" },
  { id: "queries", label: "SQL queries" },
] as const;
export type ReportTab = (typeof reportTabs)[number]["id"];

export const dayOptions = [7, 30, 90, 365] as const;
export const minRatingOptions = [0, 1, 3, 5, 10] as const;
const DEFAULT_DAYS = 30;

/** The API's own default for minRatings (handlers/reports.go): 1 for top-rated, 5 for rankings. */
export function defaultMinRatings(tab: ReportTab): number {
  return tab === "top-rated" ? 1 : 5;
}

export interface ReportState {
  tab: ReportTab;
  /** popular and activity: a window of whole UTC days ending today. */
  days: number;
  /** top-rated and rankings. */
  minRatings: number;
  /** queries: the evidence query to open ("q05"), or "". */
  sql: string;
}

export function parseReportParams(params: ParamReader): ReportState {
  const tab = oneOf<ReportTab>(params.get("tab"), reportTabs.map((t) => t.id), "mine");
  const sql = params.get("sql") ?? "";
  return {
    tab,
    days: oneOf<number>(params.get("days"), dayOptions, DEFAULT_DAYS),
    minRatings: oneOf<number>(params.get("minRatings"), minRatingOptions, defaultMinRatings(tab)),
    sql: tab === "queries" && /^q\d{2}$/.test(sql) ? sql : "",
  };
}

/** Only the parameters the tab uses are kept, so switching tabs resets them. */
export function reportHref(state: Partial<ReportState>): string {
  const tab = state.tab ?? "mine";
  const usesDays = tab === "popular" || tab === "activity";
  const usesMin = tab === "top-rated" || tab === "authors" || tab === "categories";
  return hrefWith(
    "/reports",
    { tab, days: usesDays ? state.days : undefined, minRatings: usesMin ? state.minRatings : undefined, sql: tab === "queries" ? state.sql : undefined },
    { tab: "mine", days: DEFAULT_DAYS, minRatings: defaultMinRatings(tab) },
  );
}
