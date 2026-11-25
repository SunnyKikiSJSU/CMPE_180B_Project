import { apiClient, apiClientPage, query } from "@/lib/api-client";
import type { Page, Rating, Review, ReviewInput, ReviewRatingResult, ReviewUpsertInput } from "@/types/domain";

const id = encodeURIComponent;
const MY_REVIEWS_PAGE = 100;

export const reviewService = {
  rate: (bookId: string, value: number) => apiClient<Rating>(`/api/books/${id(bookId)}/ratings`, { method: "POST", body: { value } }),
  clearRating: (bookId: string) => apiClient<void>(`/api/books/${id(bookId)}/ratings`, { method: "DELETE" }),
  bookReviews: (bookId: string, page = 1, pageSize = 10) => apiClientPage<Review>(`/api/books/${id(bookId)}/reviews${query({ sort: "newest", page, pageSize })}`),
  /** Saves the reader's review and (optionally) rating in one transaction. */
  saveReview: (bookId: string, input: ReviewUpsertInput) => apiClient<ReviewRatingResult>(`/api/books/${id(bookId)}/review`, { method: "PUT", body: input }),
  /** Title and body only; an omitted title is kept, a blank one cleared. The rating is separate. */
  updateReview: (reviewId: string, input: ReviewInput) => apiClient<Review>(`/api/reviews/${id(reviewId)}`, { method: "PATCH", body: input }),
  deleteReview: (reviewId: string) => apiClient<void>(`/api/reviews/${id(reviewId)}`, { method: "DELETE" }),
  /** Newest first, with bookTitle and bookCoverUrl. */
  myReviews: (page = 1, pageSize = MY_REVIEWS_PAGE) => apiClientPage<Review>(`/api/users/me/reviews${query({ page, pageSize })}`),
  /** Most recently changed first, with bookTitle and bookCoverUrl. */
  myRatings: (page = 1, pageSize = MY_REVIEWS_PAGE) => apiClientPage<Rating>(`/api/users/me/ratings${query({ page, pageSize })}`),
  /** The API has no "my review of this book" call, so this pages through the reader's reviews. */
  findMyReview: async (bookId: string): Promise<Review | null> => findInPages((page) => reviewService.myReviews(page), (r) => r.bookId === bookId),
};

/** Reads pages until `match` finds an item or the list ends. */
export async function findInPages<T>(load: (page: number) => Promise<Page<T>>, match: (item: T) => boolean): Promise<T | null> {
  for (let page = 1; ; page++) {
    const result = await load(page);
    const found = result.items.find(match);
    if (found) return found;
    if (result.items.length === 0 || page * result.pageSize >= result.total) return null;
  }
}
