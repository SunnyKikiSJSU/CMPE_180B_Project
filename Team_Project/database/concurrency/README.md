# Concurrency scripts

Multi-session evidence for `BACKEND_PLAN.md` §7.2. Each script opens real
concurrent `psql` sessions, prints what each one saw, and checks the final
state. Results and explanations: [`docs/evidence/concurrency.md`](../../docs/evidence/concurrency.md).

| Script | Scenario |
| --- | --- |
| `rating_summary_race.sh` | Two users rate one book at the same time (`READ COMMITTED`), then the same race with the second session at `REPEATABLE READ` |
| `duplicate_review_race.sh` | One user submits the same review from two sessions |
| `_lib.sh` | Shared helpers (sourced, not run) |

Run against a migrated database (`000001`-`000007`); seeds are not needed:

```bash
DATABASE_URL=postgres://user:password@host:5432/book_metaverse database/concurrency/rating_summary_race.sh
# Local Docker database:
PSQL="docker exec -i bm-pg psql -U book_app -d book_metaverse" database/concurrency/duplicate_review_race.sh
```

The scripts create throwaway users and books (title `Concurrency: ...`) and
delete them on exit. Each run takes about 5 seconds because session A holds
its transaction open for 2 seconds.
