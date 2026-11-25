"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { queryKeys } from "@/lib/query-keys";
import { userService } from "@/services/user-service";
import type { ChangePasswordRequest, UpdateProfileRequest, User } from "@/types/domain";

/** GET /api/users/me: the user with profile and stats. */
export function useProfile() {
  return useQuery({ queryKey: queryKeys.profile, queryFn: userService.getMe });
}

/** Saves the profile and updates the signed-in user everywhere (the header shows the display name). */
export function useUpdateProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateProfileRequest) => userService.updateMe(input),
    onSuccess: (user) => {
      client.setQueryData(queryKeys.profile, user);
      client.setQueryData<User | null>(queryKeys.me, (me) => (me ? { ...me, displayName: user.displayName, avatarUrl: user.avatarUrl } : me));
      // Review cards show the reviewer's display name.
      void client.invalidateQueries({ queryKey: queryKeys.books.all });
    },
  });
}

export function useChangePassword() {
  return useMutation({ mutationFn: (input: ChangePasswordRequest) => userService.changePassword(input) });
}

/**
 * Deletes the account. The caller leaves with a full page load, which also
 * drops every cached query; clearing the cache here first would make the
 * workspace guard redirect to /login before that.
 */
export function useDeleteAccount() {
  return useMutation({ mutationFn: userService.deleteMe });
}
