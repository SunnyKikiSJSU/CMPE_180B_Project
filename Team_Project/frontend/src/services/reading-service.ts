import { apiClient, apiClientPage, query } from "@/lib/api-client";
import type { Book, ProgressUpdateRequest, ReadingHistoryEvent, ReadingProgress, Shelf, ShelfBookResult } from "@/types/domain";

const id = encodeURIComponent;

export const readingService = {
  listShelves: () => apiClient<Shelf[]>("/api/shelves"),
  shelfBooks: (shelfId: string, page = 1, pageSize = 100) => apiClientPage<Book>(`/api/shelves/${id(shelfId)}/books${query({ page, pageSize })}`),
  /** On a status shelf this also sets the reading status; custom shelves only tag. Adding twice is harmless. */
  addToShelf: (shelfId: string, bookId: string) => apiClient<ShelfBookResult>(`/api/shelves/${id(shelfId)}/books`, { method: "POST", body: { bookId } }),
  removeFromShelf: (shelfId: string, bookId: string) => apiClient<void>(`/api/shelves/${id(shelfId)}/books/${id(bookId)}`, { method: "DELETE" }),
  listProgress: () => apiClient<ReadingProgress[]>("/api/reading-progress"),
  /**
   * Creates the progress row when there is none, moves the book to the status
   * shelf for a new status, and returns 409 STALE_VERSION for an old version.
   */
  updateProgress: (bookId: string, update: ProgressUpdateRequest) => apiClient<ReadingProgress>(`/api/reading-progress/${id(bookId)}`, { method: "PATCH", body: update }),
  /** Takes the book off every status shelf; history is kept. */
  deleteProgress: (bookId: string) => apiClient<void>(`/api/reading-progress/${id(bookId)}`, { method: "DELETE" }),
  history: (filters: { bookId?: string; limit?: number } = {}) => apiClient<ReadingHistoryEvent[]>(`/api/reading-history${query(filters)}`),
};
