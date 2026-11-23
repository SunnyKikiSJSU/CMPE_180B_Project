import { apiClient } from "@/lib/api-client";
import type { ProgressUpdateRequest, ReadingProgress, ReadingReport, Shelf } from "@/types/domain";

export const readingService = {
  getShelves: () => apiClient<Shelf[]>("/api/shelves"),
  getProgress: () => apiClient<ReadingProgress[]>("/api/reading-progress"),
  updateProgress: (bookId: string, progress: ProgressUpdateRequest) => apiClient<ReadingProgress>(`/api/reading-progress/${bookId}`, { method: "PATCH", body: progress }),
  getReport: () => apiClient<ReadingReport>("/api/reports/reading-history"),
};
