import type { ReactNode } from "react";

import { errorMessage, isApiError } from "@/lib/errors";

/** An API error with a safe message, its request ID (for support), and recovery actions. */
export function ErrorState({ error, title = "That did not load", onRetry, children }: { error: unknown; title?: string; onRetry?: () => void; children?: ReactNode }) {
  const requestId = isApiError(error) ? error.requestId : undefined;
  // Retrying cannot fix a 4xx such as a validation error, except too-many-requests.
  const retryable = !isApiError(error) || error.status === 0 || error.status === 429 || error.status >= 500;
  return (
    <div className="error-state" role="alert">
      <span className="empty-state-icon" aria-hidden="true">!</span>
      <h3>{title}</h3>
      <p>{errorMessage(error)}</p>
      {requestId && <small>Request ID {requestId}</small>}
      <div className="error-state-actions">
        {onRetry && retryable && <button className="button button-primary" type="button" onClick={onRetry}>Try again</button>}
        {children}
      </div>
    </div>
  );
}
