"use client";

import Link from "next/link";
import { useEffect } from "react";

/**
 * Last-resort boundary for an unexpected rendering error (API errors are shown
 * by each page). Shows a fixed message, never the error text, and lets the
 * reader retry without a full reload.
 */
export default function RouteError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);
  return (
    <main className="auth-required">
      <div className="auth-required-card auth-error-card" role="alert">
        <span className="brand-mark" aria-hidden="true">▥</span>
        <p className="eyebrow">Something went wrong</p>
        <h1>This page hit a snag.</h1>
        <p>Try again. If it keeps happening, reload the page or come back later.</p>
        {error.digest && <p className="muted-note">Error reference {error.digest}</p>}
        <div className="card-actions">
          <button className="button button-primary" type="button" onClick={reset}>Try again</button>
          <Link className="button button-secondary" href="/dashboard">Dashboard</Link>
        </div>
      </div>
    </main>
  );
}
