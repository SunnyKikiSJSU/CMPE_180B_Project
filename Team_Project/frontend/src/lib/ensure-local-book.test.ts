import { describe, expect, it, vi } from "vitest";

import { createEnsureLocalBook } from "@/lib/ensure-local-book";
import type { Book } from "@/types/domain";

const book = (id: string) => ({ id }) as Book;

describe("ensureLocalBook", () => {
  it("returns an existing bookId without importing", async () => {
    const importBook = vi.fn();
    const ensure = createEnsureLocalBook(importBook);
    await expect(ensure({ externalId: "OL1W", bookId: "local-1" })).resolves.toBe("local-1");
    expect(importBook).not.toHaveBeenCalled();
  });

  it("imports once for concurrent and repeated calls on the same work", async () => {
    const importBook = vi.fn().mockResolvedValue(book("new-1"));
    const ensure = createEnsureLocalBook(importBook);
    const target = { externalId: "OL1W", bookId: null };
    await expect(Promise.all([ensure(target), ensure(target)])).resolves.toEqual(["new-1", "new-1"]);
    await expect(ensure(target)).resolves.toBe("new-1");
    expect(importBook).toHaveBeenCalledTimes(1);
    expect(importBook).toHaveBeenCalledWith("OL1W");
  });

  it("retries after a failed import", async () => {
    const importBook = vi.fn().mockRejectedValueOnce(new Error("502")).mockResolvedValueOnce(book("new-2"));
    const ensure = createEnsureLocalBook(importBook);
    const target = { externalId: "OL2W", bookId: null };
    await expect(ensure(target)).rejects.toThrow("502");
    await expect(ensure(target)).resolves.toBe("new-2");
    expect(importBook).toHaveBeenCalledTimes(2);
  });
});
