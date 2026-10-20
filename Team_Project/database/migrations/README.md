# Database Migrations

Place versioned `golang-migrate` migration pairs in this directory:

```text
000001_initial_schema.up.sql
000001_initial_schema.down.sql
```

The Compose `migrate` service mounts this directory at `/migrations` and runs
all pending migrations against the local PostgreSQL database.
