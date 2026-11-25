# AWS Plan — Book Metaverse Portal

Companion to [`BACKEND_PLAN.md`](BACKEND_PLAN.md) (Phase 10) and
[`FRONTEND_PLAN.md`](FRONTEND_PLAN.md) (Phase F8). It records what exists in AWS
and Terraform today, what the deployment stage still needs, and the S3 work that
is possible after the MVP.

This is a course project, so the plan is deliberately small: one RDS database,
one EC2 instance, HTTPS through Caddy, and secrets in the gitignored `.env` file.

- **Last updated:** 2026-11-25 (deployed with HTTPS at https://3-136-227-184.sslip.io; see [`docs/evidence/deployment.md`](docs/evidence/deployment.md))
- **Terraform + Terragrunt:** [`infra/`](infra/) (AWS provider `~> 5.0`, region `us-east-2`, the only region the course organization allows);
  units `app-sg` → `data` → `app` under `infra/live/dev` (restructured 2026-11-19),
  written by the team. Applied 2026-11-20 (20 resources). The console-created
  database `book-metaverse-database` is not managed by Terraform and was not
  changed.
- **Deployment target:** one EC2 instance running Docker Compose against private
  RDS (decided 2026-11-12; see §3)

### Status legend

| Mark | Meaning |
| --- | --- |
| `[x]` | Written and checked (`terraform validate` and `fmt -check` passed 2026-11-12) |
| `[?]` | Written, but not applied (none left as of 2026-11-20) |
| `[ ]` | Not started |

Everything below marked `[x]` was applied on 2026-11-20 with Terragrunt
(state in S3 bucket `cmpe180b-tfstate-869093593722`). There was no earlier
Terraform state: the team's first database was created in the console.

---

## 1. Done

### 1.1 Data tier (Terraform)

| Item | Resource | State | Notes |
| --- | --- | --- | --- |
| RDS for PostgreSQL 18 | `aws_db_instance.this` | `[x]` applied | `cmpe180b-team-project-db`: `db.t4g.micro`, 20 GB gp3, encrypted, single-AZ, default VPC subnets, `publicly_accessible = false` |
| Database security group | `aws_security_group.db` + rules | `[x]` applied | Port 5432 from `backend_security_group_id` or `allowed_cidr_blocks` only; both empty by default, so nothing can connect until one is set |
| Master password | `random_password.db` | `[x]` applied | 24 characters, generated when `db_password` is unset |
| Database secret | `aws_secretsmanager_secret.db` (`secrets.tf`) | `[x]` applied | JSON `{engine, host, port, dbname, username, password}`; the only place the generated password is stored outside Terraform state |
| Backend IAM role | `aws_iam_role.backend`, `aws_iam_instance_profile.backend` (`iam.tf`) | `[x]` applied | EC2 assume-role; may read the database secret and nothing else |
| Outputs | `outputs.tf` | `[x]` | `db_address`, `db_port`, `db_name`, `db_endpoint`, `db_security_group_id`, `db_secret_arn`, `backend_iam_role_name`, `backend_instance_profile_name` |

The API reads this secret at startup when `DB_SECRET_ARN` is set (see §2).
Otherwise, look up the database credentials once:

```bash
aws secretsmanager get-secret-value \
  --secret-id "$(terragrunt output -raw db_secret_arn)" \
  --query SecretString --output text
```

The JSON has `host`, `dbname`, `username`, and `password`. Copy them into
`.env` as `RDS_HOST`, `RDS_DB`, `RDS_USER`, and `RDS_PASSWORD`.

### 1.2 Tooling that already works with RDS

| Item | State | Notes |
| --- | --- | --- |
| `docker-compose-rdbs.yml` | `[x]` | `api`, `migrate`, `seed`, and a `psql` client pointed at `RDS_*` values with `sslmode=require` |
| `./scripts/dev.sh rds` | `[x]` | Builds `RDS_DATABASE_URL`, runs migrations and seeds, starts the API, waits for `/readyz` |
| Migrations and seeds | `[x]` | `000001`-`000008` and seeds `001`-`004` verified on clean PostgreSQL 18 locally; not yet recorded against RDS |
| `.env.example` RDS values | `[x]` | `RDS_DB=appdb` and `RDS_USER=appadmin` match the Terraform defaults; comments say where each value comes from (a Terraform output or the secret) |

### 1.3 Optional data-tier hardening

None of these are required for the course. Each is an in-place or recreate
change to `rds.tf`, which is the team's file, so make them only with the team's
agreement, and before the first `terraform apply` if the database does not exist
yet.

| Item | Effect | Note |
| --- | --- | --- |
| `storage_encrypted = true` | Encrypts data at rest | Forces replacement on an existing instance |
| `backup_retention_period = 7` | Explicit backups, which the restore check in §4.4 uses | In-place change |
| `db_name = "book_metaverse"` | Matches the local database name | Forces replacement on an existing instance. Not needed: use `appdb` in `.env` |

---

## 2. Decisions that keep the plan small

| Topic | Decision |
| --- | --- |
| Compute | EC2, not ECS Fargate. No ECR, ALB, CloudFront, Route 53, or CloudWatch |
| Application secrets | `.env` on the instance (mode 600, never committed): `JWT_SECRET` and either the database password or `DB_SECRET_ARN`. With `DB_SECRET_ARN` set, the API reads the database credentials from the team's Secrets Manager secret at startup using the instance profile from `iam.tf` (conflict #4, decided 2026-11-19). Migrations and seeds still use the `RDS_*` values |
| `infra/README.md` guidance | It says the backend should read the database secret at startup. The API supports both: `DB_SECRET_ARN` (recommended on EC2) or the `DB_*` values from `.env` (local and `./scripts/dev.sh rds`) |
| JWT secret | `.env` (Open Library needs no key) |
| HTTPS | Caddy (`proxy` service, profile `https` in `docker-compose-rdbs.yml`) on 80/443 with a Let's Encrypt certificate for `3-136-227-184.sslip.io` (sslip.io maps the name to the Elastic IP, so no domain is bought); HTTP redirects to HTTPS; `COOKIE_SECURE=true`. Added 2026-11-25 |
| Same-site cookies, CORS | The browser only talks to the Next.js container, which forwards `/api/*` to the API over the Compose network. No CORS, no cross-site cookie question |
| Rate limiting | None |
| Database account | The API connects as the RDS master user |

Passwords and the session cookie travel only over HTTPS. Still use throwaway
demo accounts (`reader1@example.com` and friends); the seed passwords are
public in the repository.

---

## 3. Deployment design (EC2)

```text
Browser ──HTTPS :443──► EC2 instance (Docker Compose)
(:80 → 308)           ┌─────────────────────────────────────┐
                      │ proxy (Caddy, :80/:443, Let's Encrypt)│
                      │   ──► web (Next.js, :3000)           │
                      │         /api/* ──rewrite──► api :8080│
                      │ api  (Go, :8080 on the instance only)│
                      └─────────────────────────────────────┘
                                   │ 5432, sslmode=require
                                   ▼
                      RDS PostgreSQL 18 (private)
```

Why EC2 and not ECS Fargate:

- The Compose files, the RDS variant, and `dev.sh rds` already exist.
  Fargate would add ECR, a cluster, task and service definitions, task roles,
  log groups, and a load balancer.
- One instance is cheaper and quicker to set up. Check current pricing and the
  account's free-tier or credit status before choosing the instance size.
- Logs are `docker compose logs`, and you can SSH in during the demo.
- The API's port 8080 is published on the instance because `./scripts/dev.sh`
  checks `http://127.0.0.1:8080/readyz`. The security group opens only ports
  80 and 443 (and 22 for SSH from listed CIDRs, none by default), so 8080 and
  3000 are not reachable from the internet (checked 2026-11-25).

Trade-offs accepted: a single point of failure, manual patching, and manual
deploys. Revisit if the course rubric requires ECS or autoscaling.

---

## 4. Remaining tasks — deployment stage

Maps to `BACKEND_PLAN.md` Phase 10 and `FRONTEND_PLAN.md` Phase F8.

### 4.1 Before deployment (in other plans)

- [x] Backend Phases 2–7 complete (`/readyz` is a database ping).
- [x] `frontend/Dockerfile`, built with the argument
      `API_PROXY_TARGET=http://api:8080`, and a `web` service in
      `docker-compose-rdbs.yml` (F8, 2026-11-25).

### 4.2 Terraform additions (`infra/`)

Written 2026-11-19 as Terragrunt units (`infra/README.md`); applied 2026-11-20
in us-east-2.

- [x] `modules/app_security_group` (unit `app-sg`): port 80 from anywhere;
      port 22 only from `ssh_cidr_blocks` (empty by default; `0.0.0.0/0` rejected).
- [x] `modules/app` (unit `app`): Amazon Linux 2023 (SSM parameter, later AMI
      releases ignored), `t3.small`, 30 GB encrypted gp3, IMDSv2 required (hop
      limit 2 for the API container), the backend instance profile, `user_data`
      that installs Docker, git, and pinned, checksum-verified Compose and Buildx
      plugins; optional `aws_eip.app`; optional Session Manager policy so SSH can
      stay closed.
- [x] RDS accepts only the app security group: the `data` unit passes the
      `app-sg` output as `backend_security_group_id` (no change to `network.tf`).
- [x] Variables: `instance_type`, `root_volume_gb`, `key_name`, `create_eip`,
      `enable_ssm`, `ssh_cidr_blocks`.
- [x] Outputs: `app_public_ip`, `app_url`, `instance_id`.
- [x] `terragrunt run --all -- plan` and `apply` (2026-11-20). No state had to
      move: the earlier Terraform was never applied, and the console-created
      `book-metaverse-database` is left as it is (`infra/README.md`).

### 4.3 Deploying (no new scripts)

On the instance, over Session Manager
(`aws ssm start-session --profile cmpe180b --target <instance_id>`):

1. `git clone` the repository and `git checkout sunny` (or the merged branch).
2. Create `.env` from `.env.example`: fill in the `RDS_*` values (host from
   `terragrunt output -raw db_address` in `infra/live/dev/data`; database, user, and password from the
   Secrets Manager secret, §1.1) and a random `JWT_SECRET` (Open Library needs no key).
   Run `chmod 600 .env`.
3. Set `COOKIE_SECURE=true` and `SITE_ADDRESS=3-136-227-184.sslip.io` in
   `.env`, run `./scripts/dev.sh rds` (migrations, seeds, API, web), then
   `docker compose --env-file .env -f docker-compose-rdbs.yml -p book-metaverse-rds --profile https up -d proxy`
   (Caddy, HTTPS on 443 with a Let's Encrypt certificate).
4. Open `https://3-136-227-184.sslip.io/`.

The t3.small has 2 GB of memory; add a 2 GB swap file before the first build.

Seeds run only on the first deploy; they are idempotent but 003 takes about 11
seconds.

### 4.4 Verification (BACKEND_PLAN Phase 10 exit)

- [x] `/readyz` returns 200.
- [x] RDS rejects connections from anywhere except the instance's security
      group.
- [x] Register, log in, Open Library search, import, shelve, progress, review, and
      every report work on the public address.
- [x] The session cookie is `HttpOnly`, `Secure`, and `SameSite=Lax` (HTTPS
      through Caddy since 2026-11-25).
- [x] `docker compose logs` shows readable output.

Done 2026-11-25; results and steps in
[`docs/evidence/deployment.md`](docs/evidence/deployment.md). Live at
https://3-136-227-184.sslip.io.
- [ ] Restore an RDS snapshot into a separate instance and connect to it
      (`BACKEND_PLAN.md` §11.3).

### 4.5 Cost control and teardown

- [ ] Stop the instance between work sessions; RDS can be stopped for up to
      7 days at a time.
- [ ] Public IPv4 addresses, including Elastic IPs, are billed hourly; release
      the Elastic IP when the project ends.
- [ ] `terragrunt run --all -- destroy` after grading. `skip_final_snapshot = true`, so take a
      manual snapshot first if the data should be kept.

---

## 5. Post-MVP — S3 (only if time allows)

Today nothing uses S3: covers are covers.openlibrary.org or `picsum.photos` URLs in
`books.cover_url`, and `users.avatar_url` is a plain URL. The MVP does not need
S3.

### 5.1 Scope

| Use | In scope | Why |
| --- | --- | --- |
| User avatars | Yes | Users can upload a picture instead of pasting a URL |
| Covers for user-created books (`source = 'user'`) | Yes | Hand-added books often have no cover URL |
| Copies of Open Library covers | **No** | Open Library asks sites to link its covers rather than copy or crawl them; keep linking |

### 5.2 Design

- One private bucket with S3 Block Public Access on and default encryption.
- Browser uploads go straight to S3 with a **presigned POST** issued by the API,
  so files never pass through the Go server. A presigned POST policy can enforce
  `content-length-range` (for example 2 MB) and `Content-Type` (`image/jpeg`,
  `image/png`, `image/webp`); a presigned PUT cannot limit size.
- New endpoint: `POST /api/uploads` with `{ purpose: "avatar" | "cover", bookId? }`
  → `{ url, fields, key }`. Keys: `avatars/<user_id>/<uuid>`,
  `covers/<book_id>/<uuid>`.
- After upload the client saves the result with the existing endpoints:
  `PATCH /api/users/me { avatarUrl }` and `PATCH /api/books/{bookId} { coverUrl }`.
  No schema change is required.
- Serving: short-lived presigned GET URLs.
- **IAM.** The Go API needs AWS credentials to sign URLs. Attach the existing
  `aws_iam_instance_profile.backend` to the instance and add one policy to
  `aws_iam_role.backend` that allows `s3:PutObject` and `s3:GetObject` on the
  bucket's `avatars/*` and `covers/*` prefixes only.
- Bucket CORS: allow `POST` from the site's origin only (this is the one place
  CORS is needed, because the browser posts straight to S3).

### 5.3 Tasks

- [ ] `infra/s3.tf`: bucket, Block Public Access, encryption, CORS, lifecycle
      rule for abandoned uploads.
- [ ] Add the two-prefix S3 policy to the existing backend role (`infra/iam.tf`)
      and attach its instance profile to `aws_instance.app`.
- [ ] Go: AWS SDK for the presigner, `POST /api/uploads`, ownership checks
      (only the owner for avatars; only `source = 'user'` books for covers).
- [ ] Contract: add the endpoint to `openapi.yaml`.
- [ ] Frontend: upload control on `/settings` and on user-created book detail.
- [ ] Remove the user's avatar object when the account is deleted.

---

## 6. Open decisions

1. Which SSH key pair and which IPs may SSH to the instance.
2. Whether the course asks for HTTPS, ECS, autoscaling, or CloudWatch. If it
   does, §2 and §3 change: HTTPS needs a domain name plus a reverse proxy with a
   certificate in front of `web`, and ECS adds ECR, a cluster, and a load
   balancer.
3. Whether to apply the optional hardening in §1.3 (only worth it before the
   first `terraform apply`).
