"use client";

import Link from "next/link";
import { useEffect, useState, type ReactNode } from "react";

export function AuthBoundary({ children }: { children: ReactNode }) {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null);

  useEffect(() => setAuthenticated(window.localStorage.getItem("book-metaverse-session") === "demo-session"), []);

  if (authenticated === null) return <div className="auth-loading">Loading your reading space...</div>;
  if (!authenticated) return <main className="auth-required"><div className="auth-required-card"><span className="brand-mark">▥</span><p className="eyebrow">Book Metaverse</p><h1>Your reading space is private.</h1><p>Sign in to access your shelves, progress, reports, ratings, and reviews.</p><Link className="button button-primary" href="/login">Sign in to continue →</Link></div></main>;
  return children;
}
