"use client";

import { useEffect } from "react";

import { ErrorState } from "@/components/ErrorState";
import { useQueryCatalog } from "@/hooks/use-reports";

/**
 * GET /api/reports/queries: the evidence queries (q01-q10) with their fixed
 * SQL, shown as plain text. `target` (?sql=q05) opens and scrolls to one entry.
 */
export function QueryCatalog({ target }: { target: string }) {
  const catalog = useQueryCatalog();

  useEffect(() => {
    if (target && catalog.data) document.getElementById(`query-${target}`)?.scrollIntoView({ block: "start" });
  }, [target, catalog.data]);

  if (catalog.isPending) return <p className="muted-note" role="status">Loading the query catalog…</p>;
  if (catalog.isError) return <ErrorState error={catalog.error} title="The query catalog did not load" onRetry={() => void catalog.refetch()} />;
  return (
    <section className="report-section" aria-label="SQL queries">
      <div className="report-heading">
        <div>
          <h2>SQL behind the app</h2>
          <p>The named queries each feature runs, exactly as the API sends them to PostgreSQL ($1, $2 … are bound parameters). The text is fixed in the server and never built from input.</p>
        </div>
      </div>
      <ol className="query-catalog">
        {catalog.data.map((entry) => (
          <li key={entry.id} id={`query-${entry.id}`} className={entry.id === target ? "is-target" : undefined}>
            <h3><span className="query-badge">{entry.id}</span> {entry.name}</h3>
            <p>{entry.description}</p>
            <p className="query-meta">
              {entry.endpoints.map((e) => <code key={e}>{e}</code>)}
              <span>Queries: {entry.queries.join(", ")}</span>
            </p>
            {/* key: a new target remounts the entry so `open` applies again. */}
            <details key={entry.id === target ? "open" : "closed"} open={entry.id === target}>
              <summary>Show SQL</summary>
              <pre className="sql-block"><code>{entry.sql}</code></pre>
            </details>
          </li>
        ))}
      </ol>
    </section>
  );
}
