"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";

import { PageTabs } from "@/components/PageTabs";
import { QueryCatalog } from "@/components/reports/QueryCatalog";
import { ReviewActivityReport } from "@/components/reports/ReviewActivityReport";
import { MyReadingReport } from "@/components/reports/MyReadingReport";
import { BookCell, ReportBody, ReportSection, ReportSelect, type ReportQuery } from "@/components/reports/ReportParts";
import { useAuthorRankings, useCategoryRankings, usePopular, useTopRated } from "@/hooks/use-reports";
import { formatAverage, formatCount } from "@/lib/format";
import { authorHref, categoryHref, dayOptions, minRatingOptions, parseReportParams, reportHref, reportTabs, type ReportState } from "@/lib/page-params";

const TOP = 20;
const daysLabel = (n: number) => (n === 365 ? "Last year" : `Last ${n} days`);
const minLabel = (n: number) => (n === 0 ? "Any number of ratings" : `At least ${n} rating${n === 1 ? "" : "s"}`);

/** /reports: the course report queries, one tab each, plus the SQL behind them. */
export function ReportsView() {
  const state = parseReportParams(useSearchParams());
  const current = reportTabs.find((t) => t.id === state.tab)?.label ?? "Report";
  return (
    <div className="section-page reports-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">Reflect</p>
          <h1>Reading reports</h1>
          <p className="heading-copy">Your own reading statistics and library-wide rankings, computed live in PostgreSQL. Each report links to the SQL that produced it.</p>
        </div>
      </div>
      <PageTabs label="Reports" current={state.tab} tabs={reportTabs.map((t) => ({ ...t, href: reportHref({ tab: t.id }) }))} />
      <div className="panel section-panel" role="tabpanel" aria-label={current}>
        <ReportTab state={state} />
      </div>
    </div>
  );
}

function ReportTab({ state }: { state: ReportState }) {
  switch (state.tab) {
    case "popular": return <PopularReport state={state} />;
    case "top-rated": return <TopRatedReport state={state} />;
    case "authors": return <AuthorRankingReport state={state} />;
    case "categories": return <CategoryRankingReport state={state} />;
    case "activity": return <ReviewActivityReport state={state} />;
    case "queries": return <QueryCatalog target={state.sql} />;
    default: return <MyReadingReport />;
  }
}

function useGo(state: ReportState) {
  const router = useRouter();
  return (next: Partial<ReportState>) => router.replace(reportHref({ ...state, ...next }), { scroll: false });
}

function PopularReport({ state }: { state: ReportState }) {
  const go = useGo(state);
  const report = usePopular({ days: state.days, limit: TOP });
  return (
    <ReportSection title="Popular books" queryId={report.data?.queryId}
      description="Books added to shelves and rated most often in the window (whole UTC days). Score = 2 × shelf adds + ratings."
      controls={<ReportSelect label="Window" value={state.days} options={dayOptions} format={daysLabel} onChange={(days) => go({ days })} />}>
      <ReportBody query={report} isEmpty={(rows) => rows.length === 0} empty="No shelf adds or ratings in this window. Try a longer one.">
        {(rows) => (
          <div className="table-wrap">
            <table className="report-table">
              <caption className="sr-only">Popular books, {daysLabel(state.days).toLowerCase()}</caption>
              <thead><tr><th scope="col">#</th><th scope="col">Book</th><th scope="col" className="num">Shelf adds</th><th scope="col" className="num">Ratings</th><th scope="col" className="num">Average</th><th scope="col" className="num">Score</th></tr></thead>
              <tbody>
                {rows.map((row, i) => (
                  <tr key={row.bookId}>
                    <td className="rank">{i + 1}</td>
                    <td><BookCell id={row.bookId} title={row.title} author={row.primaryAuthor} cover={row.coverUrl} /></td>
                    <td className="num">{formatCount(row.shelfAdds)}</td>
                    <td className="num">{formatCount(row.ratings)}</td>
                    <td className="num">{formatAverage(row.averageRating, row.ratings)}</td>
                    <td className="num strong">{formatCount(row.score)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ReportBody>
    </ReportSection>
  );
}

function TopRatedReport({ state }: { state: ReportState }) {
  const go = useGo(state);
  const report = useTopRated({ minRatings: state.minRatings, limit: TOP });
  return (
    <ReportSection title="Top rated books" queryId={report.data?.queryId}
      description="Highest average rating, then most ratings, from the trigger-maintained rating summaries."
      controls={<ReportSelect label="Minimum" value={state.minRatings} options={minRatingOptions} format={minLabel} onChange={(minRatings) => go({ minRatings })} />}>
      <ReportBody query={report} isEmpty={(rows) => rows.length === 0} empty="No book has that many ratings yet. Lower the minimum.">
        {(rows) => (
          <div className="table-wrap">
            <table className="report-table">
              <caption className="sr-only">Top rated books</caption>
              <thead><tr><th scope="col">#</th><th scope="col">Book</th><th scope="col" className="num">Average</th><th scope="col" className="num">Ratings</th></tr></thead>
              <tbody>
                {rows.map((row, i) => (
                  <tr key={row.bookId}>
                    <td className="rank">{i + 1}</td>
                    <td><BookCell id={row.bookId} title={row.title} author={row.primaryAuthor} cover={row.coverUrl} /></td>
                    <td className="num strong">★ {formatAverage(row.averageRating, row.ratingCount)}</td>
                    <td className="num">{formatCount(row.ratingCount)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ReportBody>
    </ReportSection>
  );
}

function AuthorRankingReport({ state }: { state: ReportState }) {
  const query = useAuthorRankings({ minRatings: state.minRatings, limit: TOP });
  return <RankingReport state={state} kind="authors" query={query} rowOf={(r) => ({ ...r, id: r.authorId, href: authorHref(r.authorId) })} />;
}

function CategoryRankingReport({ state }: { state: ReportState }) {
  const query = useCategoryRankings({ minRatings: state.minRatings, limit: TOP });
  return <RankingReport state={state} kind="categories" query={query} rowOf={(r) => ({ ...r, id: r.categoryId, href: categoryHref(r.categoryId) })} />;
}

type RankingRow = { id: string; href: string; name: string; bookCount: number; ratingCount: number; averageRating: number; ranking: number };

function RankingReport<T>({ state, kind, query, rowOf }: { state: ReportState; kind: "authors" | "categories"; query: ReportQuery<T[]>; rowOf: (row: T) => RankingRow }) {
  const go = useGo(state);
  const isAuthors = kind === "authors";
  return (
    <ReportSection title={isAuthors ? "Author ranking" : "Category ranking"} queryId={query.data?.queryId}
      description={`${isAuthors ? "Authors" : "Categories"} ranked with RANK() by the average rating of their books, then by number of ratings. Ties share a rank.`}
      controls={<ReportSelect label="Minimum" value={state.minRatings} options={minRatingOptions} format={minLabel} onChange={(minRatings) => go({ minRatings })} />}>
      <ReportBody query={query} isEmpty={(rows) => rows.length === 0} empty={`No ${kind} have that many ratings yet. Lower the minimum.`}>
        {(data) => (
          <div className="table-wrap">
            <table className="report-table">
              <caption className="sr-only">{isAuthors ? "Author" : "Category"} ranking</caption>
              <thead><tr><th scope="col">Rank</th><th scope="col">{isAuthors ? "Author" : "Category"}</th><th scope="col" className="num">Books</th><th scope="col" className="num">Ratings</th><th scope="col" className="num">Average</th></tr></thead>
              <tbody>
                {data.map(rowOf).map((row) => (
                  <tr key={row.id}>
                    <td className="rank">{row.ranking}</td>
                    <td><Link className="text-link report-name" href={row.href} prefetch={false}>{row.name}</Link></td>
                    <td className="num">{formatCount(row.bookCount)}</td>
                    <td className="num">{formatCount(row.ratingCount)}</td>
                    <td className="num strong">★ {formatAverage(row.averageRating, row.ratingCount)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ReportBody>
    </ReportSection>
  );
}
