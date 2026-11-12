# Documentation

Project documentation and evidence are organized here.

Current and planned sections:

- `api/` for the OpenAPI contract, error catalog, and endpoint examples.
  - `openapi.yaml` is the frozen request/response contract.
  - `errors.md` documents the error envelope and SQLSTATE mapping.
  - `examples.md` contains curl examples for each endpoint group.
- `database/` for the schema dictionary and query catalog.
- `diagrams/` for other reviewed project diagrams. The ER diagram is
  [`../ER_diagram.png`](../ER_diagram.png), explained in
  [`../ER_diagram.md`](../ER_diagram.md).
- `runbooks/` for local, WSL, Docker, and AWS procedures.
- `evidence/` for validation, concurrency, and performance results.
- `architecture/` for architecture diagrams and design decisions.
