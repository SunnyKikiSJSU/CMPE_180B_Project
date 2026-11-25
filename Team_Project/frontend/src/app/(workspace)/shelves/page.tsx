"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { ErrorState } from "@/components/ErrorState";
import { EmptyState } from "@/components/EmptyState";
import { LoadingState } from "@/components/LoadingState";
import { useCreateShelf, useDeleteShelf, useMoveShelfBook, useRemoveShelfBook, useRenameShelf, useShelfBooks, useShelves } from "@/hooks/use-reading";
import { errorMessage } from "@/lib/errors";
import type { Shelf } from "@/types/domain";

const protectedTypes = new Set(["want_to_read", "currently_reading", "finished", "paused"]);

function shelfKind(shelf: Shelf): string {
  return protectedTypes.has(shelf.type) ? "Status shelf" : "Custom shelf";
}

export default function ShelvesPage() {
  const shelves = useShelves();
  const [selectedId, setSelectedId] = useState<string>();
  const [newName, setNewName] = useState("");
  const [editingId, setEditingId] = useState<string>();
  const [editingName, setEditingName] = useState("");
  const activeId = selectedId && shelves.data?.some((shelf) => shelf.id === selectedId) ? selectedId : shelves.data?.[0]?.id;
  const selected = shelves.data?.find((shelf) => shelf.id === activeId);
  const books = useShelfBooks(activeId);
  const create = useCreateShelf();
  const rename = useRenameShelf();
  const remove = useDeleteShelf();
  const move = useMoveShelfBook();
  const removeBook = useRemoveShelfBook();

  function submitCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = newName.trim();
    if (!name) return;
    create.mutate(name, { onSuccess: (shelf) => { setNewName(""); setSelectedId(shelf.id); } });
  }

  function submitRename(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editingId || !editingName.trim()) return;
    rename.mutate({ shelfId: editingId, name: editingName.trim() }, { onSuccess: () => setEditingId(undefined) });
  }

  if (shelves.isPending) return <LoadingState message="Loading your shelves…" cards={4} />;
  if (shelves.error) return <ErrorState error={shelves.error} onRetry={() => void shelves.refetch()} />;

  return <div className="section-page f5-page">
    <div className="page-heading"><div><p className="eyebrow">Organize</p><h1>My shelves</h1><p className="heading-copy">Keep status shelves and personal collections in one place.</p></div></div>

    <section className="panel f5-create-panel"><form className="inline-form" onSubmit={submitCreate}><label htmlFor="new-shelf">Create a custom shelf</label><input id="new-shelf" value={newName} onChange={(event) => setNewName(event.target.value)} maxLength={80} placeholder="Shelf name" /><button className="button button-primary" type="submit" disabled={!newName.trim() || create.isPending}>{create.isPending ? "Creating…" : "Create shelf"}</button></form>{create.error && <p className="field-error" role="alert">{errorMessage(create.error)}</p>}</section>

    <div className="shelf-layout">
      <section className="panel shelf-list-panel"><div className="panel-heading"><div><p className="eyebrow">Your collections</p><h2>{shelves.data?.length ?? 0} shelves</h2></div></div>{!shelves.data?.length ? <EmptyState icon="▤" title="No shelves yet" description="Create a custom shelf to organize your books." /> : <div className="shelf-list">{shelves.data.map((shelf) => <button className={`shelf-list-item${shelf.id === activeId ? " is-selected" : ""}`} key={shelf.id} type="button" onClick={() => setSelectedId(shelf.id)}><span><strong>{shelf.name}</strong><small>{shelfKind(shelf)}</small></span><b>{shelf.count}</b></button>)}</div>}</section>

      <section className="panel shelf-detail-panel">{selected ? <><div className="panel-heading"><div><p className="eyebrow">{shelfKind(selected)}</p><h2>{selected.name}</h2><p className="panel-subtitle">{selected.count} {selected.count === 1 ? "book" : "books"}</p></div>{!protectedTypes.has(selected.type) && <div className="shelf-actions"><button className="text-button" type="button" onClick={() => { setEditingId(selected.id); setEditingName(selected.name); }}>Rename</button><button className="text-button danger" type="button" disabled={remove.isPending} onClick={() => { if (window.confirm(`Delete shelf “${selected.name}”? Books will remain in your library.`)) remove.mutate(selected.id); }}>Delete</button></div>}</div>{editingId === selected.id && <form className="inline-form shelf-edit-form" onSubmit={submitRename}><label htmlFor="rename-shelf">New name</label><input id="rename-shelf" value={editingName} onChange={(event) => setEditingName(event.target.value)} maxLength={80} autoFocus /><button className="button button-primary" type="submit" disabled={!editingName.trim() || rename.isPending}>{rename.isPending ? "Saving…" : "Save"}</button><button className="text-button" type="button" onClick={() => setEditingId(undefined)}>Cancel</button></form>}{rename.error && <p className="field-error" role="alert">{errorMessage(rename.error)}</p>}{remove.error && <p className="field-error" role="alert">{errorMessage(remove.error)}</p>}{books.isPending ? <LoadingState message="Loading shelf books…" cards={3} /> : books.error ? <ErrorState error={books.error} onRetry={() => void books.refetch()} /> : !books.data?.items.length ? <EmptyState icon="▤" title="This shelf is empty" description="Add books from the catalog or from a book detail page." action="Browse catalog" href="/catalog" /> : <div className="shelf-book-list">{books.data.items.map((book) => <article className="shelf-book-row" key={book.id}><div><Link href={`/books/${book.id}`}><strong>{book.title}</strong></Link><small>{book.author}</small></div><div className="shelf-book-actions"><label><span className="sr-only">Move {book.title}</span><select value="" disabled={move.isPending} onChange={(event) => { if (event.target.value) move.mutate({ shelfId: selected.id, bookId: book.id, targetShelfId: event.target.value }); }}><option value="">Move to…</option>{shelves.data.filter((target) => target.id !== selected.id).map((target) => <option value={target.id} key={target.id}>{target.name}</option>)}</select></label><button className="text-button danger" type="button" disabled={removeBook.isPending} onClick={() => removeBook.mutate({ shelfId: selected.id, bookId: book.id })}>Remove</button></div></article>)}</div>}{(move.error || removeBook.error) && <p className="field-error" role="alert">{errorMessage(move.error ?? removeBook.error)}</p>}</> : <EmptyState icon="▤" title="Select a shelf" description="Choose a shelf to view its books." />}</section>
    </div>
  </div>;
}
