import { Suspense } from "react";

import { NameDetailView } from "@/components/authors/NameDetailView";
import { LoadingState } from "@/components/LoadingState";

export default async function CategoryPage({ params }: { params: Promise<{ categoryId: string }> }) {
  const { categoryId } = await params;
  return <Suspense fallback={<LoadingState />}><NameDetailView kind="category" id={categoryId} /></Suspense>;
}
