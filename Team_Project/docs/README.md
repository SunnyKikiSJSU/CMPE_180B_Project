# Documentation

Project documentation and evidence are organized here.

Current and planned sections:

- `api/` for the OpenAPI contract, error catalog, and endpoint examples.
  - `openapi.yaml` is the frozen request/response contract.
  - `errors.md` documents the error envelope and SQLSTATE mapping.
  - `examples.md` contains curl examples for each endpoint group.
- `database/` for the schema dictionary and query catalog.
  - [`schema-dictionary.md`](database/schema-dictionary.md): every table,
    column, constraint, index, trigger, and function, with ownership and
    cascade rules.
  - [`query-catalog.md`](database/query-catalog.md): the ten evidence
    queries (q01-q10) with purpose, SQL, and expected output.
- `diagrams/` for other reviewed project diagrams. The ER diagram is
  [`../ER_diagram.png`](../ER_diagram.png), explained in
  [`../ER_diagram.md`](../ER_diagram.md).
- `runbooks/` for local, WSL, Docker, and AWS procedures.
- `evidence/` for validation, concurrency, and performance results.
  - [`transactions.md`](evidence/transactions.md): shelf move and progress
    completion, committed and rolled back (Phase 5).
  - [`validation.md`](evidence/validation.md): 34 database validation cases
    with expected and actual results (Phase 9).
  - [`concurrency.md`](evidence/concurrency.md): multi-session races (rating
    summary at two isolation levels, duplicate review, stale progress).
  - [`performance.md`](evidence/performance.md): five index optimizations
    with before/after plans and timings, and where indexes did not help.
  - [`quality-gates.md`](evidence/quality-gates.md): the final run of every
    test suite, linter, type check, build, and `terraform validate`.
- `architecture/` for architecture diagrams and design decisions.
