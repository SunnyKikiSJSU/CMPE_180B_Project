// TanStack Query keys (FRONTEND_PLAN.md §3.2). Invalidate narrowly after a
// mutation; never build these arrays by hand elsewhere.
export const queryKeys = {
  me: ["auth", "me"] as const,
  books: {
    all: ["books"] as const,
    search: (filters: object) => ["books", "search", filters] as const,
    detail: (id: string) => ["books", id] as const,
    reviews: (id: string) => ["books", id, "reviews"] as const,
  },
  externalAll: ["external"] as const,
  external: (filters: object) => ["external", "openlibrary", filters] as const,
  shelves: {
    all: ["shelves"] as const,
    books: (id: string) => ["shelves", id, "books"] as const,
  },
  progress: {
    all: ["progress"] as const,
    detail: (bookId: string) => ["progress", bookId] as const,
  },
  history: ["history"] as const,
  myReviews: ["reviews", "me"] as const,
  myRatings: ["ratings", "me"] as const,
  authors: (filters: object) => ["authors", filters] as const,
  categories: (filters: object) => ["categories", filters] as const,
  reports: (type: string, params?: object) => ["reports", type, params ?? {}] as const,
};
