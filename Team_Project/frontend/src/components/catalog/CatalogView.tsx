"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import type { FormEvent } from "react";

import { BookGrid, LibraryBookCard } from "@/components/books/BookCard";
import { Pagination } from "@/components/books/Pagination";
import { AddToLibraryButton } from "@/components/catalog/AddToLibraryButton";
import { OpenLibraryResults } from "@/components/catalog/OpenLibraryResults";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { useCategoryOptions, useLibrarySearch } from "@/hooks/use-books";
import { catalogHref, hasExternalTerms, parseCatalogParams, toExternalSearch, toLibrarySearch, type CatalogState } from "@/lib/catalog-params";

const suggestions = [{ label: "Dune", q: "dune" }, { label: "Toni Morrison", author: "toni morrison" }, { label: "Poetry", subject: "poetry" }, { label: "Mystery", subject: "mystery" }];

/** /catalog: Open Library search (default) and the local library, with all state in the URL. */
export function CatalogView() {
  const state = parseCatalogParams(useSearchParams());
  // The library has one search box, so an Open Library title or author search carries over as its text.
  const carried = state.q || state.title || state.author;
  const tab = (target: CatalogState["tab"], label: string) => (
    <Link role="tab" aria-selected={state.tab === target} className="catalog-tab" href={catalogHref({ tab: target, q: carried })}>{label}</Link>
  );
  return (
    <div className="section-page catalog-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">Explore</p>
          <h1>Book catalog</h1>
          <p className="heading-copy">Find any book on Open Library and add it to the shared library, or browse what readers have already added.</p>
        </div>
      </div>
      <div className="catalog-tabs" role="tablist" aria-label="Catalog source">
        {tab("openlibrary", "Open Library")}
        {tab("library", "In the library")}
      </div>
      <section className="panel catalog-panel" role="tabpanel" aria-label={state.tab === "library" ? "In the library" : "Open Library"}>
        {state.tab === "library" ? <LibraryTab state={state} /> : <OpenLibraryTab state={state} />}
      </section>
    </div>
  );
}

function useGo(state: CatalogState) {
  const router = useRouter();
  return (next: Partial<CatalogState>) => router.push(catalogHref({ ...state, page: 1, ...next }));
}

const field = (form: FormData, name: string) => String(form.get(name) ?? "").trim();

function OpenLibraryTab({ state }: { state: CatalogState }) {
  const go = useGo(state);
  const searching = hasExternalTerms(state);
  const advanced = Boolean(state.title || state.author || state.subject || state.isbn);
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    go({ q: field(form, "q"), title: field(form, "title"), author: field(form, "author"), subject: field(form, "subject"), isbn: field(form, "isbn") });
  }
  // key: the uncontrolled inputs reset when the URL changes (back/forward, a suggestion).
  const formKey = [state.q, state.title, state.author, state.subject, state.isbn].join("|");
  return (
    <>
      <form className="catalog-search" role="search" onSubmit={submit} key={formKey}>
        <div className="filter-bar">
          <label className="search-field"><span aria-hidden="true">⌕</span><input name="q" defaultValue={state.q} placeholder="Search Open Library by title, author, or keyword…" aria-label="Search Open Library" maxLength={200} /></label>
          <select aria-label="Sort Open Library results" value={state.sort} onChange={(e) => go({ sort: e.target.value as CatalogState["sort"] })} disabled={!searching}>
            <option value="relevance">Best match</option>
            <option value="newest">Newest first</option>
          </select>
          <button className="button button-primary" type="submit">Search</button>
        </div>
        <details className="catalog-advanced" open={advanced}>
          <summary>More filters</summary>
          <div className="catalog-advanced-fields">
            <label>Title<input name="title" defaultValue={state.title} maxLength={200} /></label>
            <label>Author<input name="author" defaultValue={state.author} maxLength={200} /></label>
            <label>Subject<input name="subject" defaultValue={state.subject} maxLength={100} placeholder="fantasy, history…" /></label>
            <label>ISBN<input name="isbn" defaultValue={state.isbn} inputMode="numeric" placeholder="10 or 13 digits" /></label>
          </div>
        </details>
      </form>
      {searching ? (
        <OpenLibraryResults
          filters={toExternalSearch(state)}
          hrefFor={(page) => catalogHref({ ...state, page })}
          action={(book) => <AddToLibraryButton book={book} />}
          libraryHref={catalogHref({ tab: "library", q: state.q || state.title || state.author })}
        />
      ) : (
        <div className="empty-state catalog-prompt">
          <span className="empty-state-icon" aria-hidden="true">⌕</span>
          <h3>Search millions of books</h3>
          <p>Results come from Open Library. Add one to the library to shelve, track, rate, and review it.</p>
          <div className="suggestion-chips" aria-label="Try a search">
            {suggestions.map(({ label, ...terms }) => <Link key={label} href={catalogHref({ tab: "openlibrary", ...terms })}>{label}</Link>)}
          </div>
          <Link className="text-link" href={catalogHref({ tab: "library" })}>Or browse the books already in the library →</Link>
        </div>
      )}
    </>
  );
}

function LibraryTab({ state }: { state: CatalogState }) {
  const go = useGo(state);
  const search = useLibrarySearch(toLibrarySearch(state));
  const categories = useCategoryOptions();
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    go({ q: field(new FormData(event.currentTarget), "q") });
  }
  const authorName = state.authorId ? search.data?.items.flatMap((b) => b.authors).find((a) => a.id === state.authorId)?.name : undefined;
  return (
    <>
      <form className="catalog-search filter-bar" role="search" onSubmit={submit} key={state.q}>
        <label className="search-field"><span aria-hidden="true">⌕</span><input name="q" defaultValue={state.q} placeholder="Search the library by title, author, or category…" aria-label="Search books in the library" maxLength={200} /></label>
        <select aria-label="Category" value={state.categoryId} onChange={(e) => go({ categoryId: e.target.value })}>
          <option value="">All categories</option>
          {categories.data?.items.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <select aria-label="Sort library results" value={state.sort} onChange={(e) => go({ sort: e.target.value as CatalogState["sort"] })}>
          <option value="relevance">{state.q ? "Best match" : "Newest first"}</option>
          {state.q && <option value="newest">Newest first</option>}
          <option value="rating">Highest rated</option>
          <option value="popular">Most popular</option>
          <option value="title">Title A–Z</option>
        </select>
        <button className="button button-primary" type="submit">Search</button>
      </form>
      {state.authorId && (
        <p className="active-filters">
          <span>Author: {authorName ?? "selected author"}</span>
          <Link href={catalogHref({ ...state, authorId: "", page: 1 })} aria-label="Remove the author filter">✕</Link>
        </p>
      )}
      {search.isPending ? (
        <LoadingState message="Searching the library…" />
      ) : search.isError ? (
        <ErrorState error={search.error} onRetry={() => void search.refetch()} />
      ) : search.data.items.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon" aria-hidden="true">□</span>
          <h3>No books in the library match</h3>
          <p>{state.q ? "It may not have been added yet." : "Try another category or clear the filters."}</p>
          {state.q && <Link className="button button-primary" href={catalogHref({ tab: "openlibrary", q: state.q })}>Search Open Library for “{state.q}” →</Link>}
        </div>
      ) : (
        <>
          <p className="results-summary" aria-live="polite">
            {search.data.total.toLocaleString("en-US")} {search.data.total === 1 ? "book" : "books"} in the library
            {search.isPlaceholderData && <span className="results-updating"> · Updating…</span>}
          </p>
          <BookGrid busy={search.isPlaceholderData} label="Library results">
            {search.data.items.map((book) => <LibraryBookCard key={book.id} book={book} />)}
          </BookGrid>
          <Pagination page={search.data.page} pageSize={search.data.pageSize} total={search.data.total} hrefFor={(page) => catalogHref({ ...state, page })} />
        </>
      )}
    </>
  );
}
