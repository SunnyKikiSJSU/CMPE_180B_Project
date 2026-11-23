import type { ReadingHistoryEvent, ReadingProgress, ReadingStatus } from "@/types/domain";

export const statusLabels: Record<ReadingStatus, string> = { want_to_read: "Want to read", reading: "Reading", paused: "Paused", completed: "Finished" };
export const readingStatuses: ReadingStatus[] = ["want_to_read", "reading", "paused", "completed"];

type ProgressNumbers = Pick<ReadingProgress, "currentPage" | "totalPages" | "percentComplete">;

/** 0-100. With no page count (totalPages 0) the stored percentage is used; never divides by zero. */
export function progressPercent(p: ProgressNumbers): number {
  const raw = p.totalPages > 0 ? (p.currentPage / p.totalPages) * 100 : (p.percentComplete ?? 0);
  return Math.min(100, Math.max(0, Math.round(raw)));
}

/** "Page 81 of 273 · 30%", or "45% read" for a book without a page count. */
export function progressSummary(p: ProgressNumbers): string {
  const percent = progressPercent(p);
  return p.totalPages > 0 ? `Page ${p.currentPage} of ${p.totalPages} · ${percent}%` : `${percent}% read`;
}

const status = (s: string | null | undefined) => (s && s in statusLabels ? statusLabels[s as ReadingStatus] : "no status");

/** A sentence for one reading-history row. */
export function historyLabel(event: Pick<ReadingHistoryEvent, "eventType" | "fromStatus" | "toStatus">): string {
  switch (event.eventType) {
    case "book_completed":
      return "Finished the book";
    case "status_changed":
      return event.fromStatus ? `${status(event.fromStatus)} → ${status(event.toStatus)}` : `Added as ${status(event.toStatus)}`;
    case "progress_updated":
      return "Updated progress";
    case "shelf_moved":
      return "Moved between shelves";
    default:
      return event.eventType.replace(/_/g, " ");
  }
}
