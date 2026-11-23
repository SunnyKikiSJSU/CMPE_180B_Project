import Link from "next/link";

/**
 * Previous/next links for a list with pagination headers. `approximate` marks
 * Open Library's estimated total; `maxPage` caps what the API allows.
 */
export function Pagination({ page, pageSize, total, hrefFor, approximate = false, maxPage }: { page: number; pageSize: number; total: number; hrefFor: (page: number) => string; approximate?: boolean; maxPage?: number }) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  // Past maxPage the API cannot serve, so the label shows the reachable pages, not the estimate.
  const capped = maxPage !== undefined && pages > maxPage;
  const last = capped ? maxPage : pages;
  if (last <= 1 && page <= 1) return null;
  const pageLink = (target: number, label: string, rel: "prev" | "next") =>
    target < 1 || target > last ? <span className="pagination-link is-disabled" aria-disabled="true">{label}</span> : <Link className="pagination-link" href={hrefFor(target)} rel={rel} scroll={false}>{label}</Link>;
  return (
    <nav className="pagination" aria-label="Pagination">
      {pageLink(page - 1, "← Previous", "prev")}
      <span className="pagination-status">Page {page} of {approximate && !capped ? "about " : ""}{last.toLocaleString("en-US")}</span>
      {pageLink(page + 1, "Next →", "next")}
    </nav>
  );
}
