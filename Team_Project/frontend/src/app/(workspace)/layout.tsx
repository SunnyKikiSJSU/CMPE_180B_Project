import type { ReactNode } from "react";
import { AppShell } from "@/components/AppShell";
import { AuthBoundary } from "@/components/AuthBoundary";

export default function WorkspaceLayout({ children }: { children: ReactNode }) { return <AuthBoundary><AppShell>{children}</AppShell></AuthBoundary>; }
