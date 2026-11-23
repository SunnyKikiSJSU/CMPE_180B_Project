import { ApiError } from "@/lib/errors";
import type { Page } from "@/types/domain";

// Requests use relative /api/... URLs; next.config.ts forwards them to the Go
// API, so the bm_session cookie is same-origin and fetch sends it by default.

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  /** Sent as JSON. DELETE may have a body (DELETE /api/users/me needs {"confirm": true}). */
  body?: unknown;
  signal?: AbortSignal;
}

async function send(path: string, options: RequestOptions = {}): Promise<Response> {
  if (!path.startsWith("/api/")) throw new Error(`apiClient only calls the API (/api/...), got ${path}`);
  const headers: Record<string, string> = { Accept: "application/json" };
  let body: string | undefined;
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }
  let response: Response;
  try {
    response = await fetch(path, { method: options.method ?? "GET", headers, body, signal: options.signal, credentials: "same-origin" });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") throw error;
    throw new ApiError(0, "NETWORK_ERROR", "Cannot reach the server.");
  }
  if (!response.ok) throw await toApiError(response);
  return response;
}

async function toApiError(response: Response): Promise<ApiError> {
  const headerId = response.headers.get("X-Request-ID") ?? undefined;
  try {
    const envelope = (await response.json()) as { error?: { code?: string; message?: string; details?: Record<string, unknown>; requestId?: string } };
    const e = envelope.error;
    if (e?.code) return new ApiError(response.status, e.code, e.message ?? "", e.details, e.requestId ?? headerId);
  } catch {
    // Not JSON (for example the Next.js proxy's own error page).
  }
  return new ApiError(response.status, "INTERNAL", `Request failed with status ${response.status}.`, undefined, headerId);
}

/** 204 and any other empty 2xx body return undefined. */
async function readJSON<T>(response: Response): Promise<T> {
  if (response.status === 204) return undefined as T;
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export async function apiClient<T>(path: string, options?: RequestOptions): Promise<T> {
  return readJSON<T>(await send(path, options));
}

function intHeader(response: Response, name: string, fallback: number): number {
  const n = Number.parseInt(response.headers.get(name) ?? "", 10);
  return Number.isFinite(n) ? n : fallback;
}

/** A list endpoint: the raw array plus X-Total-Count, X-Page, and X-Page-Size. */
export async function apiClientPage<T>(path: string, options?: RequestOptions): Promise<Page<T>> {
  const response = await send(path, options);
  const items = (await readJSON<T[]>(response)) ?? [];
  return {
    items,
    total: intHeader(response, "X-Total-Count", items.length),
    page: intHeader(response, "X-Page", 1),
    pageSize: intHeader(response, "X-Page-Size", items.length),
  };
}

/** Builds a query string from defined, non-empty values. */
export function query(params: Record<string, string | number | undefined | null>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== "") search.set(key, String(value));
  }
  const s = search.toString();
  return s ? `?${s}` : "";
}
