import Link from "next/link";
import type { ReactNode } from "react";

const primaryNav = [["⌂", "Dashboard", "/dashboard"], ["▣", "Catalog", "/catalog"], ["▤", "My shelves", "/shelves"], ["◷", "Reading progress", "/progress"]];
const insightNav = [["▥", "Reports", "/reports"], ["✦", "Authors & categories", "/authors"], ["▢", "Reviews", "/reviews"]];

export function AppShell({ children }: { children: ReactNode }) {
  return <div className="app-shell"><aside className="app-sidebar"><Link href="/dashboard" className="brand-lockup"><span className="brand-mark">▥</span><span>Book <strong>Metaverse</strong><small>Your reading universe</small></span></Link><nav className="sidebar-nav"><p>Workspace</p>{primaryNav.map(([icon, label, href]) => <Link key={href} href={href}><span>{icon}</span>{label}</Link>)}<p>Discover & reflect</p>{insightNav.map(([icon, label, href]) => <Link key={href} href={href}><span>{icon}</span>{label}</Link>)}<p>Account</p><Link href="/settings"><span>⚙</span>Settings</Link></nav><div className="sidebar-note"><span>✧</span><strong>Your space is ready</strong><small>Start by adding your first book.</small><Link href="/catalog">Explore catalog →</Link></div></aside><div className="app-main"><header className="app-header"><button className="mobile-menu" aria-label="Open navigation">☰</button><div className="header-search">⌕ <input placeholder="Search books, authors, categories..." aria-label="Global search" /></div><div className="header-actions"><button aria-label="Notifications">♧</button><span className="profile-avatar">Y</span><span className="profile-name">Your profile</span></div></header><main className="workspace-content">{children}</main></div></div>;
}
