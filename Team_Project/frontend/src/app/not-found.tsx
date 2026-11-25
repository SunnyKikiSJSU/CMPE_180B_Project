import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "Page not found · Book Metaverse" };

/** Unknown URLs (and notFound()) get a branded page instead of the framework default. */
export default function NotFound() {
  return (
    <main className="auth-required">
      <div className="auth-required-card">
        <span className="brand-mark" aria-hidden="true">▥</span>
        <p className="eyebrow">Page not found</p>
        <h1>This chapter is missing.</h1>
        <p>The link may be old or mistyped. Try the catalog, or head back home.</p>
        <div className="card-actions">
          <Link className="button button-primary" href="/explore">Explore books →</Link>
          <Link className="button button-secondary" href="/">Home</Link>
        </div>
      </div>
    </main>
  );
}
