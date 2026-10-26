import Link from "next/link";

export function EmptyState({ icon = "□", title, description, action, href = "/catalog" }: { icon?: string; title: string; description: string; action?: string; href?: string }) {
  return <div className="empty-state"><span className="empty-state-icon">{icon}</span><h3>{title}</h3><p>{description}</p>{action && <Link className="button button-primary" href={href}>{action} <span>→</span></Link>}</div>;
}
