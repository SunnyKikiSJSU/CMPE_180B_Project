import { Suspense } from "react";

import { AuthLayout } from "@/components/auth/AuthLayout";
import { LoginForm } from "@/components/auth/LoginForm";

export default function LoginPage() {
  return (
    <AuthLayout eyebrow="Welcome back" title="Return to your reading universe." copy="Sign in to see your shelves, progress, reviews, and personal reading reports.">
      {/* LoginForm reads ?returnTo, which needs a Suspense boundary in the App Router. */}
      <Suspense fallback={null}><LoginForm /></Suspense>
    </AuthLayout>
  );
}
