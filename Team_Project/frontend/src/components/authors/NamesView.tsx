"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import type { FormEvent } from "react";

import { Pagination } from "@/components/books/Pagination";
import { ErrorState } from "@/components/ErrorState";
import { PageTabs } from "@/components/PageTabs";
import { useNameList } from "@/hooks/use-books";
import { formatCount } from "@/lib/format";
import { authorHref, categoryHref, namesHref, parseNamesParams, type NamesState } from "@/lib/page-params";

const PAGE_SIZE = 30;

/** /authors: authors and categories with their book counts; each opens its books. */
export function NamesView() {
  const state = parseNamesParams(useSearchParams());
  return (
    <div className="section-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">Discover</p>
          <h1>Authors &amp; categories</h1>
          <p className="heading-copy">Every author and category in the library, each stored once and linked to its books.</p>
        </div>
      </div>
      <PageTabs label="Authors or categories" current={state.tab} tabs={[
        { id: "authors", label: "Authors", href: namesHref({ tab: "authors" }) },
        { id: "categories", label: "Categories", href: namesHref({ tab: "categories" }) },
      ]} />
      <section className="panel section-panel" role="tabpanel" aria-label={state.tab === "categories" ? "Categories" : "Authors"}>
        <NameList state={state} />
      </section>
    </div>
  );
}

function NameList({ state }: { state: NamesState }) {
  const router = useRouter();
  const list = useNameList(state.tab, { q: state.q || undefined, page: state.page, pageSize: PAGE_SIZE });
  const isAuthors = state.tab === "authors";
  const noun = isAuthors ? ["author", "authors"] : ["category", "categories"];
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const q = String(new FormData(event.currentTarget).get("q") ?? "").trim();
    router.push(namesHref({ tab: state.tab, q, page: 1 }));
  }
  return (
    <>
      <form className="filter-bar" role="search" onSubmit={submit} key={`${state.tab}|${state.q}`}>
        <label className="search-field"><span aria-hidden="true">⌕</span><input name="q" defaultValue={state.q} maxLength={200} placeholder={`Search ${noun[1]} by name…`} aria-label={`Search ${noun[1]}`} /></label>
        <button className="button button-primary" type="submit">Search</button>
        {state.q && <Link className="text-link" href={namesHref({ tab: state.tab })}>Clear</Link>}
      </form>
      {list.isPending ? (
        <p className="muted-note" role="status">Loading {noun[1]}…</p>
      ) : list.isError ? (
        <ErrorState error={list.error} title={`The ${noun[1]} did not load`} onRetry={() => void list.refetch()} />
      ) : list.data.items.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon" aria-hidden="true">✦</span>
          <h3>No {noun[1]} {state.q ? `match “${state.q}”` : "yet"}</h3>
          <p>{state.q ? "Try a shorter or different name." : "They appear when books are added to the library."}</p>
        </div>
      ) : (
        <>
          <p className="results-summary" aria-live="polite">
            {formatCount(list.data.total)} {list.data.total === 1 ? noun[0] : noun[1]}
            {list.isPlaceholderData && <span className="results-updating"> · Updating…</span>}
          </p>
          <ul className={`name-grid${list.isPlaceholderData ? " is-busy" : ""}`} aria-busy={list.isPlaceholderData}>
            {list.data.items.map((item) => (
              <li key={item.id}>
                <Link href={isAuthors ? authorHref(item.id) : categoryHref(item.id)} prefetch={false}>
                  <strong>{item.name}</strong>
                  <span>{formatCount(item.bookCount)} {item.bookCount === 1 ? "book" : "books"}</span>
                </Link>
              </li>
            ))}
          </ul>
          <Pagination page={list.data.page} pageSize={list.data.pageSize} total={list.data.total} hrefFor={(page) => namesHref({ ...state, page })} />
        </>
      )}
    </>
  );
}
