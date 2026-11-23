import { z } from "zod";

// Mirrors the API (backend/internal/services/reviews.go): title up to 200
// characters, body 1 to 5000, both counted in characters (runes), trimmed.
const chars = (s: string) => Array.from(s).length;

export const reviewSchema = z.object({
  rating: z.number().int().min(1).max(5).nullable(),
  title: z.string().trim().refine((s) => chars(s) <= 200, "Use at most 200 characters."),
  body: z.string().trim().min(1, "Write a few words about the book.").refine((s) => chars(s) <= 5000, "Use at most 5000 characters."),
});

export type ReviewValues = z.input<typeof reviewSchema>;
export type ReviewOutput = z.output<typeof reviewSchema>;
