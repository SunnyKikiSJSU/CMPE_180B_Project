# Quality gates

The Phase 9 checks (BACKEND_PLAN.md §9 and §13), run on 2026-11-25 against
`main` after Phase 9b. All passed.

| Gate | Command | Result |
| --- | --- | --- |
| Go formatting | `gofmt -l cmd internal` (generated `internal/db` excluded) | Clean (one test file had whitespace-only misalignment; fixed in this PR) |
| Go static checks | `go vet ./...` and `go vet -tags integration ./internal/integration/` | No findings |
| Go unit tests | `go test -race -count=1 ./...` | All 8 packages with tests pass |
| Go integration tests | `DATABASE_URL=... go test -tags integration -count=1 ./internal/integration/` | Pass (12 s) against PostgreSQL 18.7 with migrations `000001`-`000008` and seeds `001`-`004` |
| Transaction suites | `psql -v ON_ERROR_STOP=0 -f database/tests/transactions_phase5.sql`, `transactions_phase6.sql` | Exit 0; the 3 and 1 `ERROR` lines are the intended failing steps that trigger the rollbacks ([transactions.md](transactions.md)) |
| Validation cases | `psql -f database/tests/validation_cases.sql` | 34 passed, 0 failed ([validation.md](validation.md)) |
| Concurrency scripts | `database/concurrency/rating_summary_race.sh`, `duplicate_review_race.sh`, `stale_progress_race.sh` | Exit 0; every `result:` line matches its `expected:` line ([concurrency.md](concurrency.md)) |
| Index benchmarks | `database/performance/01_generate_data.sql`, `02_benchmark.sql` | Recorded in [performance.md](performance.md) |
| Frontend lint | `npm run lint` | No findings |
| Frontend types | `npx tsc --noEmit` | No errors |
| Frontend tests | `npx vitest run` | 11 files, 51 tests pass |
| Frontend build | `npm run build` (Next.js 16) | Builds all 16 routes |
| Terraform | `terraform init -backend=false && terraform validate` in each of `infra/modules/{app,app_security_group,budget,data}`; `terraform fmt -check -recursive infra` | All four valid; formatting clean |
| Shell scripts | `shellcheck -x database/concurrency/stale_progress_race.sh` | No findings |

Toolchain: Go 1.26.11, Node 25.8.2, Terraform 1.14.9, PostgreSQL 18.7
(`postgres:18-alpine`).

The test scripts leave no rows behind: after the run, the database has no
`validation-*`, `conc-*`, or `tx-*` users.

The deployed database was checked separately: the validation cases pass on
RDS too ([validation.md](validation.md)), and the live site was exercised end
to end ([deployment.md](deployment.md)).
