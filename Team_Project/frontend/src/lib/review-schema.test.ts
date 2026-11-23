import { describe, expect, it } from "vitest";

import { reviewSchema } from "@/lib/review-schema";

describe("reviewSchema", () => {
  it("trims and accepts a review without a title or rating", () => {
    expect(reviewSchema.parse({ rating: null, title: "  ", body: "  Loved it. " })).toEqual({ rating: null, title: "", body: "Loved it." });
  });

  it("rejects an empty body and an out-of-range rating", () => {
    expect(reviewSchema.safeParse({ rating: null, title: "", body: "   " }).success).toBe(false);
    expect(reviewSchema.safeParse({ rating: 6, title: "", body: "ok" }).success).toBe(false);
  });

  it("counts characters, not bytes", () => {
    expect(reviewSchema.safeParse({ rating: 5, title: "é".repeat(200), body: "ok" }).success).toBe(true);
    expect(reviewSchema.safeParse({ rating: 5, title: "é".repeat(201), body: "ok" }).success).toBe(false);
  });
});
