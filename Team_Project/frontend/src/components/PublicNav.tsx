"use client";

import Link from "next/link";
import { useRef, useState } from "react";

import { Drawer } from "@/components/Drawer";
import { useMe } from "@/hooks/use-auth";

export function PublicNav() {
  const [open, setOpen] = useState(false);
  const signedIn = Boolean(useMe().data);
  const menuButton = useRef<HTMLButtonElement>(null);
  const close = () => setOpen(false);

  return (
    <>
      <nav className="public-nav">
        <button ref={menuButton} className="public-menu-button" onClick={() => setOpen(true)} aria-label="Open navigation" aria-haspopup="dialog" aria-expanded={open} aria-controls="public-drawer">☰</button>
        <Link className="public-brand" href="/">Book Metaverse<span>.</span></Link>
        <div className="public-nav-links"><Link href="/explore">Discover</Link></div>
        {signedIn ? <Link className="public-login" href="/dashboard">My library →</Link> : <Link className="public-login" href="/login">Sign in →</Link>}
      </nav>
      <Drawer id="public-drawer" label="Site navigation" open={open} onClose={close} returnFocusTo={menuButton}>
        <p className="eyebrow">Book Metaverse</p>
        <h2>Find your next chapter.</h2>
        <Link href="/" onClick={close}>Home</Link>
        <Link href="/explore" onClick={close}>Discover books</Link>
        {signedIn ? (
          <Link href="/dashboard" onClick={close}>My library</Link>
        ) : (
          <>
            <Link href="/login" onClick={close}>Sign in</Link>
            <Link href="/register" onClick={close}>Create an account</Link>
          </>
        )}
        <div className="drawer-rule" />
        <small>Your reading universe, thoughtfully organized.</small>
      </Drawer>
    </>
  );
}
