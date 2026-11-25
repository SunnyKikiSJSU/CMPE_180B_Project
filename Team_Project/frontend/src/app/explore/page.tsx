import { Suspense } from "react";

import { ExploreView } from "@/components/ExploreView";
import { LoadingState } from "@/components/LoadingState";
import { PublicNav } from "@/components/PublicNav";

/**
 * ExploreView reads the URL's search params, which are unknown when the page is
 * prerendered, so the static HTML is this fallback. Without one the page is
 * blank until JavaScript loads.
 */
function ExploreFallback() {
  return (
    <main className="public-explore">
      <PublicNav />
      <section className="explore-card">
        <LoadingState message="Loading the catalog…" cards={4} />
      </section>
    </main>
  );
}

export default function ExplorePage() {
  return <Suspense fallback={<ExploreFallback />}><ExploreView /></Suspense>;
}
