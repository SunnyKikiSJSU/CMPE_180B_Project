# Index benchmarks

Before/after evidence for the indexes in the migrations. Results and
explanations: [`docs/evidence/performance.md`](../../docs/evidence/performance.md).

| File | What it does |
| --- | --- |
| `01_generate_data.sql` | Loads about 820,000 deterministic rows (50,000 books, 2,000 readers plus one heavy reader). Refuses to run if the database already has users or books |
| `02_benchmark.sql` | Times nine app queries with and without their index (`EXPLAIN (ANALYZE, BUFFERS)`, median of 7 runs), prints a table and the plans. Indexes are dropped only inside a rolled-back subtransaction |

**Use a separate, disposable database.** The generator turns triggers off while
loading (`session_replication_role`, which needs a superuser) and adds a lot of
data. Never point it at RDS or your development database.

Local Docker example with a throwaway PostgreSQL 18 container:

```bash
docker network create bm-test
docker run -d --name bm-pg --network bm-test -e POSTGRES_USER=book_app \
  -e POSTGRES_PASSWORD=local_dev_password_only -e POSTGRES_DB=book_metaverse postgres:18-alpine
sleep 5   # let PostgreSQL start
docker exec bm-pg createdb -U book_app bm_perf
# Apply migrations 000001-000008 to bm_perf, for example:
docker run --rm --network bm-test -v "$PWD/database/migrations:/migrations:ro" migrate/migrate:v4.18.3 \
  -path=/migrations -database "postgres://book_app:local_dev_password_only@bm-pg:5432/bm_perf?sslmode=disable" up
docker exec -i bm-pg psql -U book_app -d bm_perf < database/performance/01_generate_data.sql   # about 10 s
docker exec -i bm-pg psql -U book_app -d bm_perf < database/performance/02_benchmark.sql      # about 2 s
docker rm -f bm-pg && docker network rm bm-test   # when done
```

Times depend on the machine; page counts (buffers) should match the recorded
results exactly because the data is generated with a fixed seed.
