import { z } from "zod";

// Client-side checks that mirror the API's rules (docs/api/openapi.yaml
// RegisterRequest, backend/internal/services/auth.go); the API checks again.

const utf8Bytes = (s: string) => new TextEncoder().encode(s).length;

export const passwordRule = z
  .string()
  .min(8, "Use at least 8 characters.")
  .refine((s) => utf8Bytes(s) <= 72, "Use at most 72 bytes (fewer characters for emoji or accents).")
  .refine((s) => !/[\u0000-\u001f\u007f]/.test(s), "The password contains characters that are not allowed.");

const email = z.string().trim().min(1, "Email is required.").max(254, "Email is too long.").email("Enter a valid email address.");

export const loginSchema = z.object({
  email,
  password: z.string().min(1, "Password is required."),
});

export const registerSchema = z
  .object({
    displayName: z.string().trim().min(1, "Display name is required.").max(80, "Use at most 80 characters."),
    email,
    password: passwordRule,
    confirmPassword: z.string(),
  })
  .refine((v) => v.password === v.confirmPassword, { path: ["confirmPassword"], message: "Passwords do not match." });

export type LoginValues = z.infer<typeof loginSchema>;
export type RegisterValues = z.infer<typeof registerSchema>;

/**
 * The page to open after signing in. Only same-site paths are allowed, so a
 * crafted ?returnTo=https://evil.example cannot send the user elsewhere.
 */
export function safeReturnTo(value: string | null | undefined, fallback = "/dashboard"): string {
  if (!value || !value.startsWith("/") || value.startsWith("//") || value.startsWith("/\\")) return fallback;
  if (value === "/login" || value.startsWith("/login?") || value === "/register" || value.startsWith("/register?")) return fallback;
  return value;
}
