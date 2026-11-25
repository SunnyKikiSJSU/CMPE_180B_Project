"use client";

import Link from "next/link";
import { BookCarousel } from "@/components/BookCarousel";
import { PublicNav } from "@/components/PublicNav";
import { LoadingState } from "@/components/LoadingState";
import { useLibrarySearch } from "@/hooks/use-books";

const moods = [["☼", "Something hopeful", "Gentle, bright, uplifting"], ["♡", "A little romance", "Slow burns & happy endings"], ["✦", "Big ideas", "Books to make you think"], ["☾", "Escape for a while", "Faraway worlds await"]];

export function PublicHome() {
  const featured = useLibrarySearch({ sort: "popular", page: 1, pageSize: 3 });
  // On an error the hero shows the same invitation as an empty catalog (FRONTEND_PLAN.md §5.6).
  const featuredBooks = featured.data?.items ?? [];

  return <main className="public-home"><PublicNav /><section className="public-home-hero"><div className="public-hero-copy"><p className="eyebrow">Your reading universe</p><h1>Stories worth <em>savoring.</em></h1><p>A warm little corner for curious readers. Discover books, then sign in when you are ready to make them yours.</p><div className="hero-actions"><Link className="button button-primary" href="/explore">Explore the catalog →</Link><a className="hero-secondary" href="#discover"><span>↓</span> Browse by mood</a></div></div>{featured.isPending ? <LoadingState message="Finding featured books…" cards={3} />  : featuredBooks.length > 0 ? <BookCarousel books={featuredBooks} /> : <div className="empty-state"><span className="empty-state-icon">▤</span><h3>The catalog is ready to explore</h3><p>Search for a book and make it part of your reading universe.</p><Link className="button button-primary" href="/explore">Explore books →</Link></div>}</section><section id="discover" className="public-discover"><div className="public-section-heading"><div><p className="eyebrow">A place to begin</p><h2>What are you in the mood for?</h2></div><Link href="/explore">Open catalog →</Link></div><div className="mood-grid">{moods.map(([icon, title, description]) => <a className="public-mood-card" href={`/explore?mood=${encodeURIComponent(title)}`} key={title}><span>{icon}</span><strong>{title}</strong><small>{description}</small></a>)}</div></section><footer className="public-footer"><span className="public-brand">Book Metaverse<span>.</span></span><span>Discover freely. Personalize after sign in.</span><span>Book information from Open Library</span><span>© {new Date().getFullYear()} Book Metaverse</span></footer></main>;
}
