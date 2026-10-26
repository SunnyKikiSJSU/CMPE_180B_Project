"use client";

import Link from "next/link";
import { useState } from "react";

export function PublicNav() {
  const [open, setOpen] = useState(false);
  return <><nav className="public-nav"><button className="public-menu-button" onClick={() => setOpen(true)} aria-label="Open navigation">☰</button><Link className="public-brand" href="/">Book Metaverse<span>.</span></Link><div className="public-nav-links"><Link href="/explore">Discover</Link></div><Link className="public-login" href="/login">Sign in →</Link></nav>{open && <div className="drawer-backdrop" onClick={() => setOpen(false)}><aside className="public-drawer" onClick={(event) => event.stopPropagation()}><button className="drawer-close" onClick={() => setOpen(false)} aria-label="Close navigation">×</button><p className="eyebrow">Book Metaverse</p><h2>Find your next chapter.</h2><Link href="/" onClick={() => setOpen(false)}>Home</Link><Link href="/explore" onClick={() => setOpen(false)}>Discover books</Link><Link href="/login">Sign in</Link><div className="drawer-rule" /><small>Your reading universe, thoughtfully organized.</small></aside></div>}</>;
}
