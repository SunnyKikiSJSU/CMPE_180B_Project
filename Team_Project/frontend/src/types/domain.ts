// TypeScript mirror of docs/api/openapi.yaml (components/schemas). Field names
// match the API exactly; nullable fields are `T | null`, optional ones `?`.

// --- Users and auth ---------------------------------------------------------

export interface UserProfile { bio: string | null; favoriteGenre: string | null; updatedAt?: string; }
export interface UserStats { shelfCount: number; booksCompleted: number; reviewsWritten: number; ratingsGiven: number; }
export interface User {
  id: string;
  email: string;
  displayName: string;
  avatarUrl: string | null;
  createdAt: string;
  profile?: UserProfile;
  stats?: UserStats;
}
/** register and login: the session is also set as the bm_session HttpOnly cookie. */
export interface AuthResponse { user: User; accessToken: string; }
export interface RegisterRequest { email: string; password: string; displayName: string; }
export interface LoginRequest { email: string; password: string; }
/** PATCH /api/users/me: an omitted field is unchanged; no field can be set back to null. */
export interface UpdateProfileRequest { displayName?: string; avatarUrl?: string; bio?: string; favoriteGenre?: string; }
export interface ChangePasswordRequest { currentPassword: string; newPassword: string; }

// --- Catalog ----------------------------------------------------------------

export interface Author { id: string; name: string; bookCount?: number; }
export interface Category { id: string; name: string; bookCount?: number; }
export interface AuthorDetail extends Author { books?: Book[]; }
export interface CategoryDetail extends Category { books?: Book[]; }

/** seed: demo data; openlibrary: imported (read-only); google: reserved; user: added by hand. */
export type BookSource = "seed" | "openlibrary" | "google" | "user";
export type DatePrecision = "year" | "month" | "day";

export interface Book {
  id: string;
  title: string;
  /** Primary author, "Unknown author" when there is none (kept for older components). */
  author: string;
  authors: Author[];
  categories: Category[];
  source: BookSource;
  infoLink: string | null;
  description?: string | null;
  /** YYYY-MM-DD; for year/month precision the first day of that year or month. */
  publishedDate?: string | null;
  publishedDatePrecision?: DatePrecision | null;
  publisher?: string | null;
  pageCount?: number | null;
  isbn13?: string | null;
  coverUrl?: string | null;
  averageRating?: number;
  ratingCount?: number;
  reviewCount?: number;
  myProgress?: ReadingProgress | null;
  myRating?: number | null;
}

export interface BookInput {
  title: string;
  authors?: string[];
  categories?: string[];
  description?: string;
  publishedDate?: string;
  publisher?: string;
  pageCount?: number;
  coverUrl?: string;
}
export type BookUpdate = Partial<Omit<BookInput, "authors" | "categories">>;

/** An Open Library search result; not in the library until imported. */
export interface ExternalBook {
  source: "openlibrary";
  externalId: string;
  bookId: string | null;
  title: string;
  subtitle: string | null;
  authors: string[];
  categories: string[];
  description: string | null;
  /** First publication year, for example "1965". */
  publishedDate: string | null;
  publisher: string | null;
  pageCount: number | null;
  isbn13: string | null;
  coverUrl: string | null;
  infoLink: string;
}
export interface ImportBookRequest { source?: "openlibrary"; externalId: string; }

// --- Shelves and progress ---------------------------------------------------

export type ReadingStatus = "want_to_read" | "reading" | "completed" | "paused";
export type ShelfType = "want_to_read" | "currently_reading" | "finished" | "paused" | "custom";

export interface Shelf { id: string; name: string; type: ShelfType; count: number; createdAt?: string; }
export interface BookSummary { id: string; title: string; author: string; coverUrl: string | null; }

export interface ReadingProgress {
  bookId: string;
  status: ReadingStatus;
  currentPage: number;
  /** books.page_count, or 0 when unknown (then percentComplete is edited instead). */
  totalPages: number;
  percentComplete?: number;
  version?: number;
  updatedAt: string;
  book?: BookSummary;
}
export interface ProgressUpdateRequest {
  currentPage?: number;
  percentComplete?: number;
  status?: ReadingStatus;
  /** Optimistic lock: a stale version is 409 STALE_VERSION. */
  version?: number;
}
export interface ShelfBookResult { shelfId: string; bookId: string; progress?: ReadingProgress; historyEvent?: string; }

export interface ReadingHistoryEvent {
  id: string;
  bookId: string;
  bookTitle?: string;
  eventType: string;
  fromStatus?: ReadingStatus | null;
  toStatus?: ReadingStatus | null;
  createdAt: string;
}

// --- Ratings and reviews ----------------------------------------------------

export interface RatingSummary { averageRating: number; ratingCount: number; reviewCount: number; }
export interface RatingSummaryResponse { averageRating: number; ratingCount: number; distribution: Record<"1" | "2" | "3" | "4" | "5", number>; }
export interface Rating {
  bookId: string;
  userId: string;
  value: number;
  createdAt?: string;
  updatedAt?: string;
  summary?: RatingSummary;
  /** Only in GET /api/users/me/ratings. */
  bookTitle?: string;
  bookCoverUrl?: string | null;
}
export interface ReviewUser { id: string; displayName: string; avatarUrl: string | null; }
export interface Review {
  id: string;
  bookId: string;
  userId: string;
  title: string | null;
  body: string;
  rating: number | null;
  user?: ReviewUser;
  createdAt: string;
  updatedAt?: string;
  /** Only in GET /api/users/me/reviews. */
  bookTitle?: string;
  bookCoverUrl?: string | null;
}
export interface ReviewInput { title?: string; body: string; }
export interface ReviewUpsertInput extends ReviewInput { rating?: number; }
export interface ReviewRatingResult { review: Review; rating: Rating | null; summary: RatingSummary; }

// --- Reports ----------------------------------------------------------------

export interface RecentActivity { bookId: string; status: ReadingStatus; currentPage: number; totalPages: number; updatedAt: string; }
export interface ReadingReport { booksRead: number; pagesRead: number; favoriteCategories: Category[]; recentActivity: RecentActivity[]; }
export interface UserStatsReport { booksCompleted: number; booksInProgress: number; pagesRead: number; ratingsGiven: number; averageRating: number; reviewsWritten: number; }
export interface PopularBook { bookId: string; title: string; coverUrl: string | null; primaryAuthor: string; shelfAdds: number; ratings: number; averageRating: number; score: number; }
export interface TopRatedBook { bookId: string; title: string; coverUrl: string | null; primaryAuthor: string; averageRating: number; ratingCount: number; }
export interface Recommendation { bookId: string; title: string; coverUrl: string | null; primaryAuthor: string; score: number; averageRating: number; }
export interface AuthorRanking { authorId: string; name: string; bookCount: number; ratingCount: number; averageRating: number; ranking: number; }
export interface CategoryRanking { categoryId: string; name: string; bookCount: number; ratingCount: number; averageRating: number; ranking: number; }
export interface ReviewActivityPoint { date: string; reviews: number; ratings: number; }
export interface QueryCatalogEntry { id: string; name: string; description: string; endpoints: string[]; queries: string[]; sql: string; }
/** A report body with its X-Query-Id header (the evidence ID in GET /api/reports/queries). */
export interface ReportResult<T> { data: T; queryId: string | null; }

// --- Lists ------------------------------------------------------------------

/** A list response with its pagination headers (X-Total-Count, X-Page, X-Page-Size). */
export interface Page<T> { items: T[]; total: number; page: number; pageSize: number; }
