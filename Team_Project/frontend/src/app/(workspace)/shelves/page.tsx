"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { ConfirmButton } from "@/components/ConfirmButton";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { LoadingState } from "@/components/LoadingState";
import { useCreateShelf, useDeleteShelf, useMoveShelfBook, useRemoveShelfBook, useRenameShelf, useShelfBooks, useShelves } from "@/hooks/use-reading";
import { errorMessage } from "@/lib/errors";
import type { Book, Shelf } from "@/types/domain";

const isStatusShelf = (shelf: Shelf) => shelf.type !== "custom";
const shelfKind = (shelf: Shelf) => (isStatusShelf(shelf) ? "Status shelf" : "Custom shelf");

export default function ShelvesPage() {
  const shelves = useShelves();
  const [selectedId, setSelectedId] = useState<string>();
  const [newName, setNewName] = useState("");
  const create = useCreateShelf();

  function submitCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = newName.trim();
    if (!name) return;
    create.mutate(name, { onSuccess: (shelf) => { setNewName(""); setSelectedId(shelf.id); } });
  }

  if (shelves.isPending) return <LoadingState message="Loading your shelves…" cards={4} />;
  if (shelves.isError) return <ErrorState error={shelves.error} onRetry={() => void shelves.refetch()} />;

  const list = shelves.data;
  const selected = list.find((shelf) => shelf.id === selectedId) ?? list[0];

  return (
    <div className="section-page f5-page">
      <div className="page-heading"><div><p className="eyebrow">Organize</p><h1>My shelves</h1><p className="heading-copy">Keep status shelves and personal collections in one place.</p></div></div>

      <section className="panel f5-create-panel">
        <form className="inline-form" onSubmit={submitCreate}>
          <label htmlFor="new-shelf">Create a custom shelf</label>
          <input id="new-shelf" value={newName} onChange={(event) => setNewName(event.target.value)} maxLength={80} placeholder="Shelf name" />
          <button className="button button-primary" type="submit" disabled={!newName.trim() || create.isPending}>{create.isPending ? "Creating…" : "Create shelf"}</button>
        </form>
        {create.error && <p className="field-error" role="alert">{errorMessage(create.error)}</p>}
      </section>

      <div className="shelf-layout">
        <section className="panel shelf-list-panel">
          <div className="panel-heading"><div><p className="eyebrow">Your collections</p><h2>{list.length} shelves</h2></div></div>
          {list.length === 0 ? <EmptyState icon="▤" title="No shelves yet" description="Create a custom shelf to organize your books." /> : (
            <div className="shelf-list">
              {list.map((shelf) => (
                <button className={`shelf-list-item${shelf.id === selected?.id ? " is-selected" : ""}`} aria-pressed={shelf.id === selected?.id} key={shelf.id} type="button" onClick={() => setSelectedId(shelf.id)}>
                  <span><strong>{shelf.name}</strong><small>{shelfKind(shelf)}</small></span><b>{shelf.count}</b>
                </button>
              ))}
            </div>
          )}
        </section>
        <section className="panel shelf-detail-panel">
          {/* key: switching shelves resets paging, the rename form, and old errors. */}
          {selected ? <ShelfDetail key={selected.id} shelf={selected} shelves={list} /> : <EmptyState icon="▤" title="Select a shelf" description="Choose a shelf to view its books." />}
        </section>
      </div>
    </div>
  );
}

function ShelfDetail({ shelf, shelves }: { shelf: Shelf; shelves: Shelf[] }) {
  const [page, setPage] = useState(1);
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(shelf.name);
  const books = useShelfBooks(shelf.id, page);
  const rename = useRenameShelf();
  const remove = useDeleteShelf();

  function submitRename(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim()) return;
    rename.mutate({ shelfId: shelf.id, name: name.trim() }, { onSuccess: () => setEditing(false) });
  }

  const last = books.data ? Math.max(1, Math.ceil(books.data.total / books.data.pageSize)) : 1;
  // Moving or removing the last book on a page leaves it empty; go back in range.
  if (books.data && !books.isPlaceholderData && page > last) setPage(last);
  return (
    <>
      <div className="panel-heading">
        <div><p className="eyebrow">{shelfKind(shelf)}</p><h2>{shelf.name}</h2><p className="panel-subtitle">{shelf.count} {shelf.count === 1 ? "book" : "books"}</p></div>
        {!isStatusShelf(shelf) && (
          <div className="shelf-actions">
            <button className="text-button" type="button" onClick={() => { setEditing(true); setName(shelf.name); }}>Rename</button>
            <ConfirmButton label="Delete" question={`Delete “${shelf.name}”? Its books stay in your library.`} confirmLabel="Delete" pending={remove.isPending} onConfirm={() => remove.mutate(shelf.id)} />
          </div>
        )}
      </div>
      {editing && (
        <form className="inline-form shelf-edit-form" onSubmit={submitRename}>
          <label htmlFor="rename-shelf">New name</label>
          <input id="rename-shelf" value={name} onChange={(event) => setName(event.target.value)} maxLength={80} autoFocus />
          <button className="button button-primary" type="submit" disabled={!name.trim() || rename.isPending}>{rename.isPending ? "Saving…" : "Save"}</button>
          <button className="text-button" type="button" onClick={() => { setEditing(false); rename.reset(); }}>Cancel</button>
        </form>
      )}
      {rename.error && <p className="field-error" role="alert">{errorMessage(rename.error)}</p>}
      {remove.error && <p className="field-error" role="alert">{errorMessage(remove.error)}</p>}
      {books.isPending ? <LoadingState message="Loading shelf books…" cards={3} />
        : books.isError ? <ErrorState error={books.error} onRetry={() => void books.refetch()} />
        : books.data.items.length === 0 ? <EmptyState icon="▤" title="This shelf is empty" description="Add books from the catalog or from a book detail page." action="Browse catalog" href="/catalog" />
        : (
          <div className={`shelf-book-list${books.isPlaceholderData ? " is-busy" : ""}`}>
            {books.data.items.map((book) => <ShelfBookRow key={book.id} book={book} shelf={shelf} shelves={shelves} />)}
          </div>
        )}
      {last > 1 && (
        <nav className="pagination" aria-label="Shelf pages">
          <button type="button" className="pagination-link" disabled={page <= 1} onClick={() => setPage(page - 1)}>← Previous</button>
          <span className="pagination-status">Page {page} of {last}</span>
          <button type="button" className="pagination-link" disabled={page >= last} onClick={() => setPage(page + 1)}>Next →</button>
        </nav>
      )}
    </>
  );
}

/**
 * Moving follows the API: a book has one status shelf, so choosing another
 * status shelf moves it; from a custom shelf any choice moves it; from a
 * status shelf a custom shelf is added alongside (its status is kept).
 */
function ShelfBookRow({ book, shelf, shelves }: { book: Book; shelf: Shelf; shelves: Shelf[] }) {
  const move = useMoveShelfBook();
  const remove = useRemoveShelfBook();
  const others = shelves.filter((target) => target.id !== shelf.id);
  const statusTargets = others.filter(isStatusShelf);
  const customTargets = others.filter((target) => !isStatusShelf(target));
  const fromStatus = isStatusShelf(shelf);
  const removeConfirm = fromStatus ? "Remove it and its reading progress?" : `Remove it from “${shelf.name}”?`;
  return (
    <article className="shelf-book-row">
      <div><Link href={`/books/${book.id}`}><strong>{book.title}</strong></Link><small>{book.author}</small></div>
      <div className="shelf-book-actions">
        <label>
          <span className="sr-only">Move or add {book.title}</span>
          <select value="" disabled={move.isPending} onChange={(event) => { if (event.target.value) move.mutate({ shelfId: shelf.id, bookId: book.id, targetShelfId: event.target.value }); }}>
            <option value="">{move.isPending ? "Saving…" : "Move to…"}</option>
            {statusTargets.length > 0 && <optgroup label="Reading status">{statusTargets.map((target) => <option value={target.id} key={target.id}>{target.name}</option>)}</optgroup>}
            {customTargets.length > 0 && <optgroup label={fromStatus ? "Also add to" : "Custom shelves"}>{customTargets.map((target) => <option value={target.id} key={target.id}>{target.name}</option>)}</optgroup>}
          </select>
        </label>
        <ConfirmButton label="Remove" question={removeConfirm} confirmLabel="Remove" pending={remove.isPending} onConfirm={() => remove.mutate({ shelfId: shelf.id, bookId: book.id })} />
      </div>
      {(move.error || remove.error) && <p className="field-error" role="alert">{errorMessage(move.error ?? remove.error)}</p>}
    </article>
  );
}
