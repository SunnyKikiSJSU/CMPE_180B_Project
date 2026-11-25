"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import { BookGrid, LibraryBookCard } from "@/components/books/BookCard";
import { Pagination } from "@/components/books/Pagination";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { useNameDetail } from "@/hooks/use-books";
import { catalogHref } from "@/lib/catalog-params";
import { isApiError } from "@/lib/errors";
import { formatCount } from "@/lib/format";
import { hrefWith, namesHref, pageParam } from "@/lib/page-params";

type Kind = "author" | "category";

/** /authors/[authorId] and /categories/[categoryId]: the name, its book count, and its books, paged. */
export function NameDetailView({ kind, id }: { kind: Kind; id: string }) {
  const page = pageParam(useSearchParams().get("page"));
  const query = useNameDetail(kind, id, page);
  const label = kind === "author" ? "Author" : "Category";
  const listHref = namesHref({ tab: kind === "author" ? "authors" : "categories" });
  const back = <Link className="text-link back-link" href={listHref}>← Authors &amp; categories</Link>;

  if (query.isPending) return <div className="section-page">{back}<LoadingState message={`Loading the ${label.toLowerCase()}…`} /></div>;
  if (query.isError) {
    const missing = isApiError(query.error) && query.error.status === 404;
    return (
      <div className="section-page">
        {back}
        <section className="panel">
          {missing ? (
            <div className="empty-state">
              <h3>We could not find that {label.toLowerCase()}</h3>
              <p>The link may be wrong.</p>
              <Link className="button button-primary" href={listHref}>Browse all {kind === "author" ? "authors" : "categories"} →</Link>
            </div>
          ) : (
            <ErrorState error={query.error} onRetry={() => void query.refetch()} />
          )}
        </section>
      </div>
    );
  }

  const { detail, books } = query.data;
  const count = detail.bookCount ?? books.total;
  const searchHref = catalogHref(kind === "author" ? { tab: "library", authorId: id } : { tab: "library", categoryId: id });
  const path = `/${kind === "author" ? "authors" : "categories"}/${encodeURIComponent(id)}`;
  return (
    <div className="section-page">
      {back}
      <div className="page-heading">
        <div>
          <p className="eyebrow">{label}</p>
          <h1>{detail.name}</h1>
          <p className="heading-copy">{formatCount(count)} {count === 1 ? "book" : "books"} in the library.</p>
        </div>
        <Link className="button button-secondary" href={searchHref}>Filter in the catalog <span>→</span></Link>
      </div>
      <section className="panel section-panel" aria-label={`Books ${kind === "author" ? "by" : "in"} ${detail.name}`}>
        {books.items.length === 0 ? (
          <div className="empty-state">
            <h3>{books.total > 0 ? "This page is empty" : "No books yet"}</h3>
            {books.total > 0 && <Link className="button button-primary" href={path}>Back to page 1 →</Link>}
          </div>
        ) : (
          <>
            <BookGrid busy={query.isPlaceholderData} label={`Books ${kind === "author" ? "by" : "in"} ${detail.name}`}>
              {books.items.map((book) => <LibraryBookCard key={book.id} book={book} />)}
            </BookGrid>
            <Pagination page={books.page} pageSize={books.pageSize} total={books.total} hrefFor={(p) => hrefWith(path, { page: p }, { page: 1 })} />
          </>
        )}
      </section>
    </div>
  );
}
