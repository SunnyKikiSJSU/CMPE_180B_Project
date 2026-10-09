# Team Project

## Architecture

Three-tier architecture:

```
 browser  -->  frontend (presentation tier)  -->  backend (application/logic tier)  -->  Amazon RDS (data tier)
```

- **`frontend/`** — presentation tier. Renders the UI and talks to the backend over HTTP/REST (or another agreed API style).
- **`backend/`** — application/logic tier. Owns all business logic and is the only tier that holds database credentials; exposes an API to the frontend.
- **`docs/`** — design docs, diagrams, meeting notes, and any other project documentation.

## Data Tier

Backend connects to an **Amazon RDS** instance (engine TBD — e.g. MySQL/PostgreSQL). No tier other than the backend should hold RDS credentials or query the database directly.

## Status

Scaffolding only — tech stack choices for frontend/backend are still TBD by the team.
