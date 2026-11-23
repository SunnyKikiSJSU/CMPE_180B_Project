import { Suspense } from "react";

import { CatalogView } from "@/components/catalog/CatalogView";
import { LoadingState } from "@/components/LoadingState";

export default function CatalogPage() {
  return <Suspense fallback={<LoadingState />}><CatalogView /></Suspense>;
}
