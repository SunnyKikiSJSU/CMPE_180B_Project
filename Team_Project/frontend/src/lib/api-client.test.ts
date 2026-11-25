import { afterEach, describe, expect, it, vi } from "vitest";

import { apiClient, apiClientPage, apiClientWithHeaders, query } from "@/lib/api-client";
import { ApiError, errorMessage } from "@/lib/errors";

// Each call gets a fresh Response (a body can be read only once).
function mockFetch(response: Response | Error) {
  const fn = vi.fn(async () => {
    if (response instanceof Error) throw response;
    return response.clone();
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

function json(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json", ...headers } });
}

afterEach(() => vi.unstubAllGlobals());

describe("apiClient", () => {
  it("sends JSON only when there is a body, with the same-origin cookie", async () => {
    const fetch = mockFetch(json(200, { ok: true }));
    await apiClient("/api/books/1/ratings", { method: "POST", body: { value: 4 } });
    const [path, init] = fetch.mock.calls[0] as unknown as [string, RequestInit];
    expect(path).toBe("/api/books/1/ratings");
    expect(init.method).toBe("POST");
    expect(init.body).toBe('{"value":4}');
    expect(init.credentials).toBe("same-origin");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");

    await apiClient("/api/books");
    const [, getInit] = fetch.mock.calls[1] as unknown as [string, RequestInit];
    expect(getInit.method).toBe("GET");
    expect(getInit.body).toBeUndefined();
    expect((getInit.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
  });

  it("returns undefined for 204 and empty bodies", async () => {
    mockFetch(new Response(null, { status: 204 }));
    await expect(apiClient("/api/auth/logout", { method: "POST" })).resolves.toBeUndefined();
    mockFetch(new Response("", { status: 200 }));
    await expect(apiClient("/api/x")).resolves.toBeUndefined();
  });

  it("parses the error envelope", async () => {
    mockFetch(json(400, { error: { code: "VALIDATION_ERROR", message: "Enter a valid email address.", details: { field: "email" }, requestId: "req-1" } }));
    const error = await apiClient("/api/auth/register", { method: "POST", body: {} }).catch((e) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 400, code: "VALIDATION_ERROR", message: "Enter a valid email address.", requestId: "req-1", field: "email" });
  });

  it("falls back to INTERNAL with the X-Request-ID header when the body is not an envelope", async () => {
    mockFetch(new Response("<html>Bad Gateway</html>", { status: 502, headers: { "X-Request-ID": "req-2" } }));
    const error = await apiClient("/api/books").catch((e) => e);
    expect(error).toMatchObject({ status: 502, code: "INTERNAL", requestId: "req-2" });
  });

  it("reports a network failure as NETWORK_ERROR", async () => {
    mockFetch(new TypeError("fetch failed"));
    await expect(apiClient("/api/books")).rejects.toMatchObject({ status: 0, code: "NETWORK_ERROR" });
  });

  it("only calls relative /api paths", async () => {
    const fetch = mockFetch(json(200, {}));
    await expect(apiClient("https://evil.example/api/x")).rejects.toThrow(/only calls the API/);
    await expect(apiClient("/login")).rejects.toThrow(/only calls the API/);
    expect(fetch).not.toHaveBeenCalled();
  });
});

describe("apiClientPage", () => {
  it("reads the pagination headers", async () => {
    mockFetch(json(200, [{ id: "a" }, { id: "b" }], { "X-Total-Count": "57", "X-Page": "3", "X-Page-Size": "2" }));
    await expect(apiClientPage("/api/books?page=3&pageSize=2")).resolves.toEqual({ items: [{ id: "a" }, { id: "b" }], total: 57, page: 3, pageSize: 2 });
  });

  it("defaults sensibly without headers", async () => {
    mockFetch(json(200, [{ id: "a" }]));
    await expect(apiClientPage("/api/shelves")).resolves.toEqual({ items: [{ id: "a" }], total: 1, page: 1, pageSize: 1 });
  });
});

describe("apiClientWithHeaders", () => {
  it("returns the body and the response headers", async () => {
    mockFetch(json(200, [{ bookId: "b" }], { "X-Query-Id": "q05" }));
    const { data, headers } = await apiClientWithHeaders("/api/reports/popular");
    expect(data).toEqual([{ bookId: "b" }]);
    expect(headers.get("X-Query-Id")).toBe("q05");
  });
});

describe("query", () => {
  it("drops empty values and encodes the rest", () => {
    expect(query({ q: "dune & co", page: 2, title: "", author: undefined, categoryId: null })).toBe("?q=dune+%26+co&page=2");
    expect(query({})).toBe("");
  });
});

describe("errorMessage", () => {
  it("shows the API message for errors the user can fix, and hides server details", () => {
    expect(errorMessage(new ApiError(409, "CONFLICT", "An account with this email already exists."))).toBe("An account with this email already exists.");
    expect(errorMessage(new ApiError(500, "INTERNAL", "pq: relation does not exist"))).not.toContain("pq:");
    expect(errorMessage(new ApiError(502, "UPSTREAM_UNAVAILABLE", "Open Library is not responding"))).toMatch(/book search service/);
    expect(errorMessage(new Error("boom"))).toBe("Something went wrong. Try again.");
  });
});
