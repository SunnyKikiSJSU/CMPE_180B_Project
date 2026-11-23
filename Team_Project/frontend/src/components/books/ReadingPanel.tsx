"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { StarRating } from "@/components/books/StarRating";
import { ConfirmButton } from "@/components/ConfirmButton";
import { useBookHistory, useCustomShelfMembership, useRemoveFromShelves, useShelves, useToggleCustomShelf, useUpdateProgress } from "@/hooks/use-reading";
import { useClearRating, useRate } from "@/hooks/use-reviews";
import { errorMessage, isApiError } from "@/lib/errors";
import { formatDate } from "@/lib/format";
import { historyLabel, progressPercent, progressSummary, readingStatuses, statusLabels } from "@/lib/reading";
import type { Book, ReadingProgress, ReadingStatus } from "@/types/domain";

/** The book page's side panel: reading status, progress, custom shelves, rating, and recent history. */
export function ReadingPanel({ book }: { book: Book }) {
  const progress = book.myProgress ?? null;
  // The mutations live here, not in the forms: a new version remounts
  // ProgressForm, and its STALE_VERSION message must survive that reload.
  const statusUpdate = useUpdateProgress(book.id);
  const pageUpdate = useUpdateProgress(book.id);
  return (
    <aside className="panel book-detail-activity" aria-label="Your reading">
      <p className="eyebrow">Your reading</p>
      <StatusControl bookId={book.id} progress={progress} update={statusUpdate} onSaved={pageUpdate.reset} />
      {progress && <ProgressForm key={`${progress.version}-${progress.status}`} progress={progress} update={pageUpdate} onSaved={statusUpdate.reset} />}
      <RatingRow book={book} />
      <CustomShelves bookId={book.id} />
      {progress && <RecentHistory bookId={book.id} />}
    </aside>
  );
}

type ProgressMutation = ReturnType<typeof useUpdateProgress>;

function StatusControl({ bookId, progress, update, onSaved }: { bookId: string; progress: ReadingProgress | null; update: ProgressMutation; onSaved: () => void }) {
  const remove = useRemoveFromShelves(bookId);
  const setStatus = (status: ReadingStatus) => update.mutate({ status, ...(progress ? { version: progress.version } : {}) }, { onSuccess: onSaved });
  const stale = isApiError(update.error) && update.error.code === "STALE_VERSION";
  return (
    <div className="reading-status">
      <h2>{progress ? statusLabels[progress.status] : "Not on your shelves yet"}</h2>
      <div className="status-options" role="group" aria-label="Reading status">
        {readingStatuses.map((status) => (
          <button key={status} type="button" aria-pressed={progress?.status === status} disabled={update.isPending || progress?.status === status} onClick={() => setStatus(status)}>
            {statusLabels[status]}
          </button>
        ))}
      </div>
      {update.isError && (
        <p className="field-error" role="alert">
          {stale ? "This book changed in another tab or device. The latest version is loaded now; choose again." : errorMessage(update.error)}
        </p>
      )}
      {progress && (
        <ConfirmButton label="Remove from my shelves" question="Remove it and its progress?" confirmLabel="Remove" pending={remove.isPending} onConfirm={() => remove.mutate()} />
      )}
      {remove.isError && <p className="field-error" role="alert">{errorMessage(remove.error)}</p>}
    </div>
  );
}

function ProgressForm({ progress, update, onSaved }: { progress: ReadingProgress; update: ProgressMutation; onSaved: () => void }) {
  const byPage = progress.totalPages > 0;
  const [value, setValue] = useState(String(byPage ? progress.currentPage : Math.round(progress.percentComplete ?? 0)));
  const max = byPage ? progress.totalPages : 100;
  const number = Number(value);
  const valid = value.trim() !== "" && Number.isInteger(number) && number >= 0 && number <= max;
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!valid) return;
    update.mutate(byPage ? { currentPage: number, version: progress.version } : { percentComplete: number, version: progress.version }, { onSuccess: onSaved });
  }
  const stale = isApiError(update.error) && update.error.code === "STALE_VERSION";
  const percent = progressPercent(progress);
  return (
    <form className="progress-form" onSubmit={submit} noValidate>
      <div className="progress-bar" role="progressbar" aria-label="Reading progress" aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}><span style={{ width: `${percent}%` }} /></div>
      <p className="progress-summary">{progressSummary(progress)}</p>
      {progress.status !== "want_to_read" && (
        <>
          <label>
            {byPage ? "Current page" : "Percent read"}
            <span className="progress-input">
              <input type="number" inputMode="numeric" min={0} max={max} step={1} value={value} onChange={(e) => setValue(e.target.value)} aria-invalid={!valid} />
              <span>{byPage ? `of ${progress.totalPages}` : "%"}</span>
            </span>
          </label>
          {!valid && <span className="field-error">Enter a whole number from 0 to {max}.</span>}
          <button className="button button-primary" type="submit" disabled={!valid || update.isPending}>{update.isPending ? "Saving…" : "Save progress"}</button>
          {byPage && <small className="muted-note">Reaching the last page marks the book finished.</small>}
        </>
      )}
      {update.isError && (
        <p className="field-error" role="alert">{stale ? "This book changed in another tab or device. The latest progress is loaded now; check it and save again." : errorMessage(update.error)}</p>
      )}
    </form>
  );
}

function RatingRow({ book }: { book: Book }) {
  const rate = useRate(book.id);
  const clear = useClearRating(book.id);
  const error = rate.error ?? clear.error;
  return (
    <div className="rating-row">
      <h3>Your rating</h3>
      <StarRating value={book.myRating ?? null} label="Your rating" disabled={rate.isPending || clear.isPending} onChange={(value) => value !== book.myRating && rate.mutate(value)} />
      {book.myRating ? <button type="button" className="text-button" onClick={() => clear.mutate()} disabled={clear.isPending}>Clear rating</button> : <small className="muted-note">Not rated yet</small>}
      {error && <p className="field-error" role="alert">{errorMessage(error)}</p>}
    </div>
  );
}

function CustomShelves({ bookId }: { bookId: string }) {
  const shelves = useShelves();
  const membership = useCustomShelfMembership(bookId, shelves.data);
  const toggle = useToggleCustomShelf(bookId);
  if (shelves.isPending) return null;
  return (
    <fieldset className="custom-shelves">
      <legend>My shelves</legend>
      {membership.length === 0 ? (
        <p className="muted-note">No custom shelves yet. <Link className="text-link" href="/shelves">Create one</Link></p>
      ) : (
        membership.map(({ shelf, onShelf, loading }) => (
          <label key={shelf.id}>
            <input type="checkbox" checked={Boolean(onShelf)} disabled={loading || toggle.isPending} onChange={(e) => toggle.mutate({ shelfId: shelf.id, add: e.target.checked })} />
            {shelf.name}
          </label>
        ))
      )}
      {toggle.isError && <p className="field-error" role="alert">{errorMessage(toggle.error)}</p>}
    </fieldset>
  );
}

function RecentHistory({ bookId }: { bookId: string }) {
  const history = useBookHistory(bookId);
  if (!history.data?.length) return null;
  return (
    <div className="recent-history">
      <h3>Recent activity</h3>
      <ol>
        {history.data.map((event) => <li key={event.id}><span>{historyLabel(event)}</span><time dateTime={event.createdAt}>{formatDate(event.createdAt)}</time></li>)}
      </ol>
    </div>
  );
}
