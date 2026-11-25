import Link from "next/link";

/** Tabs that are links, so the selected tab is in the URL (reload and back keep it). */
export function PageTabs({ label, tabs, current }: { label: string; tabs: readonly { id: string; label: string; href: string }[]; current: string }) {
  return (
    <div className="catalog-tabs page-tabs" role="tablist" aria-label={label}>
      {tabs.map((tab) => (
        <Link key={tab.id} role="tab" aria-selected={tab.id === current} className="catalog-tab" href={tab.href} scroll={false}>{tab.label}</Link>
      ))}
    </div>
  );
}
