import { apiClient } from "@/lib/api-client";
import type { ReadingReport, UserStatsReport } from "@/types/domain";

export const reportService = {
  userStats: () => apiClient<UserStatsReport>("/api/reports/user-stats"),
  readingHistory: () => apiClient<ReadingReport>("/api/reports/reading-history"),
};
