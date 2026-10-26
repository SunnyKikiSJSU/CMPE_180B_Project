import { EmptyState } from "@/components/EmptyState";
import { FilterBar } from "@/components/FilterBar";

export function SectionPage({ eyebrow, title, description, icon, emptyTitle, emptyDescription, action, href, filters }: { eyebrow: string; title: string; description: string; icon: string; emptyTitle: string; emptyDescription: string; action?: string; href?: string; filters?: string[] }) {
  return <div className="section-page"><div className="page-heading"><div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p className="heading-copy">{description}</p></div>{action && <a className="button button-primary" href={href}>{action} <span>→</span></a>}</div>{filters && <FilterBar filters={filters} />}<section className="panel section-panel"><EmptyState icon={icon} title={emptyTitle} description={emptyDescription} /></section></div>;
}
