import { describe, expect, it, vi } from "vitest";

import { findInPages } from "@/services/review-service";

const pages = [[1, 2], [3, 4], [5]];
const load = vi.fn(async (page: number) => ({ items: pages[page - 1] ?? [], total: 5, page, pageSize: 2 }));

describe("findInPages", () => {
  it("stops at the page with a match", async () => {
    load.mockClear();
    await expect(findInPages(load, (n) => n === 3)).resolves.toBe(3);
    expect(load).toHaveBeenCalledTimes(2);
  });

  it("returns null after the last page", async () => {
    load.mockClear();
    await expect(findInPages(load, (n) => n === 9)).resolves.toBeNull();
    expect(load).toHaveBeenCalledTimes(3);
  });
});
