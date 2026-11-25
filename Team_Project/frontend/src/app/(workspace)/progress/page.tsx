"use client";

import Link from "next/link";
import { useState } from "react";

import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { useReadingHistory, useReadingProgress, useUpdateProgress } from "@/hooks/use-reading";
import { errorMessage, isApiError } from "@/lib/errors";
import { formatDate } from "@/lib/format";
import { historyLabel, progressPercent, progressSummary, readingStatuses, statusLabels } from "@/lib/reading";
import type { ReadingProgress, ReadingStatus } from "@/types/domain";

const PAGE_SIZE = 20;

type ProgressMutation = ReturnType<typeof useUpdateProgress>;

/**
 * One book. The mutation lives here and the form below is keyed by version:
 * after a 409 STALE_VERSION the refetched row remounts the form with the
 * latest values, and the message (held by the mutation) stays visible.
 */
function ProgressRow({ progress }: { progress: ReadingProgress }) {
  const update = useUpdateProgress(progress.bookId);
  const title = progress.book?.title ?? "Untitled book";
  const percent = progressPercent(progress);
  const stale = isApiError(update.error) && update.error.code === "STALE_VERSION";
  return (
    <article className="progress-row">
      <div className="progress-row-heading">
        <div><Link href={`/books/${progress.bookId}`}><strong>{title}</strong></Link><small>{progress.book?.author ?? "Unknown author"}</small></div>
        <strong>{percent}%</strong>
      </div>
      <div className="progress-bar" role="progressbar" aria-label={`Progress for ${title}`} aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}><span style={{ width: `${percent}%` }} /></div>
      <p className="progress-summary">{progressSummary(progress)} · Updated {formatDate(progress.updatedAt)}</p>
      <ProgressForm key={progress.version} progress={progress} title={title} update={update} />
      {update.error && <p className="field-error" role="alert">{stale ? "This book changed in another tab or device. The latest progress is loaded now; check it and save again." : errorMessage(update.error)}</p>}
    </article>
  );
}

function ProgressForm({ progress, title, update }: { progress: ReadingProgress; title: string; update: ProgressMutation }) {
  const byPage = progress.totalPages > 0;
  const [value, setValue] = useState(String(byPage ? progress.currentPage : Math.round(progress.percentComplete ?? 0)));
  const [status, setStatus] = useState<ReadingStatus>(progress.status);
  const max = byPage ? progress.totalPages : 100;
  const number = Number(value);
  const valid = value.trim() !== "" && Number.isInteger(number) && number >= 0 && number <= max;
  function save() {
    if (!valid) return;
    const amount = byPage ? { currentPage: number } : { percentComplete: number };
    update.mutate({ ...amount, ...(status !== progress.status ? { status } : {}), version: progress.version });
  }
  return (
    <>
      <div className="progress-row-controls">
        <label>Status<select value={status} onChange={(event) => setStatus(event.target.value as ReadingStatus)} aria-label={`Status for ${title}`}>{readingStatuses.map((value) => <option value={value} key={value}>{statusLabels[value]}</option>)}</select></label>
        <label>{byPage ? "Current page" : "Percent read"}<input type="number" inputMode="numeric" min={0} max={max} step={1} value={value} onChange={(event) => setValue(event.target.value)} aria-invalid={!valid} aria-label={`${byPage ? "Current page" : "Percent read"} for ${title}`} /></label>
        <button className="button button-primary" type="button" disabled={!valid || update.isPending} onClick={save}>{update.isPending ? "Saving…" : "Save"}</button>
      </div>
      {!valid && <p className="field-error">Enter a whole number from 0 to {max}.</p>}
    </>
  );
}

export default function ProgressPage() {
  const [filter, setFilter] = useState<ReadingStatus | "all">("all");
  const [page, setPage] = useState(1);
  const progress = useReadingProgress({ status: filter === "all" ? undefined : filter, page, pageSize: PAGE_SIZE });
  const history = useReadingHistory();

  const filterControl = (
    <label className="page-filter">Filter<select value={filter} onChange={(event) => { setFilter(event.target.value as ReadingStatus | "all"); setPage(1); }}><option value="all">All statuses</option>{readingStatuses.map((value) => <option value={value} key={value}>{statusLabels[value]}</option>)}</select></label>
  );

  let list;
  if (progress.isPending) list = <LoadingState message="Loading your reading progress…" cards={3} />;
  else if (progress.isError) list = <ErrorState error={progress.error} onRetry={() => void progress.refetch()} />;
  else {
    const { items, total, pageSize } = progress.data;
    const last = Math.max(1, Math.ceil(total / pageSize));
    // Books can leave the last page (status change, another tab); go back in range.
    if (page > last && !progress.isPlaceholderData) setPage(last);
    list = (
      <>
        <div className="panel-heading"><div><p className="eyebrow">Your books</p><h2>{total} {total === 1 ? "book" : "books"}</h2></div></div>
        {items.length === 0 ? (
          <EmptyState icon="◷" title={filter === "all" ? "No progress yet" : `Nothing marked ${statusLabels[filter]}`} description="Start a book from its detail page and your progress will appear here." action="Browse catalog" href="/catalog" />
        ) : (
          <div className={`progress-list${progress.isPlaceholderData ? " is-busy" : ""}`}>{items.map((item) => <ProgressRow key={item.bookId} progress={item} />)}</div>
        )}
        {last > 1 && (
          <nav className="pagination" aria-label="Progress pages">
            <button type="button" className="pagination-link" disabled={page <= 1} onClick={() => setPage(page - 1)}>← Previous</button>
            <span className="pagination-status">Page {page} of {last}</span>
            <button type="button" className="pagination-link" disabled={page >= last} onClick={() => setPage(page + 1)}>Next →</button>
          </nav>
        )}
      </>
    );
  }

  return (
    <div className="section-page f5-page">
      <div className="page-heading"><div><p className="eyebrow">Track</p><h1>Reading progress</h1><p className="heading-copy">Update pages and status while keeping your reading history intact.</p></div>{filterControl}</div>
      <section className="panel progress-list-panel">{list}</section>
      <section className="panel history-panel">
        <div className="panel-heading"><div><p className="eyebrow">Timeline</p><h2>Reading history</h2></div></div>
        {history.isPending ? <p className="muted-note" role="status">Loading history…</p>
          : history.isError ? <ErrorState error={history.error} onRetry={() => void history.refetch()} />
          : !history.data.length ? <EmptyState icon="◷" title="Your history is empty" description="Shelf and progress changes will appear here." />
          : <div className="activity-empty">{history.data.map((event) => <div className="timeline-item" key={event.id}><span className="timeline-dot" /><div><strong>{event.bookTitle || "Book activity"}</strong><p>{historyLabel(event)} · {formatDate(event.createdAt)}</p></div></div>)}</div>}
      </section>
    </div>
  );
}
