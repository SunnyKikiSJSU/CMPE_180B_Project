import type { Shelf } from "@/types/domain";

const icons = ["☕", "♡", "✦"];
const colors = ["bg-[#eadcc6]", "bg-[#ead0c9]", "bg-[#d8e0d1]"];

export function ShelvesSection({ shelves }: { shelves: Shelf[] }) {
  return <section id="shelves" className="mx-auto max-w-7xl px-6 py-20 lg:px-10"><div className="flex items-end justify-between"><div><p className="mb-3 text-xs font-bold uppercase tracking-[0.2em] text-[#b86f52]">Make it yours</p><h2 className="font-serif text-4xl tracking-[-0.04em] sm:text-5xl">Your shelves, your story.</h2></div><button className="hidden text-sm font-bold text-[#b86f52] sm:block">Browse all shelves ↗</button></div><div className="mt-10 grid gap-5 md:grid-cols-3">{shelves.map((shelf, index) => <article key={shelf.id} className={`rounded-3xl ${colors[index % colors.length]} p-7 transition hover:-translate-y-1`}><div className="flex items-start justify-between"><span className="grid size-12 place-items-center rounded-2xl bg-white/50 text-xl">{icons[index % icons.length]}</span><span className="text-2xl text-[#76685d]">↗</span></div><h3 className="mt-14 font-serif text-3xl tracking-[-0.03em]">{shelf.name}</h3><p className="mt-2 text-sm text-[#76685d]">{shelf.count} books</p></article>)}</div></section>;
}
