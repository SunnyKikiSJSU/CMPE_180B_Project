import { describe, expect, it } from "vitest";

import { formatAverage, formatCount, formatDay, formatPublishedDate, formatRating } from "@/lib/format";

describe("formatPublishedDate", () => {
  it("formats by precision without shifting the day", () => {
    expect(formatPublishedDate("1965-01-01", "year")).toBe("1965");
    expect(formatPublishedDate("2004-05-01", "month")).toBe("May 2004");
    expect(formatPublishedDate("2004-05-12", "day")).toBe("12 May 2004");
    expect(formatPublishedDate("2004-01-01", null)).toBe("1 January 2004");
  });

  it("handles missing and odd values", () => {
    expect(formatPublishedDate(null, "year")).toBe("");
    expect(formatPublishedDate("someday", "day")).toBe("someday");
  });
});

describe("formatRating", () => {
  it("shows the average with a count", () => {
    expect(formatRating(4.333, 12)).toBe("4.3 (12 ratings)");
    expect(formatRating(5, 1)).toBe("5.0 (1 rating)");
    expect(formatRating(0, 0)).toBe("No ratings yet");
  });
});

describe("report formatting", () => {
  it("formats counts, averages, and UTC days", () => {
    expect(formatCount(12345)).toBe("12,345");
    expect(formatCount(undefined)).toBe("0");
    expect(formatAverage(4.083)).toBe("4.08");
    expect(formatAverage(0, 0)).toBe("—");
    expect(formatDay("2026-11-23")).toBe("23 Nov");
    expect(formatDay("bad")).toBe("bad");
  });
});
