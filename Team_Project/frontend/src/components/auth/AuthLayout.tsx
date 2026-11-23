import Link from "next/link";
import type { ReactNode } from "react";

/** Shared frame of /login and /register: form panel on the left, art on the right. */
export function AuthLayout({ eyebrow, title, copy, children }: { eyebrow: string; title: string; copy: string; children: ReactNode }) {
  return (
    <main className="login-page">
      <div className="login-panel">
        <Link className="public-brand" href="/">Book Metaverse<span>.</span></Link>
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p className="login-copy">{copy}</p>
        {children}
      </div>
      <div className="login-art"><span>▥</span><h2>Every chapter has a place here.</h2><p>Build a library that feels like yours.</p></div>
    </main>
  );
}

/** A form-level error with the request ID, so a report can be matched to the API log. */
export function FormAlert({ message, requestId }: { message: string; requestId?: string }) {
  return (
    <div className="form-alert" role="alert">
      {message}
      {requestId && <small>Reference: {requestId}</small>}
    </div>
  );
}
