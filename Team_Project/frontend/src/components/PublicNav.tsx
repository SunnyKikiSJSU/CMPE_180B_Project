"use client";

import Link from "next/link";
import { useState } from "react";

import { useMe } from "@/hooks/use-auth";

export function PublicNav() {
  const [open, setOpen] = useState(false);
  const signedIn = Boolean(useMe().data);
  return <><nav className="public-nav"><button className="public-menu-button" onClick={() => setOpen(true)} aria-label="Open navigation">☰</button><Link className="public-brand" href="/">Book Metaverse<span>.</span></Link><div className="public-nav-links"><Link href="/explore">Discover</Link></div>{signedIn ? <Link className="public-login" href="/dashboard">My library →</Link> : <Link className="public-login" href="/login">Sign in →</Link>}</nav>{open && <div className="drawer-backdrop" onClick={() => setOpen(false)}><aside className="public-drawer" onClick={(event) => event.stopPropagation()}><button className="drawer-close" onClick={() => setOpen(false)} aria-label="Close navigation">×</button><p className="eyebrow">Book Metaverse</p><h2>Find your next chapter.</h2><Link href="/" onClick={() => setOpen(false)}>Home</Link><Link href="/explore" onClick={() => setOpen(false)}>Discover books</Link>{signedIn ? <Link href="/dashboard">My library</Link> : <><Link href="/login">Sign in</Link><Link href="/register">Create an account</Link></>}<div className="drawer-rule" /><small>Your reading universe, thoughtfully organized.</small></aside></div>}</>;
}
