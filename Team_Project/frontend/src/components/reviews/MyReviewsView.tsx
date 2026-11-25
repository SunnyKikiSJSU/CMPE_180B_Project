"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";

import { CoverImage } from "@/components/books/CoverImage";
import { Pagination } from "@/components/books/Pagination";
import { StarRating } from "@/components/books/StarRating";
import { ConfirmButton } from "@/components/ConfirmButton";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { PageTabs } from "@/components/PageTabs";
import { useClearRating, useDeleteReview, useMyRatings, useMyReviews, useRate, useUpdateReview } from "@/hooks/use-reviews";
import { catalogHref } from "@/lib/catalog-params";
import { errorMessage, isApiError } from "@/lib/errors";
import { formatCount, formatDate } from "@/lib/format";
import { parseReviewsParams, reviewsHref, type ReviewsState } from "@/lib/page-params";
import { reviewSchema } from "@/lib/review-schema";
import type { Page, Rating, Review } from "@/types/domain";

const textSchema = reviewSchema.omit({ rating: true });
type TextValues = { title: string; body: string };

/** /reviews: the reader's reviews (edit, delete) and ratings (change, clear), each linked to its book. */
export function MyReviewsView() {
  const state = parseReviewsParams(useSearchParams());
  return (
    <div className="section-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">Reflect</p>
          <h1>Ratings &amp; reviews</h1>
          <p className="heading-copy">Everything you have rated or reviewed, newest first. Change a rating with the stars; open a book to see what other readers wrote.</p>
        </div>
      </div>
      <PageTabs label="Reviews or ratings" current={state.tab} tabs={[
        { id: "reviews", label: "My reviews", href: reviewsHref({ tab: "reviews" }) },
        { id: "ratings", label: "My ratings", href: reviewsHref({ tab: "ratings" }) },
      ]} />
      <section className="panel section-panel" role="tabpanel" aria-label={state.tab === "ratings" ? "My ratings" : "My reviews"}>
        {state.tab === "ratings" ? <RatingsTab state={state} /> : <ReviewsTab state={state} />}
      </section>
    </div>
  );
}

const findBooks = catalogHref({ tab: "library" });

/** Shared list states: loading, error, nothing at all, and a page past the end (after a delete). */
function ListFrame<T>({ query, state, noun, empty, children }: {
  query: { isPending: boolean; isError: boolean; error: unknown; data?: Page<T>; isPlaceholderData: boolean; refetch: () => unknown };
  state: ReviewsState;
  noun: [string, string];
  empty: { icon: string; title: string; description: string };
  children: (items: T[]) => ReactNode;
}) {
  if (query.isPending) return <p className="muted-note" role="status">Loading your {noun[1]}…</p>;
  if (query.isError || !query.data) return <ErrorState error={query.error} title={`Your ${noun[1]} did not load`} onRetry={() => void query.refetch()} />;
  const { items, total, page, pageSize } = query.data;
  if (total === 0) return <EmptyState {...empty} action="Find a book" href={findBooks} />;
  if (items.length === 0) {
    const last = Math.max(1, Math.ceil(total / pageSize));
    return (
      <div className="empty-state">
        <h3>This page is empty now</h3>
        <Link className="button button-primary" href={reviewsHref({ tab: state.tab, page: last })}>Go to page {last} →</Link>
      </div>
    );
  }
  return (
    <>
      <p className="results-summary" aria-live="polite">
        {formatCount(total)} {total === 1 ? noun[0] : noun[1]}
        {query.isPlaceholderData && <span className="results-updating"> · Updating…</span>}
      </p>
      <div className={`review-list${query.isPlaceholderData ? " is-busy" : ""}`} aria-busy={query.isPlaceholderData}>{children(items)}</div>
      <Pagination page={page} pageSize={pageSize} total={total} hrefFor={(p) => reviewsHref({ tab: state.tab, page: p })} />
    </>
  );
}

function ReviewsTab({ state }: { state: ReviewsState }) {
  const reviews = useMyReviews(state.page);
  return (
    <ListFrame query={reviews} state={state} noun={["review", "reviews"]}
      empty={{ icon: "★", title: "No reviews yet", description: "Open any book in the library and write what you thought of it. Your reviews collect here." }}>
      {(items) => items.map((review) => <MyReviewItem key={review.id} review={review} />)}
    </ListFrame>
  );
}

function BookHeading({ bookId, title, cover }: { bookId: string; title: string | undefined; cover: string | null | undefined }) {
  const name = title || "Untitled book";
  return (
    <Link className="my-item-book" href={`/books/${bookId}`} prefetch={false}>
      <CoverImage src={cover} title={name} size="thumb" />
      <strong>{name}</strong>
    </Link>
  );
}

function MyReviewItem({ review }: { review: Review }) {
  const [editing, setEditing] = useState(false);
  const remove = useDeleteReview(review.bookId);
  return (
    <article className="review-card my-item" aria-label={`Your review of ${review.bookTitle ?? "a book"}`}>
      <div className="my-item-head">
        <BookHeading bookId={review.bookId} title={review.bookTitle} cover={review.bookCoverUrl} />
        <RatingControl bookId={review.bookId} value={review.rating} label={`Your rating of ${review.bookTitle ?? "this book"}`} />
        <time dateTime={review.createdAt}>{formatDate(review.createdAt)}</time>
      </div>
      {editing ? (
        <ReviewTextForm review={review} onDone={() => setEditing(false)} />
      ) : (
        <>
          {review.title && <h3>{review.title}</h3>}
          <p>{review.body}</p>
          {review.updatedAt && review.updatedAt !== review.createdAt && <small className="muted-note">Edited {formatDate(review.updatedAt)}</small>}
          <div className="review-actions">
            <button type="button" className="text-button" onClick={() => setEditing(true)}>Edit</button>
            <ConfirmButton label="Delete" question="Delete this review? Your rating stays." confirmLabel="Delete" pending={remove.isPending} onConfirm={() => remove.mutate(review.id)} />
          </div>
          {remove.isError && <p className="field-error" role="alert">{errorMessage(remove.error)}</p>}
        </>
      )}
    </article>
  );
}

/** PATCH /api/reviews/{id}: title and body; the stars above change the rating. */
function ReviewTextForm({ review, onDone }: { review: Review; onDone: () => void }) {
  const update = useUpdateReview(review.bookId);
  const { register, handleSubmit, setError, formState: { errors } } = useForm<TextValues>({
    resolver: zodResolver(textSchema),
    defaultValues: { title: review.title ?? "", body: review.body },
  });
  const onSubmit = handleSubmit((values) =>
    update.mutate(
      // A blank title is sent as "" so the API clears it.
      { reviewId: review.id, input: { title: values.title, body: values.body } },
      {
        onSuccess: onDone,
        onError: (error) => {
          if (isApiError(error) && (error.field === "title" || error.field === "body")) setError(error.field, { message: error.message });
        },
      },
    ),
  );
  const formError = update.error && !(isApiError(update.error) && (update.error.field === "title" || update.error.field === "body")) ? update.error : null;
  return (
    <form className="review-form" onSubmit={onSubmit} noValidate aria-label="Edit your review">
      {formError && <p className="form-alert" role="alert">{errorMessage(formError)}</p>}
      <label>
        Title <small>(optional)</small>
        <input maxLength={200} aria-invalid={!!errors.title} {...register("title")} />
        {errors.title && <span className="field-error">{errors.title.message}</span>}
      </label>
      <label>
        Review
        <textarea rows={5} maxLength={5000} aria-invalid={!!errors.body} {...register("body")} />
        {errors.body && <span className="field-error">{errors.body.message}</span>}
      </label>
      <div className="review-actions">
        <button className="button button-primary" type="submit" disabled={update.isPending}>{update.isPending ? "Saving…" : "Save changes"}</button>
        <button className="text-button" type="button" onClick={onDone} disabled={update.isPending}>Cancel</button>
      </div>
    </form>
  );
}

/** Stars that save on click (POST /api/books/{bookId}/ratings), with the API error under them. */
function RatingControl({ bookId, value, label }: { bookId: string; value: number | null; label: string }) {
  const rate = useRate(bookId);
  return (
    <span className="my-item-rating">
      <StarRating value={value} label={label} disabled={rate.isPending} onChange={(v) => v !== value && rate.mutate(v)} />
      {rate.isError && <span className="field-error" role="alert">{errorMessage(rate.error)}</span>}
    </span>
  );
}

function RatingsTab({ state }: { state: ReviewsState }) {
  const ratings = useMyRatings(state.page);
  return (
    <ListFrame query={ratings} state={state} noun={["rating", "ratings"]}
      empty={{ icon: "☆", title: "No ratings yet", description: "Rate a book from its page with one to five stars. Your ratings collect here." }}>
      {(items) => items.map((rating) => <MyRatingItem key={rating.bookId} rating={rating} />)}
    </ListFrame>
  );
}

function MyRatingItem({ rating }: { rating: Rating }) {
  const clear = useClearRating(rating.bookId);
  return (
    <article className="review-card my-item" aria-label={`Your rating of ${rating.bookTitle ?? "a book"}`}>
      <div className="my-item-head">
        <BookHeading bookId={rating.bookId} title={rating.bookTitle} cover={rating.bookCoverUrl} />
        <RatingControl bookId={rating.bookId} value={rating.value} label={`Your rating of ${rating.bookTitle ?? "this book"}`} />
        {rating.updatedAt && <time dateTime={rating.updatedAt}>Rated {formatDate(rating.updatedAt)}</time>}
      </div>
      <div className="review-actions">
        <ConfirmButton label="Remove rating" question="Remove this rating?" confirmLabel="Remove" pending={clear.isPending} onConfirm={() => clear.mutate()} />
      </div>
      {clear.isError && <p className="field-error" role="alert">{errorMessage(clear.error)}</p>}
    </article>
  );
}
