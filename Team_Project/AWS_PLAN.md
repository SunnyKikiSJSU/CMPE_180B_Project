# AWS Plan — Book Metaverse Portal

Companion to [`BACKEND_PLAN.md`](BACKEND_PLAN.md) (Phase 10) and
[`FRONTEND_PLAN.md`](FRONTEND_PLAN.md) (Phase F8). It records what exists in AWS
and Terraform today, what the deployment stage still needs, and the S3 work that
is possible after the MVP.

This is a course project, so the plan is deliberately small: one RDS database,
one EC2 instance, plain HTTP, and secrets in the gitignored `.env` file.

- **Last updated:** 2026-11-12
- **Terraform:** [`infra/`](infra/) (AWS provider `~> 5.0`, region `us-west-2`),
  written by the team. This plan does not remove or change any existing
  resource; it only lists additions.
- **Deployment target:** one EC2 instance running Docker Compose against private
  RDS (decided 2026-11-12; see §3)

### Status legend

| Mark | Meaning |
| --- | --- |
| `[x]` | Written and checked (`terraform validate` and `fmt -check` passed 2026-11-12) |
| `[?]` | Written, but whether it has been applied to the AWS account is not recorded in the repo |
| `[ ]` | Not started |

Terraform state is git-ignored, so the repository cannot show whether
`terraform apply` has been run. Whoever applied it should replace the `[?]`
marks below and note the date.

---

## 1. Done

### 1.1 Data tier (Terraform)

| Item | Resource | State | Notes |
| --- | --- | --- | --- |
| RDS for PostgreSQL 18 | `aws_db_instance.this` | `[x]` `[?]` | `db.t4g.micro`, 20 GB gp3, single-AZ, default VPC subnets, `publicly_accessible = false` |
| Database security group | `aws_security_group.db` + rules | `[x]` `[?]` | Port 5432 from `backend_security_group_id` or `allowed_cidr_blocks` only; both empty by default, so nothing can connect until one is set |
| Master password | `random_password.db` | `[x]` `[?]` | 24 characters, generated when `db_password` is unset |
| Database secret | `aws_secretsmanager_secret.db` (`secrets.tf`) | `[x]` `[?]` | JSON `{engine, host, port, dbname, username, password}`; the only place the generated password is stored outside Terraform state |
| Backend IAM role | `aws_iam_role.backend`, `aws_iam_instance_profile.backend` (`iam.tf`) | `[x]` `[?]` | EC2 assume-role; may read the database secret and nothing else |
| Outputs | `outputs.tf` | `[x]` | `db_address`, `db_port`, `db_name`, `db_endpoint`, `db_security_group_id`, `db_secret_arn`, `backend_iam_role_name`, `backend_instance_profile_name` |

The application does not read Secrets Manager at runtime (see §2). The secret is
where you look up the database credentials once:

```bash
aws secretsmanager get-secret-value \
  --secret-id "$(terraform output -raw db_secret_arn)" \
  --query SecretString --output text
```

The JSON has `host`, `dbname`, `username`, and `password`. Copy them into
`.env` as `RDS_HOST`, `RDS_DB`, `RDS_USER`, and `RDS_PASSWORD`.

### 1.2 Tooling that already works with RDS

| Item | State | Notes |
| --- | --- | --- |
| `docker-compose-rdbs.yml` | `[x]` | `api`, `migrate`, `seed`, and a `psql` client pointed at `RDS_*` values with `sslmode=require` |
| `./scripts/dev.sh rds` | `[x]` | Builds `RDS_DATABASE_URL`, runs migrations and seeds, starts the API, waits for `/readyz` |
| Migrations and seeds | `[x]` | `000001`-`000006` and seeds `001`-`004` verified on clean PostgreSQL 18 locally; not yet recorded against RDS |
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
| Application secrets | `.env` on the instance (mode 600, never committed): database password, `JWT_SECRET`, Google key. The Go code does not call Secrets Manager. The team's `secrets.tf` and `iam.tf` stay as they are; the secret is only where the database password is looked up |
| `infra/README.md` guidance | It says the backend should read the database secret at startup and never take credentials from environment variables. This plan does not follow that for the course project: the API reads environment variables from `.env`. Agree this with the author of `infra/` |
| JWT secret and Google key | `.env` |
| HTTPS | None. Plain HTTP on port 80, `COOKIE_SECURE=false` |
| Same-site cookies, CORS | The browser only talks to the Next.js container, which forwards `/api/*` to the API over the Compose network. No CORS, no cross-site cookie question |
| Rate limiting | None |
| Database account | The API connects as the RDS master user |

Because there is no HTTPS, passwords cross the network in clear text. Use only
throwaway demo accounts (`reader1@example.com` and friends), never a real
password. Add HTTPS only if the course asks for it (§6).

---

## 3. Deployment design (EC2)

```text
Browser ──HTTP :80──► EC2 instance (Docker Compose)
                      ┌─────────────────────────────────────┐
                      │ web  (Next.js, published on :80)    │
                      │   /api/*  ──rewrite──► api :8080    │
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
  checks `http://127.0.0.1:8080/readyz`. The security group opens only port 80
  (and 22 for SSH), so 8080 is not reachable from the internet.

Trade-offs accepted: a single point of failure, manual patching, and manual
deploys. Revisit if the course rubric requires ECS or autoscaling.

---

## 4. Remaining tasks — deployment stage

Maps to `BACKEND_PLAN.md` Phase 10 and `FRONTEND_PLAN.md` Phase F8.

### 4.1 Before deployment (in other plans)

- [ ] Backend Phases 2–7 complete (`/readyz` is a database ping).
- [ ] `frontend/Dockerfile`, built with the argument
      `API_PROXY_TARGET=http://api:8080`, and a `web` service in
      `docker-compose-rdbs.yml`.

### 4.2 Terraform additions (`infra/`)

- [ ] `ec2.tf`
  - `aws_security_group.app`: port 80 from anywhere; port 22 from
    `var.ssh_cidr_blocks` only.
  - `aws_instance.app`: Amazon Linux 2023 AMI (data source), `t3.small`
    (2 GB; building Next.js on 1 GB tends to run out of memory), 30 GB gp3 root
    volume, `key_name = var.key_name`, `user_data` that installs Docker, the
    Compose plugin, and `git`. Optionally attach the existing
    `aws_iam_instance_profile.backend`; the app needs no AWS access until S3.
  - `aws_eip.app` (optional) so the address survives stop/start.
- [ ] `network.tf`: point `from_backend` at `aws_security_group.app.id` instead
      of the variable, so RDS accepts only the instance.
- [ ] `variables.tf`: `instance_type`, `ssh_cidr_blocks`, `key_name`.
- [ ] `outputs.tf`: `app_public_ip`.
- [ ] `terraform validate`, `terraform plan`, then `apply`.

### 4.3 Deploying (no new scripts)

On the instance, over SSH:

1. `git clone` the repository and `git checkout sunny` (or the merged branch).
2. Create `.env` from `.env.example`: fill in the `RDS_*` values (host from
   `terraform output -raw db_address`; database, user, and password from the
   Secrets Manager secret, §1.1), a random `JWT_SECRET`, and the Google key.
   Run `chmod 600 .env`.
3. `./scripts/dev.sh rds` (migrations, seeds, API), then
   `docker compose -f docker-compose-rdbs.yml up -d --build web`.
4. Open `http://<instance address>/`.

Seeds run only on the first deploy; they are idempotent but 003 takes about 11
seconds.

### 4.4 Verification (BACKEND_PLAN Phase 10 exit)

- [ ] `/readyz` returns 200.
- [ ] RDS rejects connections from anywhere except the instance's security
      group.
- [ ] Register, log in, Google search, import, shelve, progress, review, and
      every report work on the public address.
- [ ] The session cookie is `HttpOnly` and `SameSite=Lax` (not `Secure`, since
      this is plain HTTP).
- [ ] `docker compose logs` shows readable output.
- [ ] Restore an RDS snapshot into a separate instance and connect to it
      (`BACKEND_PLAN.md` §11.3).

### 4.5 Cost control and teardown

- [ ] Stop the instance between work sessions; RDS can be stopped for up to
      7 days at a time.
- [ ] Public IPv4 addresses, including Elastic IPs, are billed hourly; release
      the Elastic IP when the project ends.
- [ ] `terraform destroy` after grading. `skip_final_snapshot = true`, so take a
      manual snapshot first if the data should be kept.

---

## 5. Post-MVP — S3 (only if time allows)

Today nothing uses S3: covers are Google Books or `picsum.photos` URLs in
`books.cover_url`, and `users.avatar_url` is a plain URL. The MVP does not need
S3.

### 5.1 Scope

| Use | In scope | Why |
| --- | --- | --- |
| User avatars | Yes | Users can upload a picture instead of pasting a URL |
| Covers for user-created books (`source = 'user'`) | Yes | Hand-added books often have no cover URL |
| Copies of Google Books covers | **No** | Storing Google content permanently conflicts with Google's API terms (see `BACKEND_PLAN.md` §8); keep hotlinking with attribution |

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
