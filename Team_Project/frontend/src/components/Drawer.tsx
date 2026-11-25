"use client";

import { type KeyboardEvent, type ReactNode, type RefObject, useEffect, useRef } from "react";

/**
 * A slide-in navigation panel (the ☰ menu) built on <dialog>. showModal() moves
 * focus inside, hides the page behind it from screen readers, and closes on Esc.
 * Esc is also handled here in case something on the page blocks the browser's
 * own, and Tab wraps so focus cycles inside instead of leaving for the address
 * bar. When it closes, focus returns to the button that opened it.
 */
export function Drawer({ id, label, open, onClose, returnFocusTo, children }: { id: string; label: string; open: boolean; onClose: () => void; returnFocusTo: RefObject<HTMLElement | null>; children: ReactNode }) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  function onKeyDown(event: KeyboardEvent<HTMLDialogElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
      return;
    }
    if (event.key !== "Tab") return;
    const items = dialogRef.current?.querySelectorAll<HTMLElement>("a[href], button:not([disabled])");
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
    <dialog ref={dialogRef} id={id} className="drawer-backdrop" aria-label={label} onClose={() => { onClose(); returnFocusTo.current?.focus(); }} onKeyDown={onKeyDown} onClick={onClose}>
      <aside className="public-drawer" onClick={(event) => event.stopPropagation()}>
        <button className="drawer-close" onClick={onClose} aria-label="Close navigation">×</button>
        {children}
      </aside>
    </dialog>
  );
}
