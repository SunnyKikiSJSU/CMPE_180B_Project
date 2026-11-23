import { apiClient } from "@/lib/api-client";
import { isApiError } from "@/lib/errors";
import type { AuthResponse, LoginRequest, RegisterRequest, User } from "@/types/domain";

export const authService = {
  /** 201 with the bm_session cookie: registration also signs the user in. */
  register: (input: RegisterRequest) => apiClient<AuthResponse>("/api/auth/register", { method: "POST", body: input }),
  login: (input: LoginRequest) => apiClient<AuthResponse>("/api/auth/login", { method: "POST", body: input }),

  /** Clears the cookie. A session that already expired counts as signed out. */
  logout: async (): Promise<void> => {
    try {
      await apiClient<void>("/api/auth/logout", { method: "POST" });
    } catch (error) {
      if (!isApiError(error) || error.status !== 401) throw error;
    }
  },

  /** The signed-in user, or null when signed out (401 is not an error here). */
  me: async (): Promise<User | null> => {
    try {
      const res = await apiClient<{ user: User }>("/api/auth/me");
      return res.user;
    } catch (error) {
      if (isApiError(error) && error.status === 401) return null;
      throw error;
    }
  },
};
