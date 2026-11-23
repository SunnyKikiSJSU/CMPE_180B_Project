import { describe, expect, it } from "vitest";

import { loginSchema, registerSchema, safeReturnTo } from "@/lib/auth-schemas";

const base = { displayName: "Reader", email: "reader@example.com", password: "reading-pass", confirmPassword: "reading-pass" };

describe("registerSchema", () => {
  it("accepts a valid form and trims names", () => {
    const r = registerSchema.safeParse({ ...base, displayName: "  Reader  ", email: " reader@example.com " });
    expect(r.success && r.data).toMatchObject({ displayName: "Reader", email: "reader@example.com" });
  });

  it("applies the API's password rules (8 characters to 72 bytes, no control characters)", () => {
    const issue = (password: string) => {
      const r = registerSchema.safeParse({ ...base, password, confirmPassword: password });
      return r.success ? null : r.error.issues[0].path.join(".");
    };
    expect(issue("short")).toBe("password");
    expect(issue("a".repeat(72))).toBeNull();
    expect(issue("a".repeat(73))).toBe("password");
    expect(issue("é".repeat(37))).toBe("password"); // 74 bytes, 37 characters
    expect(issue("tab\there-ok")).toBe("password");
  });

  it("requires matching passwords and a display name", () => {
    const r = registerSchema.safeParse({ ...base, confirmPassword: "different-pass", displayName: " " });
    expect(r.success).toBe(false);
    const paths = r.success ? [] : r.error.issues.map((i) => i.path.join("."));
    expect(paths).toEqual(expect.arrayContaining(["displayName", "confirmPassword"]));
  });
});

describe("loginSchema", () => {
  it("needs an email and a password", () => {
    expect(loginSchema.safeParse({ email: "reader@example.com", password: "x" }).success).toBe(true);
    expect(loginSchema.safeParse({ email: "not-an-email", password: "x" }).success).toBe(false);
    expect(loginSchema.safeParse({ email: "reader@example.com", password: "" }).success).toBe(false);
  });
});

describe("safeReturnTo", () => {
  it("keeps same-site paths and rejects everything else", () => {
    expect(safeReturnTo("/shelves?tab=reading")).toBe("/shelves?tab=reading");
    for (const bad of [null, "", "https://evil.example", "//evil.example", "/\\evil.example", "javascript:alert(1)", "/login", "/login?returnTo=/x", "/register"]) {
      expect(safeReturnTo(bad)).toBe("/dashboard");
    }
  });
});
