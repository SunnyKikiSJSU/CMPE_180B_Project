"use client";

import { useMutation, useQueries, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { isApiError } from "@/lib/errors";
import { queryKeys } from "@/lib/query-keys";
import { findInPages } from "@/services/review-service";
import { readingService } from "@/services/reading-service";
import type { Book, ProgressUpdateRequest, ReadingProgress, Shelf, ShelfBookResult } from "@/types/domain";

/** Everything a status, progress, or shelf change can affect. */
function refreshReading(client: QueryClient, bookId: string) {
  for (const queryKey of [queryKeys.books.detail(bookId), queryKeys.progress.all, queryKeys.shelves.all, queryKeys.history, ["reports"]]) {
    void client.invalidateQueries({ queryKey });
  }
}

function setMyProgress(client: QueryClient, bookId: string, progress: ReadingProgress | null) {
  client.setQueryData<Book>(queryKeys.books.detail(bookId), (book) => (book ? { ...book, myProgress: progress } : book));
}

export function useShelves() {
  return useQuery({ queryKey: queryKeys.shelves.all, queryFn: readingService.listShelves });
}

export function useShelfBooks(shelfId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.shelves.books(shelfId ?? ""),
    queryFn: () => readingService.shelfBooks(shelfId ?? ""),
    enabled: Boolean(shelfId),
  });
}

function refreshShelves(client: QueryClient) {
  void client.invalidateQueries({ queryKey: queryKeys.shelves.all });
  void client.invalidateQueries({ queryKey: ["shelves"] });
  void client.invalidateQueries({ queryKey: queryKeys.progress.all });
  void client.invalidateQueries({ queryKey: queryKeys.history });
  void client.invalidateQueries({ queryKey: ["reports"] });
}

export function useCreateShelf() {
  const client = useQueryClient();
  return useMutation({ mutationFn: (name: string) => readingService.createShelf(name), onSuccess: () => refreshShelves(client) });
}

export function useRenameShelf() {
  const client = useQueryClient();
  return useMutation({ mutationFn: ({ shelfId, name }: { shelfId: string; name: string }) => readingService.renameShelf(shelfId, name), onSuccess: () => refreshShelves(client) });
}

export function useDeleteShelf() {
  const client = useQueryClient();
  return useMutation({ mutationFn: (shelfId: string) => readingService.deleteShelf(shelfId), onSuccess: () => refreshShelves(client) });
}

export function useMoveShelfBook() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ shelfId, bookId, targetShelfId }: { shelfId: string; bookId: string; targetShelfId: string }) => readingService.moveShelfBook(shelfId, bookId, targetShelfId),
    onSuccess: () => refreshShelves(client),
  });
}

export function useRemoveShelfBook() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ shelfId, bookId }: { shelfId: string; bookId: string }) => readingService.removeFromShelf(shelfId, bookId),
    onSuccess: () => refreshShelves(client),
  });
}

export function useReadingProgress() {
  return useQuery({ queryKey: queryKeys.progress.all, queryFn: readingService.listProgress });
}

export function useReadingHistory(limit = 100) {
  return useQuery({ queryKey: ["history", limit], queryFn: () => readingService.history({ limit }) });
}

/**
 * Which of the reader's custom shelves hold the book. The API has no direct
 * call, so each custom shelf's books are read (readers have few custom shelves).
 */
export function useCustomShelfMembership(bookId: string, shelves: Shelf[] | undefined) {
  const custom = (shelves ?? []).filter((s) => s.type === "custom");
  const results = useQueries({
    queries: custom.map((shelf) => ({
      queryKey: queryKeys.shelves.contains(shelf.id, bookId),
      queryFn: async () => (await findInPages((page) => readingService.shelfBooks(shelf.id, page), (b) => b.id === bookId)) !== null,
      enabled: shelf.count > 0,
    })),
  });
  return custom.map((shelf, i) => ({ shelf, onShelf: shelf.count > 0 ? results[i].data : false, loading: shelf.count > 0 && results[i].isPending }));
}

/**
 * PATCH progress (status, page, or percentage) with the version the reader
 * saw. A 409 STALE_VERSION reloads the book so the next attempt uses the latest.
 */
export function useUpdateProgress(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (update: ProgressUpdateRequest) => readingService.updateProgress(bookId, update),
    onSuccess: (progress) => {
      setMyProgress(client, bookId, progress);
      refreshReading(client, bookId);
    },
    onError: (error) => {
      if (isApiError(error) && (error.code === "STALE_VERSION" || error.status === 404)) refreshReading(client, bookId);
    },
  });
}

export function useRemoveFromShelves(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => readingService.deleteProgress(bookId),
    onSuccess: () => {
      setMyProgress(client, bookId, null);
      refreshReading(client, bookId);
    },
  });
}

/**
 * Adding to a custom shelf goes through fn_move_book_to_shelf, which also
 * creates a want_to_read progress row when there is none, bumps the progress
 * version, and writes history; so the returned progress replaces the cached one.
 */
export function useToggleCustomShelf(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ shelfId, add }: { shelfId: string; add: boolean }): Promise<ShelfBookResult | null> => {
      if (add) return readingService.addToShelf(shelfId, bookId);
      await readingService.removeFromShelf(shelfId, bookId);
      return null;
    },
    onSuccess: (result, { shelfId, add }) => {
      client.setQueryData(queryKeys.shelves.contains(shelfId, bookId), add);
      if (result?.progress) setMyProgress(client, bookId, result.progress);
      refreshReading(client, bookId);
    },
  });
}

export function useBookHistory(bookId: string) {
  return useQuery({ queryKey: queryKeys.historyFor(bookId), queryFn: () => readingService.history({ bookId, limit: 5 }) });
}
