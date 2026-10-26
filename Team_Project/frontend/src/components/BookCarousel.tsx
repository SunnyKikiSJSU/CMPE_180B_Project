"use client";
/* Static editorial imagery is intentionally loaded as a plain img for now; the API will later provide optimized cover assets. */
/* eslint-disable @next/next/no-img-element */

import { useState } from "react";
import type { Book } from "@/types/domain";

const colors = ["bg-[#b86f52]", "bg-[#87977b]", "bg-[#c7a875]"];
const marks = ["✦", "❧", "♡"];

export function BookCarousel({ books }: { books: Book[] }) {
  const [active, setActive] = useState(0);
  const book = books[active];
  const move = (direction: number) => setActive((current) => (current + direction + books.length) % books.length);
  return <div className="relative mx-auto w-full max-w-[30rem] animate-float lg:mr-0"><div className="absolute -right-5 -top-5 z-10 grid size-24 rotate-6 place-items-center rounded-full bg-[#e2b95f] text-center text-xs font-bold uppercase leading-4 tracking-wider text-[#604b2d] shadow-lg">read<br />slowly<br />♡</div><div className="relative aspect-[0.92] overflow-hidden rounded-[2rem] bg-[#e6d7c4] p-5 shadow-[12px_18px_0_#e8ddd0] sm:p-8"><div className={`relative flex h-full flex-col justify-between overflow-hidden rounded-[1.35rem] ${colors[active % colors.length]} p-7 text-[#fff8ee] transition-colors duration-500 sm:p-10`}><img src={book.coverUrl} alt="" className="absolute inset-0 size-full object-cover opacity-35 mix-blend-multiply transition-all duration-700" /><div className="absolute inset-0 bg-gradient-to-t from-black/75 via-black/20 to-transparent" /><div className="absolute -right-20 top-16 size-64 rounded-full border border-white/20" /><div className="absolute -bottom-24 -left-16 size-72 rounded-full border border-white/20" /><div className="relative flex items-start justify-between"><span className="rounded-full border border-white/40 px-3 py-1 text-[10px] font-bold uppercase tracking-[0.2em]">Featured read</span><span className="text-3xl">{marks[active % marks.length]}</span></div><div className="relative"><p className="mb-3 text-xs font-bold uppercase tracking-[0.2em] opacity-80">{book.categories?.[0]?.name ?? "A good story"}</p><h2 className="max-w-xs font-serif text-5xl leading-[0.95] tracking-[-0.04em] sm:text-6xl">{book.title}</h2><p className="mt-5 text-sm font-medium opacity-85">{book.author}</p></div></div></div><div className="mt-7 flex items-center justify-between px-2"><div className="flex gap-2">{books.map((item, index) => <button key={item.id} aria-label={`Show ${item.title}`} onClick={() => setActive(index)} className={`h-1.5 rounded-full transition-all ${index === active ? "w-10 bg-[#b86f52]" : "w-5 bg-[#d7c9ba]"}`} />)}</div><div className="flex gap-2"><button onClick={() => move(-1)} aria-label="Previous featured book" className="grid size-10 place-items-center rounded-full border border-[#d8cbbd] text-lg hover:bg-white">←</button><button onClick={() => move(1)} aria-label="Next featured book" className="grid size-10 place-items-center rounded-full border border-[#d8cbbd] text-lg hover:bg-white">→</button></div></div></div>;
}
