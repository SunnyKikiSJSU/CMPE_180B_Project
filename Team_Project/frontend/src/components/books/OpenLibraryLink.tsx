/** Attribution for Open Library data: a link to the work's page (FRONTEND_PLAN.md decision 10). */
export function OpenLibraryLink({ href, label = "View on Open Library" }: { href: string | null | undefined; label?: string }) {
  if (!href || !href.startsWith("https://openlibrary.org/")) return null;
  return <a className="source-link" href={href} target="_blank" rel="noopener noreferrer">{label} <span aria-hidden="true">↗</span></a>;
}
