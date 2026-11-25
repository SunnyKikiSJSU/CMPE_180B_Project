import { z } from "zod";

import { passwordRule } from "@/lib/auth-schemas";
import type { UpdateProfileRequest, User } from "@/types/domain";

// Mirrors backend/internal/services/validate.go and users.go: lengths are in
// characters after trimming; control characters are refused (the bio may have
// line breaks and tabs).
const chars = (s: string) => Array.from(s).length;
const controls = /[\u0000-\u001f\u007f]/;
const controlsExceptLineBreaks = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/;
const notAllowed = "Contains characters that are not allowed.";

function isWebAddress(value: string): boolean {
  if (new TextEncoder().encode(value).length > 2048) return false;
  try {
    const url = new URL(value);
    return (url.protocol === "https:" || url.protocol === "http:") && url.hostname !== "" && !url.username && !url.password;
  } catch {
    return false;
  }
}

export const profileSchema = z.object({
  displayName: z.string().trim().min(1, "Display name is required.").refine((s) => chars(s) <= 80, "Use at most 80 characters.").refine((s) => !controls.test(s), notAllowed),
  /** Empty means "keep the current avatar": the API cannot clear it. */
  avatarUrl: z.string().trim().refine((s) => s === "" || isWebAddress(s), "Enter an http or https address."),
  bio: z.string().trim().refine((s) => chars(s) <= 2000, "Use at most 2000 characters.").refine((s) => !controlsExceptLineBreaks.test(s), notAllowed),
  favoriteGenre: z.string().trim().refine((s) => chars(s) <= 80, "Use at most 80 characters.").refine((s) => !controls.test(s), notAllowed),
});

export type ProfileValues = z.input<typeof profileSchema>;
export type ProfileOutput = z.output<typeof profileSchema>;

export function profileDefaults(user: User): ProfileValues {
  return { displayName: user.displayName, avatarUrl: user.avatarUrl ?? "", bio: user.profile?.bio ?? "", favoriteGenre: user.profile?.favoriteGenre ?? "" };
}

/**
 * The PATCH body: only fields that changed. An empty avatar address is left
 * out because the API has no way to remove an avatar; a blank bio or genre is
 * sent as "" and clears it.
 */
export function profileChanges(values: ProfileOutput, user: User): UpdateProfileRequest {
  const current = profileDefaults(user);
  const changes: UpdateProfileRequest = {};
  if (values.displayName !== current.displayName) changes.displayName = values.displayName;
  if (values.avatarUrl !== "" && values.avatarUrl !== current.avatarUrl) changes.avatarUrl = values.avatarUrl;
  if (values.bio !== current.bio) changes.bio = values.bio;
  if (values.favoriteGenre !== current.favoriteGenre) changes.favoriteGenre = values.favoriteGenre;
  return changes;
}

export const passwordSchema = z
  .object({
    currentPassword: z.string().min(1, "Enter your current password."),
    newPassword: passwordRule,
    confirmPassword: z.string(),
  })
  .refine((v) => v.newPassword === v.confirmPassword, { path: ["confirmPassword"], message: "Passwords do not match." });

export type PasswordValues = z.infer<typeof passwordSchema>;
