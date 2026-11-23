"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { useEnsureLocalBook } from "@/hooks/use-books";
import { errorMessage } from "@/lib/errors";
import type { ExternalBook } from "@/types/domain";

/**
 * The call to action on an Open Library result for a signed-in reader: open
 * the local copy, or import it first (ensureLocalBook) and then open it.
 */
export function AddToLibraryButton({ book }: { book: ExternalBook }) {
  const router = useRouter();
  const ensure = useEnsureLocalBook();
  if (book.bookId) return <Link className="button button-secondary" href={`/books/${book.bookId}`}>Open in library →</Link>;
  const add = () => ensure.mutate(book, { onSuccess: (bookId) => router.push(`/books/${bookId}`) });
  return (
    <>
      <button className="button button-primary" type="button" onClick={add} disabled={ensure.isPending} aria-describedby={ensure.isError ? `import-error-${book.externalId}` : undefined}>
        {ensure.isPending ? "Adding…" : "Add to library"}
      </button>
      {ensure.isError && <p className="field-error" id={`import-error-${book.externalId}`} role="alert">{errorMessage(ensure.error)}</p>}
    </>
  );
}
