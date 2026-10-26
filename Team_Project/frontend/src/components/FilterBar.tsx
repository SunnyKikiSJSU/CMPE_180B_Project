"use client";

export function FilterBar({ searchPlaceholder = "Search your library...", filters = ["All statuses", "All genres", "Recently updated"] }: { searchPlaceholder?: string; filters?: string[] }) {
  return <div className="filter-bar"><label className="search-field"><span>⌕</span><input placeholder={searchPlaceholder} aria-label={searchPlaceholder} /></label>{filters.map((filter) => <select key={filter} aria-label={filter} defaultValue=""><option value="" disabled>{filter}</option><option>All</option><option>Recently added</option><option>Alphabetical</option></select>)}</div>;
}
