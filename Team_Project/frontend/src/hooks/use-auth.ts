"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { queryKeys } from "@/lib/query-keys";
import { authService } from "@/services/auth-service";
import type { LoginRequest, RegisterRequest } from "@/types/domain";

/** The signed-in user; data is null when signed out. */
export function useMe() {
  return useQuery({ queryKey: queryKeys.me, queryFn: authService.me, staleTime: 5 * 60_000 });
}

/** Login and register both return the user and set the session cookie. */
export function useLogin() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: LoginRequest) => authService.login(input),
    onSuccess: (res) => client.setQueryData(queryKeys.me, res.user),
  });
}

export function useRegister() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: RegisterRequest) => authService.register(input),
    onSuccess: (res) => client.setQueryData(queryKeys.me, res.user),
  });
}

/** Clears the cookie, then every cached query, so no data of the previous user remains. */
export function useLogout() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: authService.logout,
    onSuccess: () => {
      client.clear();
      client.setQueryData(queryKeys.me, null);
    },
  });
}
