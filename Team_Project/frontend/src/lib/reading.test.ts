import { describe, expect, it } from "vitest";

import { historyLabel, progressPercent, progressSummary } from "@/lib/reading";

describe("progress", () => {
  it("uses pages when the page count is known", () => {
    expect(progressSummary({ currentPage: 81, totalPages: 273, percentComplete: 0 })).toBe("Page 81 of 273 · 30%");
  });

  it("uses the stored percentage when there is no page count", () => {
    expect(progressPercent({ currentPage: 0, totalPages: 0, percentComplete: 44.6 })).toBe(45);
    expect(progressSummary({ currentPage: 0, totalPages: 0, percentComplete: undefined })).toBe("0% read");
  });

  it("stays within 0-100", () => {
    expect(progressPercent({ currentPage: 400, totalPages: 300, percentComplete: 0 })).toBe(100);
  });
});

describe("historyLabel", () => {
  it("describes status changes and completion", () => {
    expect(historyLabel({ eventType: "status_changed", fromStatus: null, toStatus: "want_to_read" })).toBe("Added as Want to read");
    expect(historyLabel({ eventType: "status_changed", fromStatus: "want_to_read", toStatus: "reading" })).toBe("Want to read → Reading");
    expect(historyLabel({ eventType: "book_completed", fromStatus: "reading", toStatus: "completed" })).toBe("Finished the book");
    expect(historyLabel({ eventType: "custom_event", fromStatus: null, toStatus: null })).toBe("custom event");
  });
});
