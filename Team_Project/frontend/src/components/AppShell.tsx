"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useRef, useState, type FormEvent, type ReactNode } from "react";

import { Drawer } from "@/components/Drawer";
import { useLogout, useMe } from "@/hooks/use-auth";
import { catalogHref } from "@/lib/catalog-params";

const primaryNav = [["⌂", "Dashboard", "/dashboard"], ["▣", "Catalog", "/catalog"], ["▤", "My shelves", "/shelves"], ["◷", "Reading progress", "/progress"]];
const insightNav = [["▥", "Reports", "/reports"], ["✦", "Authors & categories", "/authors"], ["▢", "Reviews", "/reviews"]];

export function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { data: user } = useMe();
  const logout = useLogout();
  const [search, setSearch] = useState("");
  // On phones the sidebar is hidden (globals.css), so ☰ opens the same links in a drawer.
  const [menuOpen, setMenuOpen] = useState(false);
  const menuButton = useRef<HTMLButtonElement>(null);
  const closeMenu = () => setMenuOpen(false);

  // Category pages (/categories/[id]) belong to "Authors & categories".
  const section = pathname.startsWith("/categories/") ? "/authors" : pathname;
  // The icons are decorative, so screen readers hear only the page name.
  const navLinks = (onNavigate?: () => void) => {
    const navLink = ([icon, label, href]: string[]) => (
      <Link key={href} href={href} onClick={onNavigate} aria-current={section === href || section.startsWith(`${href}/`) ? "page" : undefined}>
        <span aria-hidden="true">{icon}</span>{label}
      </Link>
    );
    return (
      <>
        <p>Workspace</p>{primaryNav.map(navLink)}
        <p>Discover & reflect</p>{insightNav.map(navLink)}
        <p>Account</p>{navLink(["⚙", "Settings", "/settings"])}
      </>
    );
  };

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const q = search.trim();
    router.push(catalogHref({ tab: "library", q }));
  }

  function signOut() {
    logout.mutate(undefined, { onSuccess: () => router.replace("/login") });
  }

  const name = user?.displayName ?? "";
  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <Link href="/dashboard" className="brand-lockup"><span className="brand-mark">▥</span><span>Book <strong>Metaverse</strong><small>Your reading universe</small></span></Link>
        <nav className="sidebar-nav" aria-label="Workspace">{navLinks()}</nav>
        <div className="sidebar-note"><span>✧</span><strong>Your space is ready</strong><small>Start by adding your first book.</small><Link href="/catalog">Explore catalog →</Link></div>
      </aside>
      <div className="app-main">
        <header className="app-header">
          <button ref={menuButton} className="mobile-menu" type="button" onClick={() => setMenuOpen(true)} aria-label="Open navigation" aria-haspopup="dialog" aria-expanded={menuOpen} aria-controls="workspace-drawer">☰</button>
          <form className="header-search" role="search" onSubmit={submitSearch}>
            <span aria-hidden="true">⌕</span> <input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search books, authors, categories..." aria-label="Search the library" />
          </form>
          <div className="header-actions">
            <span className="profile-avatar" aria-hidden="true">{name ? name.charAt(0).toUpperCase() : "·"}</span>
            <span className="profile-name">{name}</span>
            <button className="header-logout" type="button" onClick={signOut} disabled={logout.isPending}>{logout.isPending ? "Signing out…" : "Sign out"}</button>
          </div>
        </header>
        <main className="workspace-content">{children}</main>
      </div>
      <Drawer id="workspace-drawer" label="Workspace navigation" open={menuOpen} onClose={closeMenu} returnFocusTo={menuButton}>
        <p className="eyebrow">Book Metaverse</p>
        <nav className="sidebar-nav" aria-label="Workspace">{navLinks(closeMenu)}</nav>
      </Drawer>
    </div>
  );
}
