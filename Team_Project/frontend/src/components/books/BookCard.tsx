import Link from "next/link";
import type { ReactNode } from "react";

import { CoverImage } from "@/components/books/CoverImage";
import { OpenLibraryLink } from "@/components/books/OpenLibraryLink";
import { formatRating } from "@/lib/format";
import type { Book, ExternalBook } from "@/types/domain";

/** A book already in the library; the whole card opens its detail page. */
export function LibraryBookCard({ book }: { book: Book }) {
  const authors = book.authors?.length ? book.authors.map((a) => a.name).join(", ") : book.author;
  return (
    <article className="book-card">
      <Link className="book-card-link" href={`/books/${book.id}`} prefetch={false}>
        <CoverImage src={book.coverUrl} title={book.title} />
        <div className="book-card-body">
          <h3>{book.title}</h3>
          <p className="book-card-authors">{authors}</p>
          <p className="book-card-meta">★ {formatRating(book.averageRating, book.ratingCount)}</p>
          {book.categories?.length > 0 && <p className="book-card-tags">{book.categories.slice(0, 2).map((c) => <span key={c.id}>{c.name}</span>)}</p>}
        </div>
      </Link>
      {book.source === "openlibrary" && <p className="book-card-source">From Open Library</p>}
    </article>
  );
}

/** An Open Library search result; `action` is the call to action the page allows. */
export function ExternalBookCard({ book, action }: { book: ExternalBook; action: ReactNode }) {
  const meta = [book.publishedDate, book.publisher, book.pageCount ? `${book.pageCount} pages` : null].filter(Boolean).join(" · ");
  return (
    <article className="book-card">
      <div className="book-card-link">
        <CoverImage src={book.coverUrl} title={book.title} />
        <div className="book-card-body">
          {book.bookId && <span className="book-card-badge">In the library</span>}
          <h3>{book.title}</h3>
          {book.subtitle && <p className="book-card-subtitle">{book.subtitle}</p>}
          <p className="book-card-authors">{book.authors.length ? book.authors.join(", ") : "Unknown author"}</p>
          {meta && <p className="book-card-meta">{meta}</p>}
          {book.categories.length > 0 && <p className="book-card-tags">{book.categories.slice(0, 3).map((c) => <span key={c}>{c}</span>)}</p>}
        </div>
      </div>
      <div className="book-card-actions">
        {action}
        <OpenLibraryLink href={book.infoLink} label="Open Library" />
      </div>
    </article>
  );
}

/** Dims the previous results while the next page or search loads. */
export function BookGrid({ children, busy = false, label }: { children: ReactNode; busy?: boolean; label: string }) {
  return <div className={`book-grid${busy ? " is-busy" : ""}`} role="group" aria-busy={busy} aria-label={label}>{children}</div>;
}
