# Open conflicts and decisions

Items that still need a decision or a check before the work they affect. Each
row names who decides and when it is needed. Detailed notes are in the
working-notes file `AGENTS.md` (items 27, 29, 31, and 33, kept locally and
git-ignored) and in the plans linked below.

| # | Open item | What conflicts | Who decides | Needed by |
| --- | --- | --- | --- | --- |
| 1 | **ER diagram cardinality labels.** The four `1..1` labels at `BOOKS` on reading progress, reading history, ratings, and reviews should be `0..*`. Users and books are many-to-many, limited to one row per user–book pair for progress, ratings, and reviews. | `ER_diagram.png` shows `1..1`; the schema and `ER_diagram.md` §4 say many-to-many. Also add `openlibrary_id UK` to `BOOKS` (migration `000008`). | You (edit the PNG) | Before the final report and presentation |
| 2 | **"My reviews" and "my ratings" rows have no book details.** `ListMyReviews` and `ListMyRatings` return the book title and cover, but the `Review` and `Rating` schemas in `openapi.yaml` have no book field, so the `/reviews` page can show only a book ID. Options: add an optional `book` object to both schemas, or have the UI fetch titles separately. Phase 6 returns the schemas as written; adding an optional `book` object later does not break existing clients. | Contract vs. query columns (`AGENTS.md` item 29). | Team | Phase 8 (the `/reviews` page) |
| 3 | **Local `.env` may not match the Terraform database name.** `.env.example` uses `RDS_DB=appdb` and `RDS_USER=appadmin`, the Terraform defaults. A local `.env` copied from an older example may use a different name. Check the value against the Secrets Manager secret (`AWS_PLAN.md` §1.1) without printing the password. | `RDS_DB` in `.env` vs. the database Terraform creates (`AGENTS.md` item 31). | You | Before `./scripts/dev.sh rds` |

## Not open

These are constraints already recorded and followed. They are listed here so
nobody reopens them by accident.

- **Where outside books come from (was #5, decided 2026-11-23).** Google's API
  terms restrict keeping copies of its content, which needed the instructor's
  answer. Outside search and import use **Open Library** instead: the Internet
  Archive asserts no copyright over its catalog data, so imported works are
  stored like other books, and covers are linked from covers.openlibrary.org,
  not copied (its cover guidelines). A Google source can be added later; the
  `google_books_id` column and `source = 'google'` value stay reserved. See
  `BACKEND_PLAN.md` Phase 4.

- **Where the deployed API reads database credentials (was #4, decided
  2026-11-19).** Both: when `DB_SECRET_ARN` is set, the API reads the
  Terraform Secrets Manager secret at startup with the instance role;
  otherwise it uses the `DB_*` values from `.env`. `DATABASE_URL` wins over
  both. See `backend/README.md` and `BACKEND_PLAN.md` §8.

- The API port 8080 stays published on the instance, because `./scripts/dev.sh`
  checks `/readyz` from the host. The EC2 security group opens only ports 80
  and 22 (`AGENTS.md` item 30).
- The contract's field names differ from some sqlc row names. The service
  layer renames or combines them; no query needs to change (`AGENTS.md` item 28).
- `infra/` was restructured into Terragrunt units with the agreement of its
  author (2026-11-19). The data tier's resources and addresses are unchanged;
  only where its state lives moves (`infra/README.md`).
