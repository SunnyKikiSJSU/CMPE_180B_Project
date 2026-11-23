"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";

import { useMe } from "@/hooks/use-auth";
import { safeReturnTo } from "@/lib/auth-schemas";

/** Where to go after signing in, and an automatic redirect if already signed in. */
export function useAuthRedirect() {
  const router = useRouter();
  const returnTo = safeReturnTo(useSearchParams().get("returnTo"));
  const me = useMe();
  useEffect(() => {
    if (me.data) router.replace(returnTo);
  }, [me.data, returnTo, router]);
  return { returnTo, goNext: () => router.replace(returnTo) };
}
