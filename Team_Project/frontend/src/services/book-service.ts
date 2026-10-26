import { apiClient } from "@/lib/api-client";
import type { Book, Review, Rating } from "@/types/domain";

export const bookService = {
  search: (query: string) => apiClient<Book[]>(`/api/books?q=${encodeURIComponent(query)}`),
  getById: (bookId: string) => apiClient<Book>(`/api/books/${bookId}`),
  getReviews: (bookId: string) => apiClient<Review[]>(`/api/books/${bookId}/reviews`),
  rate: (bookId: string, value: number) => apiClient<Rating>(`/api/books/${bookId}/ratings`, { method: "POST", body: JSON.stringify({ value }) }),
};
