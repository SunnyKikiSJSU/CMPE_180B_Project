"use client";
/* Covers are hotlinked from covers.openlibrary.org (never copied), so a plain img is used (FRONTEND_PLAN.md decision 14). */
/* eslint-disable @next/next/no-img-element */

import { useState } from "react";

/** A book cover, or a lettered placeholder when there is none or it fails to load. */
export function CoverImage({ src, title, size = "card" }: { src: string | null | undefined; title: string; size?: "card" | "detail" }) {
  const [failed, setFailed] = useState(false);
  const className = `cover cover-${size}`;
  if (!src || failed || !src.startsWith("https://")) {
    return <div className={`${className} cover-placeholder`} aria-hidden="true"><span>{title.trim().charAt(0).toUpperCase() || "?"}</span></div>;
  }
  return <img className={className} src={src} alt={`Cover of ${title}`} loading="lazy" referrerPolicy="no-referrer" onError={() => setFailed(true)} />;
}
