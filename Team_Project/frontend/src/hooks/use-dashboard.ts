"use client";

import { useQuery } from "@tanstack/react-query";

import { useUserStats } from "@/hooks/use-reports";
import { queryKeys } from "@/lib/query-keys";
import { readingService } from "@/services/reading-service";

export function useDashboardData() {
  // Shares the cache entry of /reports. Its data is { data, queryId }, so the
  // dashboard reads stats.data.data.
  const stats = useUserStats();
  const reading = { status: "reading", pageSize: 3 } as const;
  const progress = useQuery({ queryKey: queryKeys.progress.list(reading), queryFn: () => readingService.listProgress(reading) });
  const shelves = useQuery({ queryKey: queryKeys.shelves.all, queryFn: readingService.listShelves });
  const history = useQuery({ queryKey: queryKeys.historyRecent(5), queryFn: () => readingService.history({ limit: 5 }) });

  const queries = [stats, progress, shelves, history];
  return {
    stats,
    progress,
    shelves,
    history,
    isPending: queries.some((query) => query.isPending),
    error: queries.find((query) => query.isError)?.error,
    retry: () => queries.filter((query) => query.isError).forEach((query) => void query.refetch()),
  };
}
