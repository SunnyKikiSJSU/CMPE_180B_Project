import { Suspense } from "react";

import { LoadingState } from "@/components/LoadingState";
import { MyReviewsView } from "@/components/reviews/MyReviewsView";

export default function ReviewsPage() {
  return <Suspense fallback={<LoadingState />}><MyReviewsView /></Suspense>;
}
