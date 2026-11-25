"use client";

import Link from "next/link";

import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { useDashboardData } from "@/hooks/use-dashboard";
import { useMe } from "@/hooks/use-auth";
import { formatDate } from "@/lib/format";
import type { ReadingStatus } from "@/types/domain";

const statusLabels: Record<ReadingStatus, string> = {
  want_to_read: "Want to read",
  reading: "Currently reading",
  completed: "Finished",
  paused: "Paused",
};

function formatPercent(percent: number | undefined): string {
  return `${Math.round(percent ?? 0)}%`;
}

export function DashboardView() {
  const me = useMe();
  const dashboard = useDashboardData();

  if (dashboard.isPending) return <LoadingState message="Loading your reading dashboard…" cards={4} />;
  if (dashboard.error) return <ErrorState error={dashboard.error} onRetry={dashboard.retry} />;

  const stats = dashboard.stats.data;
  const progress = dashboard.progress.data ?? [];
  const shelves = dashboard.shelves.data ?? [];
  const history = dashboard.history.data ?? [];
  const currentlyReading = progress.filter((item) => item.status === "reading").slice(0, 3);
  const userName = me.data?.displayName ?? "reader";
  const today = new Intl.DateTimeFormat("en-US", { dateStyle: "full" }).format(new Date());
  const statCards = [
    ["Books read", String(stats?.booksCompleted ?? 0), "Completed books", "▤", "stat-blue"],
    ["Currently reading", String(stats?.booksInProgress ?? 0), "Books in progress", "◷", "stat-lilac"],
    ["Pages read", String(stats?.pagesRead ?? 0), "Pages across your history", "▥", "stat-sage"],
    ["Average rating", stats?.averageRating ? stats.averageRating.toFixed(2) : "—", `${stats?.ratingsGiven ?? 0} ratings given`, "★", "stat-copper"],
  ];

  return <div className="dashboard-page modern-dashboard">
    <div className="dashboard-welcome"><div><p className="eyebrow">{today}</p><h1>Welcome back, {userName}.</h1><p>Make room for a good story today.</p></div><div className="welcome-orbit"><span>✦</span><span>◌</span><span>·</span></div><Link className="button button-light" href="/catalog">+ Add a book</Link></div>

    <section className="stat-grid modern-stat-grid">{statCards.map(([label, value, hint, icon, tone]) => <article className={`stat-card modern-stat-card ${tone}`} key={label}><div className="stat-top"><span className="stat-icon">{icon}</span><span className="stat-menu">•••</span></div><p>{label}</p><strong>{value}</strong><small>{hint}</small><div className="stat-spark"><i /><i /><i /><i /><i /><i /></div></article>)}</section>

    <section className="dashboard-grid modern-dashboard-grid">
      <article className="panel panel-wide modern-panel reading-panel"><div className="panel-heading"><div><p className="eyebrow">Your library</p><h2>Currently reading</h2></div><Link className="panel-action" href="/progress">View progress <span>↗</span></Link></div>{currentlyReading.length === 0 ? <EmptyState icon="▤" title="Your reading list is waiting" description="Books you mark as currently reading will appear here with progress, notes, and next steps." action="Find your first book" href="/catalog" /> : <div className="dashboard-book-list">{currentlyReading.map((item) => <Link className="dashboard-book-row" href={`/books/${item.bookId}`} key={item.bookId}><div><strong>{item.book?.title ?? "Untitled book"}</strong><small>{item.book?.author ?? "Unknown author"}</small></div><span>{formatPercent(item.percentComplete)}</span></Link>)}</div>}</article>
      <article className="panel modern-panel goal-panel"><div className="panel-heading"><div><p className="eyebrow">Reading goal</p><h2>This year</h2></div><span className="panel-kebab">•••</span></div><div className="goal-ring"><div><strong>{stats?.booksCompleted ?? 0}</strong><small>books</small></div></div><p className="center-note">{stats?.booksCompleted ? "Keep your reading momentum going." : "Set a goal when you are ready to begin."}</p><Link className="text-link centered" href="/settings">Set reading goal <span>→</span></Link></article>
      <article className="panel panel-wide modern-panel activity-panel"><div className="panel-heading"><div><p className="eyebrow">Your activity</p><h2>Reading history</h2></div><Link className="panel-action" href="/reports">View reports <span>↗</span></Link></div>{history.length === 0 ? <EmptyState icon="◷" title="Your reading story starts here" description="Finished books and progress updates will appear on your timeline." /> : <div className="activity-empty"><div className="timeline-line" />{history.slice(0, 3).map((event) => <div className="timeline-item" key={event.id}><span className="timeline-dot" /><div><strong>{event.bookTitle ?? "Book activity"}</strong><p>{event.toStatus ? statusLabels[event.toStatus] : event.eventType} · {formatDate(event.createdAt)}</p></div></div>)}</div>}</article>
      <article className="panel modern-panel shelves-panel"><div className="panel-heading"><div><p className="eyebrow">Organize</p><h2>Your shelves</h2></div><Link className="panel-action" href="/shelves">Manage <span>↗</span></Link></div>{shelves.length === 0 ? <EmptyState icon="▤" title="No shelves yet" description="Create a shelf for books you want to read, love, or remember." action="Create a shelf" href="/shelves" /> : <div className="dashboard-shelf-list">{shelves.slice(0, 5).map((shelf) => <Link href="/shelves" className="dashboard-shelf-row" key={shelf.id}><span>{shelf.name}</span><strong>{shelf.count}</strong></Link>)}</div>}</article>
    </section>

    <section className="panel modern-panel discover-panel modern-discover"><div className="panel-heading"><div><p className="eyebrow">Start exploring</p><h2>Find something that fits your mood</h2><p className="panel-subtitle">Search the shared catalog by title, author, category, or reading mood.</p></div><Link className="panel-action" href="/catalog">Open catalog <span>↗</span></Link></div><div className="discover-empty modern-discover-empty"><span>⌕</span><div><strong>Your recommendations are waiting</strong><p>Explore the catalog to find your next book.</p></div></div></section>

    <section className="quick-actions"><p className="eyebrow">Quick actions</p><div><Link href="/catalog"><span>⌕</span><strong>Find a book</strong><small>Explore the catalog</small></Link><Link href="/shelves"><span>＋</span><strong>Make a shelf</strong><small>Organize your reading</small></Link><Link href="/reviews"><span>★</span><strong>Write a review</strong><small>Share your thoughts</small></Link><Link href="/reports"><span>▥</span><strong>See your reports</strong><small>Reflect on your habits</small></Link></div></section>
  </div>;
}
