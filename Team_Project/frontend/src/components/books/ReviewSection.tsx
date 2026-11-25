"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { StarRating } from "@/components/books/StarRating";
import { ConfirmButton } from "@/components/ConfirmButton";
import { ErrorState } from "@/components/ErrorState";
import { useMe } from "@/hooks/use-auth";
import { useBookReviews, useDeleteReview, useMyReview, useSaveReview } from "@/hooks/use-reviews";
import { errorMessage, isApiError } from "@/lib/errors";
import { formatDate } from "@/lib/format";
import { reviewSchema, type ReviewOutput, type ReviewValues } from "@/lib/review-schema";
import type { Book, Review } from "@/types/domain";

/** The reader's own review (write, edit, delete) and everyone else's, newest first. */
export function ReviewSection({ book }: { book: Book }) {
  return (
    <section className="panel review-section" aria-labelledby="reviews-heading">
      <div className="panel-heading"><h2 id="reviews-heading">Reviews</h2></div>
      <MyReview book={book} />
      <OtherReviews bookId={book.id} />
    </section>
  );
}

function MyReview({ book }: { book: Book }) {
  const mine = useMyReview(book.id);
  const remove = useDeleteReview(book.id);
  const [editing, setEditing] = useState(false);
  if (mine.isPending) return null;
  if (mine.isError) return <ErrorState error={mine.error} title="Your review did not load" onRetry={() => void mine.refetch()} />;
  const review = mine.data;
  if (editing) return <ReviewForm book={book} review={review} onDone={() => setEditing(false)} />;
  if (!review) {
    return (
      <div className="my-review-empty">
        <p>Share what you thought of <em>{book.title}</em>.</p>
        <button className="button button-primary" type="button" onClick={() => setEditing(true)}>Write a review</button>
      </div>
    );
  }
  return (
    <article className="review-card is-mine" aria-label="Your review">
      <ReviewBody review={review} author="Your review" rating={book.myRating ?? review.rating} />
      <div className="review-actions">
        <button type="button" className="text-button" onClick={() => setEditing(true)}>Edit</button>
        <ConfirmButton label="Delete" question="Delete your review?" confirmLabel="Delete" pending={remove.isPending} onConfirm={() => remove.mutate(review.id)} />
      </div>
      {remove.isError && <p className="field-error" role="alert">{errorMessage(remove.error)}</p>}
    </article>
  );
}

function ReviewForm({ book, review, onDone }: { book: Book; review: Review | null; onDone: () => void }) {
  const save = useSaveReview(book.id);
  const { register, control, handleSubmit, setError, formState: { errors } } = useForm<ReviewValues, unknown, ReviewOutput>({
    resolver: zodResolver(reviewSchema),
    defaultValues: { rating: book.myRating ?? review?.rating ?? null, title: review?.title ?? "", body: review?.body ?? "" },
  });
  const onSubmit = handleSubmit((values) =>
    save.mutate(
      { body: values.body, title: values.title, ...(values.rating ? { rating: values.rating } : {}) },
      {
        onSuccess: onDone,
        onError: (error) => {
          if (isApiError(error) && (error.field === "title" || error.field === "body" || error.field === "rating")) setError(error.field, { message: error.message });
        },
      },
    ),
  );
  const formError = save.error && !(isApiError(save.error) && ["title", "body", "rating"].includes(save.error.field ?? "")) ? save.error : null;
  return (
    <form className="review-form" onSubmit={onSubmit} noValidate aria-label={review ? "Edit your review" : "Write a review"}>
      {formError && <p className="form-alert" role="alert">{errorMessage(formError)}</p>}
      <div className="review-form-rating">
        <span>Rating <small>(optional)</small></span>
        <Controller control={control} name="rating" render={({ field }) => <StarRating value={field.value} onChange={field.onChange} label="Rating for this review" />} />
        {errors.rating && <span className="field-error">Pick 1 to 5 stars.</span>}
      </div>
      <label>
        Title <small>(optional)</small>
        <input maxLength={200} aria-invalid={!!errors.title} {...register("title")} />
        {errors.title && <span className="field-error">{errors.title.message}</span>}
      </label>
      <label>
        Review
        <textarea rows={6} maxLength={5000} aria-invalid={!!errors.body} {...register("body")} />
        {errors.body && <span className="field-error">{errors.body.message}</span>}
      </label>
      <div className="review-actions">
        <button className="button button-primary" type="submit" disabled={save.isPending}>{save.isPending ? "Saving…" : review ? "Save changes" : "Post review"}</button>
        <button className="text-button" type="button" onClick={onDone} disabled={save.isPending}>Cancel</button>
      </div>
    </form>
  );
}

function OtherReviews({ bookId }: { bookId: string }) {
  const me = useMe();
  const [page, setPage] = useState(1);
  const reviews = useBookReviews(bookId, page);
  if (reviews.isPending) return <p className="muted-note" role="status">Loading reviews…</p>;
  if (reviews.isError) return <ErrorState error={reviews.error} title="Reviews did not load" onRetry={() => void reviews.refetch()} />;
  const others = reviews.data.items.filter((r) => r.userId !== me.data?.id);
  const last = Math.max(1, Math.ceil(reviews.data.total / reviews.data.pageSize));
  if (reviews.data.total === 0) return <p className="muted-note">No reviews yet.</p>;
  return (
    <>
      <p className="results-summary">{reviews.data.total} {reviews.data.total === 1 ? "review" : "reviews"}</p>
      <div className={`review-list${reviews.isPlaceholderData ? " is-busy" : ""}`}>
        {others.map((review) => (
          <article className="review-card" key={review.id}>
            <ReviewBody review={review} author={review.user?.displayName ?? "A reader"} rating={review.rating} />
          </article>
        ))}
      </div>
      {last > 1 && (
        <nav className="pagination" aria-label="Review pages">
          <button type="button" className="pagination-link" disabled={page <= 1} onClick={() => setPage(page - 1)}>← Newer</button>
          <span className="pagination-status">Page {page} of {last}</span>
          <button type="button" className="pagination-link" disabled={page >= last} onClick={() => setPage(page + 1)}>Older →</button>
        </nav>
      )}
    </>
  );
}

function ReviewBody({ review, author, rating }: { review: Review; author: string; rating: number | null | undefined }) {
  return (
    <>
      <header>
        <strong>{author}</strong>
        {rating ? <span className="review-stars" role="img" aria-label={`${rating} of 5 stars`}>{"★".repeat(rating)}<span aria-hidden="true">{"★".repeat(5 - rating)}</span></span> : null}
        <time dateTime={review.createdAt}>{formatDate(review.createdAt)}</time>
      </header>
      {review.title && <h3>{review.title}</h3>}
      <p>{review.body}</p>
    </>
  );
}
