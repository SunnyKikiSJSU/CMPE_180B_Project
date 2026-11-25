"use client";

import Link from "next/link";
import type { ReactNode } from "react";

import { CoverImage } from "@/components/books/CoverImage";
import { ErrorState } from "@/components/ErrorState";
import { reportHref } from "@/lib/page-params";
import type { ReportResult } from "@/types/domain";

/** A titled report block with a link to the SQL that produced it (X-Query-Id). */
export function ReportSection({ title, description, queryId, controls, children }: { title: string; description: string; queryId?: string | null; controls?: ReactNode; children: ReactNode }) {
  return (
    <section className="report-section" aria-label={title}>
      <div className="report-heading">
        <div>
          <h2>{title}</h2>
          <p>{description}</p>
        </div>
        <div className="report-tools">
          {controls}
          {queryId && <Link className="query-badge" href={reportHref({ tab: "queries", sql: queryId })} title="Show the SQL for this report">SQL {queryId}</Link>}
        </div>
      </div>
      {children}
    </section>
  );
}

export type ReportQuery<T> = { isPending: boolean; isError: boolean; error: unknown; data?: ReportResult<T>; isPlaceholderData?: boolean; refetch: () => unknown };

/** Loading, error, and empty states for one report; `children` gets the data. */
export function ReportBody<T>({ query, isEmpty, empty, children }: { query: ReportQuery<T>; isEmpty?: (data: T) => boolean; empty: string; children: (data: T) => ReactNode }) {
  if (query.isPending) return <p className="muted-note" role="status">Running the report…</p>;
  if (query.isError || !query.data) return <ErrorState error={query.error} title="This report did not load" onRetry={() => void query.refetch()} />;
  if (isEmpty?.(query.data.data)) return <p className="report-empty">{empty}</p>;
  return <div className={query.isPlaceholderData ? "is-busy report-body" : "report-body"} aria-busy={query.isPlaceholderData}>{children(query.data.data)}</div>;
}

/** A labelled select that changes a URL parameter. */
export function ReportSelect({ label, value, options, format, onChange }: { label: string; value: number; options: readonly number[]; format: (n: number) => string; onChange: (n: number) => void }) {
  return (
    <label className="report-select">
      <span>{label}</span>
      <select value={value} onChange={(e) => onChange(Number(e.target.value))}>
        {options.map((n) => <option key={n} value={n}>{format(n)}</option>)}
      </select>
    </label>
  );
}

/** A book cell: cover, linked title, and author. */
export function BookCell({ id, title, author, cover }: { id: string; title: string; author?: string; cover?: string | null }) {
  return (
    <Link className="report-book" href={`/books/${id}`} prefetch={false}>
      <CoverImage src={cover} title={title} size="thumb" />
      <span><strong>{title}</strong>{author && <small>{author}</small>}</span>
    </Link>
  );
}
