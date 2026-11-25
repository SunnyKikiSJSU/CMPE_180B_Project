import { apiClient, apiClientPage, apiClientWithHeaders, pageHeaders, query } from "@/lib/api-client";
import type { Author, AuthorDetail, Book, Category, CategoryDetail, ExternalBook, Page } from "@/types/domain";

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
  getAuthor: (authorId: string, page = 1, pageSize = DETAIL_PAGE_SIZE) => detailWithBooks<AuthorDetail>(`/api/authors/${encodeURIComponent(authorId)}${query({ page, pageSize })}`),
  getCategory: (categoryId: string, page = 1, pageSize = DETAIL_PAGE_SIZE) => detailWithBooks<CategoryDetail>(`/api/categories/${encodeURIComponent(categoryId)}${query({ page, pageSize })}`),
};

export const DETAIL_PAGE_SIZE = 20;

/** An author or category with one page of its books; the pagination headers page the books. */
export type DetailWithBooks<T> = { detail: Omit<T, "books">; books: Page<Book> };

async function detailWithBooks<T extends { books?: Book[] }>(path: string): Promise<DetailWithBooks<T>> {
  const { data, headers } = await apiClientWithHeaders<T>(path);
  const { books = [], ...detail } = data;
  return { detail, books: { items: books, ...pageHeaders(headers, books.length) } };
}
