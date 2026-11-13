# Open conflicts and decisions

Items that still need a decision or a check before the work they affect. Each
row names who decides and when it is needed. Detailed notes are in the
working-notes file `AGENTS.md` (items 27, 29, 31, and 33, kept locally and
git-ignored) and in the plans linked below.

| # | Open item | What conflicts | Who decides | Needed by |
| --- | --- | --- | --- | --- |
| 1 | **ER diagram cardinality labels.** The four `1..1` labels at `BOOKS` on reading progress, reading history, ratings, and reviews should be `0..*`. Users and books are many-to-many, limited to one row per user–book pair for progress, ratings, and reviews. | `ER_diagram.png` shows `1..1`; the schema and `ER_diagram.md` §4 say many-to-many. | You (edit the PNG) | Before the final report and presentation |
| 2 | **"My reviews" and "my ratings" rows have no book details.** `ListMyReviews` and `ListMyRatings` return the book title and cover, but the `Review` and `Rating` schemas in `openapi.yaml` have no book field, so the `/reviews` page can show only a book ID. Options: add an optional `book` object to both schemas, or have the UI fetch titles separately. | Contract vs. query columns (`AGENTS.md` item 29). | Team | Phase 6 |
| 3 | **Local `.env` may not match the Terraform database name.** `.env.example` uses `RDS_DB=appdb` and `RDS_USER=appadmin`, the Terraform defaults. A local `.env` copied from an older example may use a different name. Check the value against the Secrets Manager secret (`AWS_PLAN.md` §1.1) without printing the password. | `RDS_DB` in `.env` vs. the database Terraform creates (`AGENTS.md` item 31). | You | Before `./scripts/dev.sh rds` |
| 4 | **Where the deployed API reads secrets.** `infra/README.md` (written by a groupmate) says the backend should read the database secret from Secrets Manager at startup and never from environment variables. The plans read every secret from `.env`. | `infra/README.md` vs. `AWS_PLAN.md` §2 and `BACKEND_PLAN.md` §8 (`AGENTS.md` item 33). | You and the author of `infra/` | Phase 10 (deployment) |
| 5 | **Storing books users act on.** Google's API terms restrict keeping permanent copies of API content. The design imports a Google book only when a user shelves, rates, reviews, or tracks it. | `BACKEND_PLAN.md` §8 and §16 (decision 4) vs. the import design in §6.4. Fallback if refused: store only `google_books_id` and the user's own data, and fetch display metadata from Google when needed. | Instructor | Phase 4 (before the import endpoint is built) |

## Not open

These are constraints already recorded and followed. They are listed here so
nobody reopens them by accident.

- The API port 8080 stays published on the instance, because `./scripts/dev.sh`
  checks `/readyz` from the host. The EC2 security group opens only ports 80
  and 22 (`AGENTS.md` item 30).
- The contract's field names differ from some sqlc row names. The service
  layer renames or combines them; no query needs to change (`AGENTS.md` item 28).
- `infra/` belongs to a teammate and is not changed by this project's work.
