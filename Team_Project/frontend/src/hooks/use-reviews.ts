"use client";

import { keepPreviousData, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { queryKeys } from "@/lib/query-keys";
import { reviewService } from "@/services/review-service";
import type { Book, RatingSummary, ReviewUpsertInput } from "@/types/domain";

/** Ratings and reviews change the book's summary, every list showing it, and the reader's own lists. */
function refreshReviews(client: QueryClient) {
  for (const queryKey of [queryKeys.books.all, queryKeys.myReviews, queryKeys.myRatings, ["reports"]]) {
    void client.invalidateQueries({ queryKey });
  }
}

function patchBook(client: QueryClient, bookId: string, patch: Partial<Book>, summary?: RatingSummary) {
  client.setQueryData<Book>(queryKeys.books.detail(bookId), (book) => (book ? { ...book, ...patch, ...(summary ?? {}) } : book));
}

export function useBookReviews(bookId: string, page: number) {
  return useQuery({ queryKey: queryKeys.books.reviews(bookId, page), queryFn: () => reviewService.bookReviews(bookId, page), placeholderData: keepPreviousData });
}

export function useMyReview(bookId: string) {
  return useQuery({ queryKey: queryKeys.myReviewFor(bookId), queryFn: () => reviewService.findMyReview(bookId) });
}

export function useRate(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (value: number) => reviewService.rate(bookId, value),
    onSuccess: (rating) => {
      patchBook(client, bookId, { myRating: rating.value }, rating.summary);
      refreshReviews(client);
    },
  });
}

export function useClearRating(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => reviewService.clearRating(bookId),
    onSuccess: () => {
      patchBook(client, bookId, { myRating: null });
      refreshReviews(client);
    },
  });
}

/** PUT /api/books/{bookId}/review: the review and its rating in one transaction. */
export function useSaveReview(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: ReviewUpsertInput) => reviewService.saveReview(bookId, input),
    onSuccess: (result) => {
      client.setQueryData(queryKeys.myReviewFor(bookId), result.review);
      patchBook(client, bookId, result.rating ? { myRating: result.rating.value } : {}, result.summary);
      refreshReviews(client);
    },
  });
}

export function useDeleteReview(bookId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (reviewId: string) => reviewService.deleteReview(reviewId),
    onSuccess: () => {
      client.setQueryData(queryKeys.myReviewFor(bookId), null);
      refreshReviews(client);
    },
  });
}
