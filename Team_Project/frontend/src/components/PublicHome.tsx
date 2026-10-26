import Link from "next/link";
import { BookCarousel } from "@/components/BookCarousel";
import { PublicNav } from "@/components/PublicNav";
import { featuredBooks } from "@/data/mock-data";

const moods = [["☼", "Something hopeful", "Gentle, bright, uplifting"], ["♡", "A little romance", "Slow burns & happy endings"], ["✦", "Big ideas", "Books to make you think"], ["☾", "Escape for a while", "Faraway worlds await"]];

export function PublicHome() {
  return <main className="public-home"><PublicNav /><section className="public-home-hero"><div className="public-hero-copy"><p className="eyebrow">Your reading universe</p><h1>Stories worth <em>savoring.</em></h1><p>A warm little corner for curious readers. Discover books, then sign in when you are ready to make them yours.</p><div className="hero-actions"><Link className="button button-primary" href="/explore">Explore the catalog →</Link><a className="hero-secondary" href="#discover"><span>↓</span> Browse by mood</a></div></div><BookCarousel books={featuredBooks} /></section><section id="discover" className="public-discover"><div className="public-section-heading"><div><p className="eyebrow">A place to begin</p><h2>What are you in the mood for?</h2></div><Link href="/explore">Open catalog →</Link></div><div className="mood-grid">{moods.map(([icon, title, description]) => <a className="public-mood-card" href={`/explore?mood=${encodeURIComponent(title)}`} key={title}><span>{icon}</span><strong>{title}</strong><small>{description}</small></a>)}</div></section><footer className="public-footer"><span className="public-brand">Book Metaverse<span>.</span></span><span>Discover freely. Personalize after sign in.</span><span>© 2026 Book Metaverse</span></footer></main>;
}
