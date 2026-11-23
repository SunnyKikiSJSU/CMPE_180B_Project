import type { Book, ExternalBook } from "@/types/domain";

type ImportTarget = Pick<ExternalBook, "externalId" | "bookId">;

/**
 * Builds ensureLocalBook (FRONTEND_PLAN.md decision 5): an Open Library result
 * becomes a local book ID, importing it only when the result has no bookId.
 * Concurrent and repeated calls for the same work share one import, so a
 * double click or a stale result list never imports twice. A failed import is
 * forgotten, so the next attempt tries again.
 */
export function createEnsureLocalBook(importBook: (externalId: string) => Promise<Book>) {
  const imports = new Map<string, Promise<string>>();
  return function ensureLocalBook(external: ImportTarget): Promise<string> {
    if (external.bookId) return Promise.resolve(external.bookId);
    let pending = imports.get(external.externalId);
    if (!pending) {
      pending = importBook(external.externalId).then((book) => book.id);
      pending.catch(() => imports.delete(external.externalId));
      imports.set(external.externalId, pending);
    }
    return pending;
  };
}
