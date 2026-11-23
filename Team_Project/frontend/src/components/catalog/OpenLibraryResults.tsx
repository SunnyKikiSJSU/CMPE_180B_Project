"use client";

import Link from "next/link";
import type { ReactNode } from "react";

import { BookGrid, ExternalBookCard } from "@/components/books/BookCard";
import { Pagination } from "@/components/books/Pagination";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { useExternalSearch } from "@/hooks/use-books";
import { EXTERNAL_MAX_PAGE } from "@/lib/catalog-params";
import { isApiError } from "@/lib/errors";
import type { ExternalSearch } from "@/services/book-service";
import type { ExternalBook } from "@/types/domain";

/**
 * Open Library results for a search (shared by /catalog and /explore). Each
 * page decides the card action and where "search the library instead" goes.
 */
export function OpenLibraryResults({ filters, hrefFor, action, libraryHref }: { filters: ExternalSearch; hrefFor: (page: number) => string; action: (book: ExternalBook) => ReactNode; libraryHref: string }) {
  const search = useExternalSearch(filters);
  if (search.isPending) return <LoadingState message="Searching Open Library… this can take a few seconds." />;
  if (search.isError) {
    const upstream = isApiError(search.error) && (search.error.code === "UPSTREAM_UNAVAILABLE" || search.error.status === 504);
    return (
      <ErrorState error={search.error} title={upstream ? "Open Library is not responding" : isApiError(search.error) && search.error.status === 400 ? "Check the search" : "The search did not work"} onRetry={() => void search.refetch()}>
        {upstream && <Link className="button button-secondary" href={libraryHref}>Search the library instead</Link>}
      </ErrorState>
    );
  }
  const { items, total, page, pageSize } = search.data;
  if (items.length === 0) {
    return (
      <div className="empty-state">
        <span className="empty-state-icon" aria-hidden="true">⌕</span>
        <h3>No Open Library matches</h3>
        <p>Try fewer words, check the spelling, or search by author or ISBN.</p>
      </div>
    );
  }
  return (
    <>
      <p className="results-summary" aria-live="polite">
        About {total.toLocaleString("en-US")} {total === 1 ? "result" : "results"} from Open Library
        {search.isPlaceholderData && <span className="results-updating"> · Updating…</span>}
      </p>
      <BookGrid busy={search.isPlaceholderData} label="Open Library results">
        {items.map((book) => <ExternalBookCard key={book.externalId} book={book} action={action(book)} />)}
      </BookGrid>
      <Pagination page={page} pageSize={pageSize} total={total} hrefFor={hrefFor} approximate maxPage={EXTERNAL_MAX_PAGE} />
    </>
  );
}
