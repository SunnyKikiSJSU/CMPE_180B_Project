import { apiClient, apiClientPage, query } from "@/lib/api-client";
import type { Book, ProgressUpdateRequest, ReadingHistoryEvent, ReadingProgress, ReadingStatus, Shelf, ShelfBookResult } from "@/types/domain";

const id = encodeURIComponent;

export type ProgressFilters = { status?: ReadingStatus; page?: number; pageSize?: number };

export const readingService = {
  listShelves: () => apiClient<Shelf[]>("/api/shelves"),
  createShelf: (name: string) => apiClient<Shelf>("/api/shelves", { method: "POST", body: { name } }),
  renameShelf: (shelfId: string, name: string) => apiClient<Shelf>(`/api/shelves/${id(shelfId)}`, { method: "PATCH", body: { name } }),
  deleteShelf: (shelfId: string) => apiClient<void>(`/api/shelves/${id(shelfId)}`, { method: "DELETE" }),
  shelfBooks: (shelfId: string, page = 1, pageSize = 100, sort: "recent" | "title" = "recent") => apiClientPage<Book>(`/api/shelves/${id(shelfId)}/books${query({ page, pageSize, sort })}`),
  /** On a status shelf this also sets the reading status; custom shelves only tag. Adding twice is harmless. */
  addToShelf: (shelfId: string, bookId: string) => apiClient<ShelfBookResult>(`/api/shelves/${id(shelfId)}/books`, { method: "POST", body: { bookId } }),
  moveShelfBook: (shelfId: string, bookId: string, targetShelfId: string) => apiClient<ShelfBookResult>(`/api/shelves/${id(shelfId)}/books/${id(bookId)}`, { method: "PATCH", body: { targetShelfId } }),
  removeFromShelf: (shelfId: string, bookId: string) => apiClient<void>(`/api/shelves/${id(shelfId)}/books/${id(bookId)}`, { method: "DELETE" }),
  /** Paginated (the API defaults to 20 per page); `status` filters on the server. */
  listProgress: (filters: ProgressFilters = {}) => apiClientPage<ReadingProgress>(`/api/reading-progress${query(filters)}`),
  /**
   * Creates the progress row when there is none, moves the book to the status
   * shelf for a new status, and returns 409 STALE_VERSION for an old version.
   */
  updateProgress: (bookId: string, update: ProgressUpdateRequest) => apiClient<ReadingProgress>(`/api/reading-progress/${id(bookId)}`, { method: "PATCH", body: update }),
  /** Takes the book off every status shelf; history is kept. */
  deleteProgress: (bookId: string) => apiClient<void>(`/api/reading-progress/${id(bookId)}`, { method: "DELETE" }),
  history: (filters: { bookId?: string; limit?: number } = {}) => apiClient<ReadingHistoryEvent[]>(`/api/reading-history${query(filters)}`),
};
