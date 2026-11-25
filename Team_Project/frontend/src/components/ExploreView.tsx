"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import type { FormEvent } from "react";

import { AddToLibraryButton } from "@/components/catalog/AddToLibraryButton";
import { OpenLibraryResults } from "@/components/catalog/OpenLibraryResults";
import { PublicNav } from "@/components/PublicNav";
import { useMe } from "@/hooks/use-auth";
import { catalogHref, EXTERNAL_MAX_PAGE, EXTERNAL_PAGE_SIZE } from "@/lib/catalog-params";

/** The home page's mood cards link here as ?mood=; each maps to an Open Library subject. */
export const moodSubjects: Record<string, string> = {
  "Something hopeful": "hope",
  "A little romance": "romance",
  "Big ideas": "philosophy",
  "Escape for a while": "fantasy",
};

function exploreHref(q: string, mood: string, page = 1): string {
  const search = new URLSearchParams();
  if (q) search.set("q", q);
  if (mood) search.set("mood", mood);
  if (page > 1) search.set("page", String(page));
  const s = search.toString();
  return s ? `/explore?${s}` : "/explore";
}

/** Public /explore: anyone can search Open Library; saving a book needs an account. */
export function ExploreView() {
  const params = useSearchParams();
  const router = useRouter();
  const me = useMe();
  const q = (params.get("q") ?? "").trim();
  const rawMood = params.get("mood") ?? "";
  const mood = Object.hasOwn(moodSubjects, rawMood) ? rawMood : "";
  const rawPage = Number.parseInt(params.get("page") ?? "", 10);
  const page = Number.isFinite(rawPage) && rawPage > 0 ? Math.min(rawPage, EXTERNAL_MAX_PAGE) : 1;
  const here = exploreHref(q, mood, page);
  const libraryHref = catalogHref({ tab: "library", q });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    router.push(exploreHref(String(new FormData(event.currentTarget).get("q") ?? "").trim(), mood));
  }

  return (
    <main className="public-explore">
      <PublicNav />
      <section className="explore-heading">
        <div>
          <p className="eyebrow">Public catalog</p>
          <h1>{mood ? `${mood}.` : "Find your next book."}</h1>
          <p>Search millions of books on Open Library. Sign in only when you want to save, track, rate, or review one.</p>
        </div>
        {!me.data && <Link className="button button-primary" href={`/login?returnTo=${encodeURIComponent(here)}`}>Sign in to personalize →</Link>}
      </section>
      <section className="explore-card">
        <form className="filter-bar" role="search" onSubmit={submit} key={q}>
          <label className="search-field"><span aria-hidden="true">⌕</span><input name="q" defaultValue={q} placeholder="Search by title, author, or keyword…" aria-label="Search books" maxLength={200} /></label>
          <button className="button button-primary" type="submit">Search</button>
        </form>
        <div className="suggestion-chips explore-moods" role="group" aria-label="Browse by mood">
          {Object.keys(moodSubjects).map((name) => <Link key={name} href={exploreHref(q, mood === name ? "" : name)} aria-current={mood === name ? "true" : undefined}>{name}</Link>)}
        </div>
        {q || mood ? (
          <OpenLibraryResults
            filters={{ q: q || undefined, category: mood ? moodSubjects[mood] : undefined, page, pageSize: EXTERNAL_PAGE_SIZE }}
            hrefFor={(target) => exploreHref(q, mood, target)}
            action={(book) => (me.data ? <AddToLibraryButton book={book} /> : <Link className="button button-secondary" href={`/login?returnTo=${encodeURIComponent(here)}`}>Sign in to save</Link>)}
            libraryHref={me.data ? libraryHref : `/login?returnTo=${encodeURIComponent(libraryHref)}`}
          />
        ) : (
          <div className="explore-empty">
            <div className="empty-state">
              <span className="empty-state-icon" aria-hidden="true">⌕</span>
              <h3>What would you like to read?</h3>
              <p>Search for a title or author, or pick a mood above.</p>
            </div>
          </div>
        )}
      </section>
    </main>
  );
}
