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

function ProgressRow({ progress }: { progress: ReadingProgress }) {
  const update = useUpdateProgress(progress.bookId);
  const [page, setPage] = useState(String(progress.totalPages > 0 ? progress.currentPage : Math.round(progress.percentComplete ?? 0)));
  const [status, setStatus] = useState<ReadingStatus>(progress.status);
  const byPage = progress.totalPages > 0;
  const max = byPage ? progress.totalPages : 100;
  const number = Number(page);
  const valid = page.trim() !== "" && Number.isInteger(number) && number >= 0 && number <= max;
  const stale = isApiError(update.error) && update.error.code === "STALE_VERSION";
  function save() {
    if (!valid) return;
    update.mutate(byPage ? { currentPage: number, status, version: progress.version } : { percentComplete: number, status, version: progress.version });
  }
  return <article className="progress-row"><div className="progress-row-heading"><div><Link href={`/books/${progress.bookId}`}><strong>{progress.book?.title ?? "Untitled book"}</strong></Link><small>{progress.book?.author ?? "Unknown author"}</small></div><strong>{progressPercent(progress)}%</strong></div><div className="progress-bar" role="progressbar" aria-label={`Progress for ${progress.book?.title ?? "book"}`} aria-valuemin={0} aria-valuemax={100} aria-valuenow={progressPercent(progress)}><span style={{ width: `${progressPercent(progress)}%` }} /></div><p className="progress-summary">{progressSummary(progress)} · Updated {formatDate(progress.updatedAt)}</p><div className="progress-row-controls"><label>Status<select value={status} onChange={(event) => setStatus(event.target.value as ReadingStatus)}>{readingStatuses.map((value) => <option value={value} key={value}>{statusLabels[value]}</option>)}</select></label><label>{byPage ? "Current page" : "Percent read"}<input type="number" min={0} max={max} step={1} value={page} onChange={(event) => setPage(event.target.value)} /></label><button className="button button-primary" type="button" disabled={!valid || update.isPending} onClick={save}>{update.isPending ? "Saving…" : "Save"}</button></div>{!valid && <p className="field-error">Enter a whole number from 0 to {max}.</p>}{update.error && <p className="field-error" role="alert">{stale ? "This progress changed elsewhere. The latest version has been loaded; review the values and save again." : errorMessage(update.error)}</p>}</article>;
}

export default function ProgressPage() {
  const progress = useReadingProgress();
  const history = useReadingHistory();
  const [filter, setFilter] = useState<ReadingStatus | "all">("all");
  if (progress.isPending || history.isPending) return <LoadingState message="Loading your reading progress…" cards={5} />;
  if (progress.error) return <ErrorState error={progress.error} onRetry={() => void progress.refetch()} />;
  if (history.error) return <ErrorState error={history.error} onRetry={() => void history.refetch()} />;
  const items = (progress.data ?? []).filter((item) => filter === "all" || item.status === filter);
  return <div className="section-page f5-page"><div className="page-heading"><div><p className="eyebrow">Track</p><h1>Reading progress</h1><p className="heading-copy">Update pages and status while keeping your reading history intact.</p></div><label className="page-filter">Filter<select value={filter} onChange={(event) => setFilter(event.target.value as ReadingStatus | "all")}><option value="all">All statuses</option>{readingStatuses.map((value) => <option value={value} key={value}>{statusLabels[value]}</option>)}</select></label></div><section className="panel progress-list-panel"><div className="panel-heading"><div><p className="eyebrow">Your books</p><h2>{items.length} {items.length === 1 ? "book" : "books"}</h2></div></div>{items.length === 0 ? <EmptyState icon="◷" title="No progress yet" description="Start a book from its detail page and your progress will appear here." action="Browse catalog" href="/catalog" /> : <div className="progress-list">{items.map((item) => <ProgressRow key={`${item.bookId}-${item.version}`} progress={item} />)}</div>}</section><section className="panel history-panel"><div className="panel-heading"><div><p className="eyebrow">Timeline</p><h2>Reading history</h2></div></div>{!history.data?.length ? <EmptyState icon="◷" title="Your history is empty" description="Shelf and progress changes will appear here." /> : <div className="activity-empty">{history.data.map((event) => <div className="timeline-item" key={event.id}><span className="timeline-dot" /><div><strong>{event.bookTitle ?? "Book activity"}</strong><p>{historyLabel(event)} · {formatDate(event.createdAt)}</p></div></div>)}</div>}</section></div>;
}
