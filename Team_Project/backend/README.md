# Backend

Minimal Go API for the Book Metaverse Portal.

## Endpoints

- `/healthz` checks that the API process is running.
- `/readyz` checks that the API can connect to PostgreSQL.

The API reads database connection details from `DB_HOST`, `DB_PORT`, `DB_NAME`,
`DB_USER`, `DB_PASSWORD`, and `DB_SSLMODE`. It also accepts a complete
`DATABASE_URL`.

Run the complete environment from WSL:

```bash
./scripts/dev.sh local
./scripts/dev.sh rds
```

See [Development Setup](../DEVELOPMENT.md) for environment variables,
troubleshooting, logs, and shutdown commands.
