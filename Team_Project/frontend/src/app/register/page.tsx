import { Suspense } from "react";

import { AuthLayout } from "@/components/auth/AuthLayout";
import { RegisterForm } from "@/components/auth/RegisterForm";

export default function RegisterPage() {
  return (
    <AuthLayout eyebrow="Join Book Metaverse" title="Start your reading universe." copy="Create an account to keep shelves, track progress, and rate and review books. Four shelves are ready the moment you join.">
      <Suspense fallback={null}><RegisterForm /></Suspense>
    </AuthLayout>
  );
}
