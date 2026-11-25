import { Suspense } from "react";

import { NameDetailView } from "@/components/authors/NameDetailView";
import { LoadingState } from "@/components/LoadingState";

export default async function AuthorPage({ params }: { params: Promise<{ authorId: string }> }) {
  const { authorId } = await params;
  return <Suspense fallback={<LoadingState />}><NameDetailView kind="author" id={authorId} /></Suspense>;
}
