"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!email || !password) return;
    // Temporary frontend-only session until the authentication API is connected.
    window.localStorage.setItem("book-metaverse-session", "demo-session");
    router.push("/dashboard");
  }

  return <main className="login-page"><div className="login-panel"><Link className="public-brand" href="/">Book Metaverse<span>.</span></Link><p className="eyebrow">Welcome back</p><h1>Return to your reading universe.</h1><p className="login-copy">Sign in to see your shelves, progress, reviews, and personal reading reports.</p><form onSubmit={handleSubmit}><label>Email address<input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@example.com" required /></label><label>Password<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="••••••••" required /></label><button className="button button-primary" type="submit">Sign in →</button></form><p className="login-footnote">Authentication is currently a frontend demo. The Go API will replace this temporary session.</p></div><div className="login-art"><span>▥</span><h2>Every chapter has a place here.</h2><p>Build a library that feels like yours.</p></div></main>;
}
