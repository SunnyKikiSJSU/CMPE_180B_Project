"use client";

import Link from "next/link";
import { type KeyboardEvent, useEffect, useRef, useState } from "react";

import { useMe } from "@/hooks/use-auth";

export function PublicNav() {
  const [open, setOpen] = useState(false);
  const signedIn = Boolean(useMe().data);
  const drawer = useRef<HTMLDialogElement>(null);
  const menuButton = useRef<HTMLButtonElement>(null);

  // A <dialog> opened with showModal() moves focus inside, keeps Tab inside,
  // hides the page behind it from screen readers, and closes on Esc.
  useEffect(() => {
    const dialog = drawer.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  const close = () => setOpen(false);

  // Handle Esc ourselves too, in case something on the page blocks the browser's
  // built-in Esc, and wrap Tab so focus cycles inside the drawer instead of
  // leaving for the browser's address bar.
  function onDrawerKeyDown(event: KeyboardEvent<HTMLDialogElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      close();
      return;
    }
    if (event.key !== "Tab") return;
    const items = drawer.current?.querySelectorAll<HTMLElement>("a[href], button:not([disabled])");
    if (!items || items.length === 0) return;
    const first = items[0];
    const last = items[items.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  return (
    <>
      <nav className="public-nav">
        <button ref={menuButton} className="public-menu-button" onClick={() => setOpen(true)} aria-label="Open navigation" aria-haspopup="dialog" aria-expanded={open} aria-controls="public-drawer">☰</button>
        <Link className="public-brand" href="/">Book Metaverse<span>.</span></Link>
        <div className="public-nav-links"><Link href="/explore">Discover</Link></div>
        {signedIn ? <Link className="public-login" href="/dashboard">My library →</Link> : <Link className="public-login" href="/login">Sign in →</Link>}
      </nav>
      {/* Esc, the × button, a link, or a click on the dimmed area closes it; focus then returns to ☰. */}
      <dialog ref={drawer} id="public-drawer" className="drawer-backdrop" aria-label="Site navigation" onClose={() => { close(); menuButton.current?.focus(); }} onKeyDown={onDrawerKeyDown} onClick={close}>
        <aside className="public-drawer" onClick={(event) => event.stopPropagation()}>
          <button className="drawer-close" onClick={close} aria-label="Close navigation">×</button>
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
        </aside>
      </dialog>
    </>
  );
}
