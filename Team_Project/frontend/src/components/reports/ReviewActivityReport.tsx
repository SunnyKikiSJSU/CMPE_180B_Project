"use client";

import { useRouter } from "next/navigation";

import { ReportBody, ReportSection, ReportSelect } from "@/components/reports/ReportParts";
import { useReviewActivity } from "@/hooks/use-reports";
import { formatCount, formatDay } from "@/lib/format";
import { dayOptions, reportHref, type ReportState } from "@/lib/page-params";
import type { ReviewActivityPoint } from "@/types/domain";

const daysLabel = (n: number) => (n === 365 ? "Last year" : `Last ${n} days`);

/** q09: reviews and ratings per UTC day, with empty days filled in by generate_series. */
export function ReviewActivityReport({ state }: { state: ReportState }) {
  const router = useRouter();
  const report = useReviewActivity({ days: state.days });
  return (
    <ReportSection title="Review activity per day" queryId={report.data?.queryId}
      description="New reviews and ratings on each UTC day, everyone included. Days without activity are counted as zero."
      controls={<ReportSelect label="Window" value={state.days} options={dayOptions} format={daysLabel} onChange={(days) => router.replace(reportHref({ ...state, days }), { scroll: false })} />}>
      <ReportBody query={report} empty="">
        {(points) => <ActivityChart points={points} />}
      </ReportBody>
    </ReportSection>
  );
}

function ActivityChart({ points }: { points: ReviewActivityPoint[] }) {
  const reviews = points.reduce((n, p) => n + p.reviews, 0);
  const ratings = points.reduce((n, p) => n + p.ratings, 0);
  const max = Math.max(1, ...points.map((p) => Math.max(p.reviews, p.ratings)));
  const busiest = points.reduce<ReviewActivityPoint | null>((best, p) => (p.reviews + p.ratings > (best ? best.reviews + best.ratings : 0) ? p : best), null);
  const first = points[0]?.date;
  const last = points[points.length - 1]?.date;
  const summary = `${formatCount(reviews)} reviews and ${formatCount(ratings)} ratings from ${first ? formatDay(first) : ""} to ${last ? formatDay(last) : ""}.`;
  return (
    <>
      <p className="report-summary-line">
        <span className="legend legend-reviews">{formatCount(reviews)} reviews</span>
        <span className="legend legend-ratings">{formatCount(ratings)} ratings</span>
        {busiest && <span>Busiest day: {formatDay(busiest.date)} ({busiest.reviews + busiest.ratings})</span>}
      </p>
      {reviews + ratings === 0 ? (
        <p className="report-empty">No reviews or ratings in this window. Try a longer one.</p>
      ) : (
        <div className="activity-chart" role="img" aria-label={summary}>
          {points.map((p) => (
            <div key={p.date} className="activity-day" title={`${formatDay(p.date)}: ${p.reviews} reviews, ${p.ratings} ratings`}>
              <span className="bar bar-reviews" style={{ height: `${(p.reviews / max) * 100}%` }} />
              <span className="bar bar-ratings" style={{ height: `${(p.ratings / max) * 100}%` }} />
            </div>
          ))}
        </div>
      )}
      <p className="activity-axis" aria-hidden="true"><span>{first && formatDay(first)}</span><span>{last && formatDay(last)}</span></p>
      <details className="report-details">
        <summary>Show the numbers</summary>
        <div className="table-wrap">
          <table className="report-table compact">
            <caption className="sr-only">Reviews and ratings per day</caption>
            <thead><tr><th scope="col">Day (UTC)</th><th scope="col" className="num">Reviews</th><th scope="col" className="num">Ratings</th></tr></thead>
            <tbody>
              {[...points].reverse().map((p) => <tr key={p.date}><td>{p.date}</td><td className="num">{p.reviews}</td><td className="num">{p.ratings}</td></tr>)}
            </tbody>
          </table>
        </div>
      </details>
    </>
  );
}
