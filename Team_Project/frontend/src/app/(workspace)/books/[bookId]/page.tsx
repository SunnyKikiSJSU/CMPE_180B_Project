import { BookDetailView } from "@/components/books/BookDetailView";

export default async function BookPage({ params }: { params: Promise<{ bookId: string }> }) {
  const { bookId } = await params;
  return <BookDetailView bookId={bookId} />;
}
