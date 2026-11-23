import type { DatePrecision } from "@/types/domain";

const monthYear = new Intl.DateTimeFormat("en-US", { month: "long", year: "numeric", timeZone: "UTC" });
const fullDate = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "long", year: "numeric", timeZone: "UTC" });

/**
 * Shows a published date at its real precision (FRONTEND_PLAN.md decision 12):
 * "2004", "May 2004", or "12 May 2004". Dates are calendar dates, so they are
 * formatted in UTC to avoid shifting a day in the user's time zone.
 */
export function formatPublishedDate(date: string | null | undefined, precision: DatePrecision | null | undefined): string {
  if (!date) return "";
  const d = new Date(`${date.slice(0, 10)}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) return date;
  switch (precision ?? "day") {
    case "year":
      return String(d.getUTCFullYear());
    case "month":
      return monthYear.format(d);
    default:
      return fullDate.format(d);
  }
}

/** "4.3 (12 ratings)", or "No ratings yet". */
export function formatRating(average: number | undefined, count: number | undefined): string {
  if (!count) return "No ratings yet";
  return `${(average ?? 0).toFixed(1)} (${count} ${count === 1 ? "rating" : "ratings"})`;
}
