"use client";

import { QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { isApiError } from "@/lib/errors";
import { queryKeys } from "@/lib/query-keys";

// Query defaults from FRONTEND_PLAN.md §3.3.
function makeQueryClient(): QueryClient {
  const client: QueryClient = new QueryClient({
    // A query that fails with 401 means the session ended: re-check who is
    // signed in, and AuthBoundary sends the user to /login. Mutations (login,
    // a wrong current password) are not covered, so their 401 stays on the form.
    queryCache: new QueryCache({
      onError: (error, query) => {
        if (isApiError(error) && error.status === 401 && query.queryKey[0] !== queryKeys.me[0]) {
          void client.invalidateQueries({ queryKey: queryKeys.me });
        }
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        // Never retry a 4xx; retry once for network errors and 5xx.
        retry: (failureCount, error) => {
          if (isApiError(error) && error.status >= 400 && error.status < 500) return false;
          return failureCount < 1;
        },
      },
      mutations: { retry: false },
    },
  });
  return client;
}

export function QueryProvider({ children }: { children: ReactNode }) {
  const [client] = useState(makeQueryClient);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
