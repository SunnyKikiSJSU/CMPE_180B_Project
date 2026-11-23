import { apiClient, apiClientPage, query } from "@/lib/api-client";
import type { Author, Book, Category, ExternalBook, Rating, Review } from "@/types/domain";

export type LibrarySort = "relevance" | "newest" | "rating" | "title" | "popular";
export type ExternalSort = "relevance" | "newest";

/** GET /api/books filters. Type aliases (not interfaces) so they fit query(). */
export type LibrarySearch = {
  q?: string;
  authorId?: string;
  categoryId?: string;
  sort?: LibrarySort;
  page?: number;
  pageSize?: number;
};

/** GET /api/external/books filters (Open Library; pageSize at most 40). */
export type ExternalSearch = {
  q?: string;
  title?: string;
  author?: string;
  category?: string;
  isbn?: string;
  sort?: ExternalSort;
  page?: number;
  pageSize?: number;
};

export type NameSearch = { q?: string; page?: number; pageSize?: number };

export const bookService = {
  searchLibrary: (filters: LibrarySearch, signal?: AbortSignal) => apiClientPage<Book>(`/api/books${query(filters)}`, { signal }),
  /** The API calls Open Library; X-Total-Count is its estimate (numFound). */
  searchExternal: (filters: ExternalSearch, signal?: AbortSignal) => apiClientPage<ExternalBook>(`/api/external/books${query(filters)}`, { signal }),
  /** 201 when imported, 200 when the library already had it; both return the local Book. */
  importExternal: (externalId: string) => apiClient<Book>("/api/books/import", { method: "POST", body: { source: "openlibrary", externalId } }),
  getById: (bookId: string, signal?: AbortSignal) => apiClient<Book>(`/api/books/${encodeURIComponent(bookId)}`, { signal }),
  listAuthors: (filters: NameSearch = {}) => apiClientPage<Author>(`/api/authors${query(filters)}`),
  listCategories: (filters: NameSearch = {}) => apiClientPage<Category>(`/api/categories${query(filters)}`),
  getReviews: (bookId: string) => apiClient<Review[]>(`/api/books/${encodeURIComponent(bookId)}/reviews`),
  rate: (bookId: string, value: number) => apiClient<Rating>(`/api/books/${encodeURIComponent(bookId)}/ratings`, { method: "POST", body: { value } }),
};
