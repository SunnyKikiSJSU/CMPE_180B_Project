import { afterEach, describe, expect, it, vi } from "vitest";

import { authService } from "@/services/auth-service";

const envelope = (status: number, code: string) =>
  new Response(JSON.stringify({ error: { code, message: code } }), { status, headers: { "Content-Type": "application/json" } });

afterEach(() => vi.unstubAllGlobals());

describe("authService", () => {
  it("me unwraps { user } and treats 401 as signed out", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ user: { id: "u1", displayName: "R" } }), { status: 200 })));
    await expect(authService.me()).resolves.toMatchObject({ id: "u1" });
    vi.stubGlobal("fetch", vi.fn(async () => envelope(401, "UNAUTHENTICATED")));
    await expect(authService.me()).resolves.toBeNull();
    vi.stubGlobal("fetch", vi.fn(async () => envelope(503, "DATABASE_UNAVAILABLE")));
    await expect(authService.me()).rejects.toMatchObject({ code: "DATABASE_UNAVAILABLE" });
  });

  it("logout succeeds when the session already expired", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => envelope(401, "UNAUTHENTICATED")));
    await expect(authService.logout()).resolves.toBeUndefined();
    vi.stubGlobal("fetch", vi.fn(async () => envelope(500, "INTERNAL")));
    await expect(authService.logout()).rejects.toMatchObject({ code: "INTERNAL" });
  });
});
