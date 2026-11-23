"use client";

import Link from "next/link";

import { CoverImage } from "@/components/books/CoverImage";
import { OpenLibraryLink } from "@/components/books/OpenLibraryLink";
import { ErrorState } from "@/components/ErrorState";
import { useBook } from "@/hooks/use-books";
import { catalogHref } from "@/lib/catalog-params";
import { isApiError } from "@/lib/errors";
import { formatPublishedDate, formatRating } from "@/lib/format";
import type { Book, ReadingStatus } from "@/types/domain";

const statusLabels: Record<ReadingStatus, string> = { want_to_read: "Want to read", reading: "Reading", completed: "Finished", paused: "Paused" };

/** /books/[bookId]: catalog details, rating summary, and the reader's own progress and rating. */
export function BookDetailView({ bookId }: { bookId: string }) {
  const book = useBook(bookId);
  if (book.isPending) return <div className="book-detail" role="status" aria-live="polite"><p className="results-summary">Loading the book…</p></div>;
  if (book.isError) {
    const missing = isApiError(book.error) && book.error.status === 404;
    return (
      <div className="section-page">
        <section className="panel section-panel">
          {missing ? (
            <div className="empty-state">
              <span className="empty-state-icon" aria-hidden="true">?</span>
              <h3>We could not find that book</h3>
              <p>It may have been removed, or the link is wrong.</p>
              <Link className="button button-primary" href="/catalog">Back to the catalog →</Link>
            </div>
          ) : (
            <ErrorState error={book.error} onRetry={() => void book.refetch()} />
          )}
        </section>
      </div>
    );
  }
  return <BookDetail book={book.data} />;
}

function BookDetail({ book }: { book: Book }) {
  const published = formatPublishedDate(book.publishedDate, book.publishedDatePrecision);
  const facts: [string, string][] = [
    ["Published", published],
    ["Publisher", book.publisher ?? ""],
    ["Pages", book.pageCount ? String(book.pageCount) : ""],
    ["ISBN-13", book.isbn13 ?? ""],
  ].filter((fact): fact is [string, string] => Boolean(fact[1]));
  const progress = book.myProgress;
  return (
    <article className="book-detail">
      <Link className="text-link back-link" href="/catalog?tab=library">← Catalog</Link>
      <div className="book-detail-grid">
        <div className="book-detail-cover"><CoverImage src={book.coverUrl} title={book.title} size="detail" /></div>
        <div className="book-detail-main">
          <p className="eyebrow">{book.source === "openlibrary" ? "From Open Library" : "In the library"}</p>
          <h1>{book.title}</h1>
          <p className="book-detail-authors">
            {book.authors.length
              ? book.authors.map((a, i) => <span key={a.id}>{i > 0 && ", "}<Link href={catalogHref({ tab: "library", authorId: a.id })}>{a.name}</Link></span>)
              : book.author}
          </p>
          <p className="book-detail-rating">
            <strong>★ {formatRating(book.averageRating, book.ratingCount)}</strong>
            <span>{book.reviewCount ?? 0} {book.reviewCount === 1 ? "review" : "reviews"}</span>
          </p>
          {book.categories.length > 0 && (
            <p className="book-card-tags book-detail-tags">
              {book.categories.map((c) => <Link key={c.id} href={catalogHref({ tab: "library", categoryId: c.id })}>{c.name}</Link>)}
            </p>
          )}
          {book.description ? <p className="book-detail-description">{book.description}</p> : <p className="book-detail-description muted">No description yet.</p>}
          {facts.length > 0 && (
            <dl className="book-detail-facts">
              {facts.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}
            </dl>
          )}
          {book.source === "openlibrary" && <p className="book-detail-source">Book information from Open Library. <OpenLibraryLink href={book.infoLink} /></p>}
        </div>
        <aside className="panel book-detail-activity" aria-label="Your reading">
          <p className="eyebrow">Your reading</p>
          {progress ? (
            <>
              <h2>{statusLabels[progress.status]}</h2>
              <p>{progress.totalPages > 0 ? `Page ${progress.currentPage} of ${progress.totalPages}` : `${Math.round(progress.percentComplete ?? 0)}% read`}</p>
            </>
          ) : (
            <h2>Not on your shelves yet</h2>
          )}
          <p>{book.myRating ? `You rated it ${book.myRating} of 5.` : "You have not rated it."}</p>
        </aside>
      </div>
    </article>
  );
}
