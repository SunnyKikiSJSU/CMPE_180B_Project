import { describe, expect, it } from "vitest";

import { passwordSchema, profileChanges, profileDefaults, profileSchema } from "@/lib/account-schemas";
import type { User } from "@/types/domain";

const user: User = {
  id: "u1",
  email: "reader1@example.com",
  displayName: "Reader One",
  avatarUrl: "https://example.com/a.png",
  createdAt: "2026-01-01T00:00:00Z",
  profile: { bio: null, favoriteGenre: "Mystery" },
};

const valid = { displayName: "Reader", avatarUrl: "", bio: "", favoriteGenre: "" };

describe("profileSchema", () => {
  it("trims and accepts empty optional fields", () => {
    expect(profileSchema.parse({ ...valid, displayName: "  Reader  ", bio: " Line one\nLine two " })).toEqual({ ...valid, bio: "Line one\nLine two" });
  });

  it("mirrors the API's limits", () => {
    expect(profileSchema.safeParse({ ...valid, displayName: "   " }).success).toBe(false);
    expect(profileSchema.safeParse({ ...valid, displayName: "é".repeat(80) }).success).toBe(true);
    expect(profileSchema.safeParse({ ...valid, displayName: "é".repeat(81) }).success).toBe(false);
    expect(profileSchema.safeParse({ ...valid, bio: "x".repeat(2001) }).success).toBe(false);
    expect(profileSchema.safeParse({ ...valid, favoriteGenre: "Sci\nFi" }).success).toBe(false);
  });

  it("accepts only http(s) avatar addresses without credentials", () => {
    expect(profileSchema.safeParse({ ...valid, avatarUrl: "https://example.com/me.png" }).success).toBe(true);
    for (const bad of ["example.com/me.png", "javascript:alert(1)", "ftp://example.com/a", "https://user:pw@example.com/a"]) {
      expect(profileSchema.safeParse({ ...valid, avatarUrl: bad }).success).toBe(false);
    }
  });
});

describe("profileChanges", () => {
  it("sends only changed fields", () => {
    expect(profileChanges(profileSchema.parse(profileDefaults(user)), user)).toEqual({});
    expect(profileChanges({ displayName: "New", avatarUrl: "https://example.com/a.png", bio: "Hi", favoriteGenre: "Mystery" }, user)).toEqual({ displayName: "New", bio: "Hi" });
  });

  it("never sends an empty avatar (the API cannot remove one) but clears bio and genre", () => {
    expect(profileChanges({ displayName: "Reader One", avatarUrl: "", bio: "", favoriteGenre: "" }, user)).toEqual({ favoriteGenre: "" });
  });
});

describe("passwordSchema", () => {
  it("requires the current password and a matching new one", () => {
    expect(passwordSchema.safeParse({ currentPassword: "", newPassword: "Password456!", confirmPassword: "Password456!" }).success).toBe(false);
    expect(passwordSchema.safeParse({ currentPassword: "old", newPassword: "short", confirmPassword: "short" }).success).toBe(false);
    const mismatch = passwordSchema.safeParse({ currentPassword: "old", newPassword: "Password456!", confirmPassword: "Password457!" });
    expect(mismatch.success).toBe(false);
    expect(mismatch.error?.issues[0].path).toEqual(["confirmPassword"]);
    expect(passwordSchema.safeParse({ currentPassword: "old", newPassword: "Password456!", confirmPassword: "Password456!" }).success).toBe(true);
  });
});
