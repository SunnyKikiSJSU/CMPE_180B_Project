"use client";

import { keepPreviousData, useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";

import { createEnsureLocalBook } from "@/lib/ensure-local-book";
import { queryKeys } from "@/lib/query-keys";
import { bookService, type DetailWithBooks, type ExternalSearch, type LibrarySearch, type NameSearch } from "@/services/book-service";
import type { AuthorDetail, CategoryDetail, ExternalBook } from "@/types/domain";

const ensureLocalBook = createEnsureLocalBook(bookService.importExternal);

export function useLibrarySearch(filters: LibrarySearch, enabled = true) {
  return useQuery({
    queryKey: queryKeys.books.search(filters),
    queryFn: ({ signal }) => bookService.searchLibrary(filters, signal),
    placeholderData: keepPreviousData,
    enabled,
  });
}

/** Open Library is slow (2-7 s), so results are kept for 5 minutes and paging keeps the old grid. */
export function useExternalSearch(filters: ExternalSearch, enabled = true) {
  return useQuery({
    queryKey: queryKeys.external(filters),
    queryFn: ({ signal }) => bookService.searchExternal(filters, signal),
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
    enabled,
  });
}

export function useBook(bookId: string) {
  return useQuery({ queryKey: queryKeys.books.detail(bookId), queryFn: ({ signal }) => bookService.getById(bookId, signal) });
}

/** Filter options; the seed has fewer than 100 of each, the API's maximum page. */
export function useAuthorOptions() {
  return useQuery({ queryKey: queryKeys.authors({ pageSize: 100 }), queryFn: () => bookService.listAuthors({ pageSize: 100 }), staleTime: 5 * 60_000 });
}

export function useCategoryOptions() {
  return useQuery({ queryKey: queryKeys.categories({ pageSize: 100 }), queryFn: () => bookService.listCategories({ pageSize: 100 }), staleTime: 5 * 60_000 });
}

/** A few books by ID (a report row that has only bookId), sharing the book page cache. */
export function useBooksById(ids: string[]) {
  const results = useQueries({
    queries: ids.map((id) => ({ queryKey: queryKeys.books.detail(id), queryFn: ({ signal }: { signal: AbortSignal }) => bookService.getById(id, signal), staleTime: 5 * 60_000 })),
  });
  return new Map(ids.map((id, i) => [id, results[i]?.data]));
}

/** A searchable, paged author or category list (/authors). */
export function useNameList(kind: "authors" | "categories", filters: NameSearch) {
  return useQuery({
    queryKey: kind === "authors" ? queryKeys.authors(filters) : queryKeys.categories(filters),
    queryFn: () => (kind === "authors" ? bookService.listAuthors(filters) : bookService.listCategories(filters)),
    placeholderData: keepPreviousData,
  });
}

/** Keeps the previous page while the next loads, but never shows another author's or category's books. */
const samePrevious = (id: string) => <T,>(previous: T | undefined, query?: { queryKey: readonly unknown[] }) => (query?.queryKey[2] === id ? previous : undefined);

/** An author or category with one page of its books. */
export function useNameDetail(kind: "author" | "category", id: string, page: number) {
  return useQuery({
    queryKey: kind === "author" ? queryKeys.author(id, page) : queryKeys.category(id, page),
    queryFn: (): Promise<DetailWithBooks<AuthorDetail | CategoryDetail>> => (kind === "author" ? bookService.getAuthor(id, page) : bookService.getCategory(id, page)),
    placeholderData: samePrevious(id),
  });
}

/**
 * Turns an Open Library result into a local book ID (importing when needed),
 * then refreshes library lists and search results so they show the new bookId.
 */
export function useEnsureLocalBook() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (external: ExternalBook) => ensureLocalBook(external),
    onSuccess: (_bookId, external) => {
      if (external.bookId) return;
      void client.invalidateQueries({ queryKey: queryKeys.books.all });
      void client.invalidateQueries({ queryKey: queryKeys.externalAll });
    },
  });
}
