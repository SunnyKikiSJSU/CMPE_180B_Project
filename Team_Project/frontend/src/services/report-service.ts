import { apiClient, apiClientWithHeaders, query } from "@/lib/api-client";
import type {
  AuthorRanking,
  CategoryRanking,
  PopularBook,
  QueryCatalogEntry,
  ReadingReport,
  Recommendation,
  ReportResult,
  ReviewActivityPoint,
  TopRatedBook,
  UserStatsReport,
} from "@/types/domain";

/** Every report response names the SQL that produced it in X-Query-Id. */
async function report<T>(path: string): Promise<ReportResult<T>> {
  const { data, headers } = await apiClientWithHeaders<T>(path);
  return { data, queryId: headers.get("X-Query-Id") };
}

// Parameter ranges (handlers/reports.go): days 1-365, limit 1-100, minRatings 0-1000.
export type WindowParams = { days?: number; endDate?: string; limit?: number };
export type RankingParams = { minRatings?: number; limit?: number };

export const reportService = {
  /** Signed-in only. */
  readingHistory: () => report<ReadingReport>("/api/reports/reading-history"),
  /** Signed-in only. */
  userStats: () => report<UserStatsReport>("/api/reports/user-stats"),
  /** Signed-in only. */
  recommendations: (limit = 10) => report<Recommendation[]>(`/api/reports/recommendations${query({ limit })}`),
  popular: (params: WindowParams = {}) => report<PopularBook[]>(`/api/reports/popular${query(params)}`),
  topRated: (params: RankingParams = {}) => report<TopRatedBook[]>(`/api/reports/top-rated${query(params)}`),
  authorRankings: (params: RankingParams = {}) => report<AuthorRanking[]>(`/api/reports/authors${query(params)}`),
  categoryRankings: (params: RankingParams = {}) => report<CategoryRanking[]>(`/api/reports/categories${query(params)}`),
  reviewActivity: (params: Omit<WindowParams, "limit"> = {}) => report<ReviewActivityPoint[]>(`/api/reports/review-activity${query(params)}`),
  /** Fixed SQL text of every evidence query (q01-q10); never user input. */
  queries: () => apiClient<QueryCatalogEntry[]>("/api/reports/queries"),
};
