import type { ExternalSearch, ExternalSort, LibrarySearch, LibrarySort } from "@/services/book-service";

export type CatalogTab = "openlibrary" | "library";

/** The catalog's state lives in the URL so a search can be shared, reloaded, and navigated back to. */
export interface CatalogState {
  tab: CatalogTab;
  q: string;
  page: number;
  sort: LibrarySort | ExternalSort;
  /** Library tab only. */
  authorId: string;
  categoryId: string;
  /** Open Library tab only. */
  title: string;
  author: string;
  subject: string;
  isbn: string;
}

interface ParamReader { get(name: string): string | null; }

const librarySorts: readonly LibrarySort[] = ["relevance", "newest", "rating", "title", "popular"];
const externalSorts: readonly ExternalSort[] = ["relevance", "newest"];
export const LIBRARY_PAGE_SIZE = 20;
export const EXTERNAL_PAGE_SIZE = 20;
/** The API allows Open Library pages 1..100. */
export const EXTERNAL_MAX_PAGE = 100;

export function parseCatalogParams(params: ParamReader): CatalogState {
  const tab: CatalogTab = params.get("tab") === "library" ? "library" : "openlibrary";
  const text = (name: string) => (params.get(name) ?? "").trim();
  const rawPage = Number.parseInt(params.get("page") ?? "", 10);
  let page = Number.isFinite(rawPage) && rawPage > 0 ? rawPage : 1;
  if (tab === "openlibrary") page = Math.min(page, EXTERNAL_MAX_PAGE);
  const sorts: readonly string[] = tab === "library" ? librarySorts : externalSorts;
  const sort = (sorts.includes(text("sort")) ? text("sort") : "relevance") as CatalogState["sort"];
  const libraryOnly = tab === "library";
  return {
    tab,
    q: text("q"),
    page,
    sort,
    authorId: libraryOnly ? text("authorId") : "",
    categoryId: libraryOnly ? text("categoryId") : "",
    title: libraryOnly ? "" : text("title"),
    author: libraryOnly ? "" : text("author"),
    subject: libraryOnly ? "" : text("subject"),
    isbn: libraryOnly ? "" : text("isbn").replace(/[\s-]/g, ""),
  };
}

/** A /catalog URL for the state, leaving out defaults and the other tab's fields. */
export function catalogHref(state: Partial<CatalogState>): string {
  const full = parseCatalogParams({ get: (name) => { const v = state[name as keyof CatalogState]; return v === undefined ? null : String(v); } });
  const search = new URLSearchParams();
  if (full.tab === "library") search.set("tab", "library");
  const keys = full.tab === "library" ? (["q", "authorId", "categoryId"] as const) : (["q", "title", "author", "subject", "isbn"] as const);
  for (const key of keys) if (full[key]) search.set(key, full[key]);
  if (full.sort !== "relevance") search.set("sort", full.sort);
  if (full.page > 1) search.set("page", String(full.page));
  const s = search.toString();
  return s ? `/catalog?${s}` : "/catalog";
}

/** Open Library needs at least one search term; without one the tab shows its prompt. */
export function hasExternalTerms(state: CatalogState): boolean {
  return Boolean(state.q || state.title || state.author || state.subject || state.isbn);
}

export function toLibrarySearch(state: CatalogState): LibrarySearch {
  // "relevance" is left out so the API picks it with a search term and newest without one.
  const sort = state.sort === "relevance" ? undefined : (state.sort as LibrarySort);
  return { q: state.q || undefined, authorId: state.authorId || undefined, categoryId: state.categoryId || undefined, sort, page: state.page, pageSize: LIBRARY_PAGE_SIZE };
}

export function toExternalSearch(state: CatalogState): ExternalSearch {
  return {
    q: state.q || undefined,
    title: state.title || undefined,
    author: state.author || undefined,
    category: state.subject || undefined,
    isbn: state.isbn || undefined,
    sort: state.sort as ExternalSort,
    page: state.page,
    pageSize: EXTERNAL_PAGE_SIZE,
  };
}
