"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { queryKeys } from "@/lib/query-keys";
import { reportService, type RankingParams, type WindowParams } from "@/services/report-service";

// Reports are aggregates over everyone's activity; a minute of staleness is fine.
const staleTime = 60_000;

export function useUserStats() {
  return useQuery({ queryKey: queryKeys.reports("user-stats"), queryFn: reportService.userStats, staleTime });
}

export function useReadingReport() {
  return useQuery({ queryKey: queryKeys.reports("reading-history"), queryFn: reportService.readingHistory, staleTime });
}

export function useRecommendations(limit = 10) {
  return useQuery({ queryKey: queryKeys.reports("recommendations", { limit }), queryFn: () => reportService.recommendations(limit), staleTime });
}

export function usePopular(params: WindowParams) {
  return useQuery({ queryKey: queryKeys.reports("popular", params), queryFn: () => reportService.popular(params), staleTime, placeholderData: keepPreviousData });
}

export function useTopRated(params: RankingParams) {
  return useQuery({ queryKey: queryKeys.reports("top-rated", params), queryFn: () => reportService.topRated(params), staleTime, placeholderData: keepPreviousData });
}

export function useAuthorRankings(params: RankingParams) {
  return useQuery({ queryKey: queryKeys.reports("authors", params), queryFn: () => reportService.authorRankings(params), staleTime, placeholderData: keepPreviousData });
}

export function useCategoryRankings(params: RankingParams) {
  return useQuery({ queryKey: queryKeys.reports("categories", params), queryFn: () => reportService.categoryRankings(params), staleTime, placeholderData: keepPreviousData });
}

export function useReviewActivity(params: Omit<WindowParams, "limit">) {
  return useQuery({ queryKey: queryKeys.reports("review-activity", params), queryFn: () => reportService.reviewActivity(params), staleTime, placeholderData: keepPreviousData });
}

/** The SQL text never changes while the API runs. */
export function useQueryCatalog() {
  return useQuery({ queryKey: queryKeys.reports("queries"), queryFn: reportService.queries, staleTime: Infinity });
}
