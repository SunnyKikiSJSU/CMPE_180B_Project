import Link from "next/link";
import { EmptyState } from "@/components/EmptyState";
import { FilterBar } from "@/components/FilterBar";

const statCards = [
  ["Books read", "0", "Start your history", "▤", "stat-blue"],
  ["Currently reading", "0", "Nothing in progress", "◷", "stat-lilac"],
  ["Pages read", "0", "Your pages will appear here", "▥", "stat-sage"],
  ["Average rating", "—", "Rate books to see this", "★", "stat-copper"],
];

export function DashboardView() {
  return <div className="dashboard-page modern-dashboard">
    <div className="dashboard-welcome"><div><p className="eyebrow">Sunday · October 25, 2026</p><h1>Welcome back, reader.</h1><p>Make room for a good story today.</p></div><div className="welcome-orbit"><span>✦</span><span>◌</span><span>·</span></div><Link className="button button-light" href="/catalog">+ Add a book</Link></div>

    <section className="stat-grid modern-stat-grid">{statCards.map(([label, value, hint, icon, tone]) => <article className={`stat-card modern-stat-card ${tone}`} key={label}><div className="stat-top"><span className="stat-icon">{icon}</span><span className="stat-menu">•••</span></div><p>{label}</p><strong>{value}</strong><small>{hint}</small><div className="stat-spark"><i /><i /><i /><i /><i /><i /></div></article>)}</section>

    <section className="dashboard-grid modern-dashboard-grid">
      <article className="panel panel-wide modern-panel reading-panel"><div className="panel-heading"><div><p className="eyebrow">Your library</p><h2>Currently reading</h2></div><Link className="panel-action" href="/shelves">View shelves <span>↗</span></Link></div><EmptyState icon="▤" title="Your reading list is waiting" description="Books you mark as currently reading will appear here with progress, notes, and next steps." action="Find your first book" href="/catalog" /></article>
      <article className="panel modern-panel goal-panel"><div className="panel-heading"><div><p className="eyebrow">Reading goal</p><h2>This year</h2></div><span className="panel-kebab">•••</span></div><div className="goal-ring"><div><strong>0</strong><small>books</small></div></div><p className="center-note">Set a goal when you are ready to begin.</p><Link className="text-link centered" href="/settings">Set reading goal <span>→</span></Link></article>
      <article className="panel panel-wide modern-panel activity-panel"><div className="panel-heading"><div><p className="eyebrow">Your activity</p><h2>Reading history</h2></div><Link className="panel-action" href="/reports">View reports <span>↗</span></Link></div><div className="activity-empty"><div className="timeline-line" /><div className="timeline-item"><span className="timeline-dot" /><div><strong>Your reading story starts here</strong><p>Finished books and progress updates will appear on your timeline.</p></div></div><div className="timeline-item muted"><span className="timeline-dot" /><div><strong>Reports will grow with you</strong><p>Patterns from your reading history will become visible over time.</p></div></div></div></article>
      <article className="panel modern-panel shelves-panel"><div className="panel-heading"><div><p className="eyebrow">Organize</p><h2>Your shelves</h2></div><Link className="panel-action" href="/shelves">Manage <span>↗</span></Link></div><EmptyState icon="▤" title="No shelves yet" description="Create a shelf for books you want to read, love, or remember." action="Create a shelf" href="/shelves" /></article>
    </section>

    <section className="panel modern-panel discover-panel modern-discover"><div className="panel-heading"><div><p className="eyebrow">Start exploring</p><h2>Find something that fits your mood</h2><p className="panel-subtitle">Search the shared catalog by title, author, category, or reading mood.</p></div><Link className="panel-action" href="/catalog">Open catalog <span>↗</span></Link></div><FilterBar searchPlaceholder="Search books, authors, categories..." filters={["Choose a genre", "Choose an author", "Sort by"]} /><div className="discover-empty modern-discover-empty"><span>⌕</span><div><strong>Your recommendations are waiting</strong><p>Connect the catalog to discover books based on your reading activity.</p></div></div></section>

    <section className="quick-actions"><p className="eyebrow">Quick actions</p><div><Link href="/catalog"><span>⌕</span><strong>Find a book</strong><small>Explore the catalog</small></Link><Link href="/shelves"><span>＋</span><strong>Make a shelf</strong><small>Organize your reading</small></Link><Link href="/reviews"><span>★</span><strong>Write a review</strong><small>Share your thoughts</small></Link><Link href="/reports"><span>▥</span><strong>See your reports</strong><small>Reflect on your habits</small></Link></div></section>
  </div>;
}
