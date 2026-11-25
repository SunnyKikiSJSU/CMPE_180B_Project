import { apiClient } from "@/lib/api-client";
import type { ChangePasswordRequest, UpdateProfileRequest, User } from "@/types/domain";

export const userService = {
  /** The signed-in user with profile and stats (GET /api/auth/me has neither). */
  getMe: () => apiClient<User>("/api/users/me"),
  updateMe: (input: UpdateProfileRequest) => apiClient<User>("/api/users/me", { method: "PATCH", body: input }),
  /** 204. A wrong current password is 400 on the currentPassword field, never 401. */
  changePassword: (input: ChangePasswordRequest) => apiClient<void>("/api/users/me/password", { method: "PUT", body: input }),
  /** 204; deletes everything the account owns and clears the session cookie. */
  deleteMe: () => apiClient<void>("/api/users/me", { method: "DELETE", body: { confirm: true } }),
};
