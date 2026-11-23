"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { useMe } from "@/hooks/use-auth";
import { errorMessage } from "@/lib/errors";

/** Shows the workspace only to a signed-in user (session = bm_session cookie, checked with GET /api/auth/me). */
export function AuthBoundary({ children }: { children: ReactNode }) {
  const router = useRouter();
  const me = useMe();
  const signedOut = me.isSuccess && me.data === null;

  useEffect(() => {
    if (!signedOut) return;
    const here = window.location.pathname + window.location.search;
    router.replace(`/login?returnTo=${encodeURIComponent(here)}`);
  }, [signedOut, router]);

  if (me.isPending) return <div className="auth-loading" aria-busy="true">Loading your reading space...</div>;
  if (me.isError) {
    return (
      <main className="auth-required">
        <div className="auth-required-card auth-error-card" role="alert">
          <span className="brand-mark">▥</span>
          <p className="eyebrow">Book Metaverse</p>
          <h1>We could not reach your library.</h1>
          <p>{errorMessage(me.error)}</p>
          <button className="button button-primary" type="button" onClick={() => void me.refetch()}>Try again</button>
        </div>
      </main>
    );
  }
  if (signedOut) {
    return (
      <main className="auth-required">
        <div className="auth-required-card">
          <span className="brand-mark">▥</span>
          <p className="eyebrow">Book Metaverse</p>
          <h1>Your reading space is private.</h1>
          <p>Sign in to access your shelves, progress, reports, ratings, and reviews.</p>
          <Link className="button button-primary" href="/login">Sign in to continue →</Link>
        </div>
      </main>
    );
  }
  return children;
}
