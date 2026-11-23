// Error codes from docs/api/errors.md, plus NETWORK_ERROR for a request that
// never reached the API (offline, proxy down).
export type ErrorCode =
  | "VALIDATION_ERROR"
  | "UNAUTHENTICATED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "STALE_VERSION"
  | "UPSTREAM_UNAVAILABLE"
  | "DATABASE_UNAVAILABLE"
  | "INTERNAL"
  | "NETWORK_ERROR";

/** An API failure: the error envelope { error: { code, message, details, requestId } }. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ErrorCode | string;
  readonly details?: Record<string, unknown>;
  readonly requestId?: string;

  constructor(status: number, code: string, message: string, details?: Record<string, unknown>, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    this.requestId = requestId;
  }

  /** The request field a validation or conflict error is about, if any. */
  get field(): string | undefined {
    const field = this.details?.field;
    return typeof field === "string" ? field : undefined;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

const fallbackMessages: Record<string, string> = {
  UNAUTHENTICATED: "Please sign in to continue.",
  FORBIDDEN: "You are not allowed to do that.",
  NOT_FOUND: "We could not find that.",
  CONFLICT: "That already exists.",
  STALE_VERSION: "This changed in another tab or device. Reload to see the latest version.",
  UPSTREAM_UNAVAILABLE: "The book search service is not responding. Try again in a moment.",
  DATABASE_UNAVAILABLE: "The library is temporarily unavailable. Try again in a moment.",
  NETWORK_ERROR: "Cannot reach the server. Check your connection and try again.",
};

/**
 * A message that is safe to show. The API's own message is used for errors
 * the user can act on (validation, conflict, not found); server-side failures
 * get a fixed message so no internal detail reaches the page.
 */
export function errorMessage(error: unknown): string {
  if (!isApiError(error)) return "Something went wrong. Try again.";
  if (error.status >= 500 || error.code === "INTERNAL") {
    return fallbackMessages[error.code] ?? "Something went wrong on our side. Try again later.";
  }
  return error.message || fallbackMessages[error.code] || "Something went wrong. Try again.";
}
