"use client";

import Link from "next/link";

import { BookCell, ReportBody, ReportSection } from "@/components/reports/ReportParts";
import { useBooksById } from "@/hooks/use-books";
import { useReadingReport, useRecommendations, useUserStats } from "@/hooks/use-reports";
import { catalogHref } from "@/lib/catalog-params";
import { formatAverage, formatCount, formatDate } from "@/lib/format";
import { categoryHref } from "@/lib/page-params";
import { progressSummary, statusLabels } from "@/lib/reading";
import type { RecentActivity } from "@/types/domain";

/** The signed-in reader's statistics (q07), recent activity, and recommendations (q06). */
export function MyReadingReport() {
  const stats = useUserStats();
  const reading = useReadingReport();
  const recommendations = useRecommendations(10);
  return (
    <div className="report-stack">
      <ReportSection title="Your reading at a glance" queryId={stats.data?.queryId}
        description="Pages read counts the full page count of finished books plus the current page of books you are reading or paused.">
        <ReportBody query={stats} empty="">
          {(s) => (
            <dl className="report-stats">
              <Stat label="Books finished" value={formatCount(s.booksCompleted)} />
              <Stat label="Reading or paused" value={formatCount(s.booksInProgress)} />
              <Stat label="Pages read" value={formatCount(s.pagesRead)} />
              <Stat label="Ratings given" value={formatCount(s.ratingsGiven)} />
              <Stat label="Your average rating" value={formatAverage(s.averageRating, s.ratingsGiven)} />
              <Stat label="Reviews written" value={formatCount(s.reviewsWritten)} />
            </dl>
          )}
        </ReportBody>
      </ReportSection>

      <ReportSection title="Favorite categories and recent activity" queryId={reading.data?.queryId}
        description="Categories of the books you finished or rated 4 or 5 stars, and the books whose progress changed most recently.">
        <ReportBody query={reading} empty="">
          {(r) => (
            <div className="report-columns">
              <div>
                <h3>Favorite categories</h3>
                {r.favoriteCategories.length === 0 ? (
                  <p className="report-empty">Finish or rate a book 4+ stars to see your favorite categories.</p>
                ) : (
                  <ol className="report-chips">
                    {r.favoriteCategories.map((c) => (
                      <li key={c.id}><Link href={categoryHref(c.id)} prefetch={false}>{c.name}{c.bookCount !== undefined && <small>{c.bookCount}</small>}</Link></li>
                    ))}
                  </ol>
                )}
              </div>
              <div>
                <h3>Recent activity</h3>
                {r.recentActivity.length === 0 ? <p className="report-empty">No reading progress yet. Set a status on any book to start.</p> : <RecentActivityList rows={r.recentActivity} />}
              </div>
            </div>
          )}
        </ReportBody>
      </ReportSection>

      <ReportSection title="Recommended for you" queryId={recommendations.data?.queryId}
        description="Books from the categories you liked, weighted by how many liked books share each category, leaving out books you already track, rated, or reviewed.">
        <ReportBody query={recommendations} isEmpty={(rows) => rows.length === 0} empty="Nothing to recommend yet. Finish or rate a few books 4+ stars first.">
          {(rows) => (
            <ol className="report-list">
              {rows.map((row) => (
                <li key={row.bookId}>
                  <BookCell id={row.bookId} title={row.title} author={row.primaryAuthor} cover={row.coverUrl} />
                  <span className="report-list-meta">★ {formatAverage(row.averageRating)} · score {formatCount(row.score)}</span>
                </li>
              ))}
            </ol>
          )}
        </ReportBody>
      </ReportSection>
      <p className="muted-note">Want more books in these reports? <Link className="text-link" href={catalogHref({ tab: "openlibrary" })}>Add some from Open Library</Link>.</p>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return <div className="report-stat"><dt>{label}</dt><dd>{value}</dd></div>;
}

/** The report has only book IDs; titles come from the (cached) book detail. */
function RecentActivityList({ rows }: { rows: RecentActivity[] }) {
  const books = useBooksById(rows.map((r) => r.bookId));
  return (
    <ol className="report-list">
      {rows.map((row) => {
        const book = books.get(row.bookId);
        return (
          <li key={row.bookId}>
            <BookCell id={row.bookId} title={book?.title ?? "Loading…"} author={book?.author} cover={book?.coverUrl} />
            <span className="report-list-meta">
              {statusLabels[row.status] ?? row.status}
              {row.totalPages > 0 && ` · ${progressSummary({ currentPage: row.currentPage, totalPages: row.totalPages })}`}
              <br /><time dateTime={row.updatedAt}>{formatDate(row.updatedAt)}</time>
            </span>
          </li>
        );
      })}
    </ol>
  );
}
