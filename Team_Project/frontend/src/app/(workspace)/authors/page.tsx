import { Suspense } from "react";

import { NamesView } from "@/components/authors/NamesView";
import { LoadingState } from "@/components/LoadingState";

export default function AuthorsPage() {
  return <Suspense fallback={<LoadingState />}><NamesView /></Suspense>;
}
