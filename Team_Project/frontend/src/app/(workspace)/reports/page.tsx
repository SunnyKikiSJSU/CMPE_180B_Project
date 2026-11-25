import { Suspense } from "react";

import { LoadingState } from "@/components/LoadingState";
import { ReportsView } from "@/components/reports/ReportsView";

export default function ReportsPage() {
  return <Suspense fallback={<LoadingState />}><ReportsView /></Suspense>;
}
