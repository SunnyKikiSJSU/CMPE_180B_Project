import Link from "next/link";
import { EmptyState } from "@/components/EmptyState";
import { FilterBar } from "@/components/FilterBar";
import { PublicNav } from "@/components/PublicNav";

export default async function ExplorePage({ searchParams }: { searchParams: Promise<{ mood?: string }> }) {
  const { mood } = await searchParams;
  return <main className="public-explore"><PublicNav /><section className="explore-heading"><div><p className="eyebrow">Public catalog</p><h1>{mood ? `${mood} books` : "Find your next book."}</h1><p>Browse the shared catalog freely. Sign in only when you want to save, track, rate, or review a book.</p></div><Link className="button button-primary" href="/login">Sign in to personalize →</Link></section><section className="explore-card"><FilterBar searchPlaceholder="Search by title, author, or category..." filters={["All categories", "All authors", "Sort by"]} /><div className="explore-empty"><EmptyState icon="⌕" title="The catalog is waiting for its first records" description="Books, authors, categories, and cover images will appear here once the database is connected." action="Sign in to continue" href="/login" /></div></section></main>;
}
