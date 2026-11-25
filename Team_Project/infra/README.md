# Infra (Terraform + Terragrunt)

AWS for the Book Metaverse Portal in `us-east-2` (the course organization's
service control policy denies every other region): the RDS data tier written
by the team, plus the EC2 app tier from [`AWS_PLAN.md`](../AWS_PLAN.md) §4.2.
Applied 2026-11-20; see [Current deployment](#current-deployment).
Terraform modules hold the resources; Terragrunt supplies remote state, the
provider, and the wiring between units.

## Layout

```text
infra/
├── root.hcl                    # S3 remote state (one key per unit, S3 lock file), aws provider, shared inputs
├── modules/
│   ├── data/                   # The team's RDS tier, unchanged: RDS PostgreSQL 18, DB security group,
│   │                           # random password, Secrets Manager secret, backend IAM role + instance profile
│   ├── app_security_group/     # App security group: in HTTP 80 and HTTPS 443 (+ SSH from listed CIDRs); out 443, 80, 5432 to the VPC
│   └── app/                    # EC2 (Amazon Linux 2023, Docker + Compose), optional Elastic IP, Session Manager
└── live/
    └── dev/
        ├── env.hcl             # environment, region, project_name
        ├── app-sg/terragrunt.hcl
        ├── data/terragrunt.hcl # + .terraform.lock.hcl (aws 5.100.0, random 3.9.1)
        └── app/terragrunt.hcl  # each unit commits its .terraform.lock.hcl
```

Order (Terragrunt resolves it from the `dependency` blocks): **app-sg → data → app**.
The app security group is its own unit so the database security group can allow
it without a cycle (the app instance needs the data tier's instance profile).

| Unit | Creates | Uses |
| --- | --- | --- |
| `app-sg` | `aws_security_group.app` and its rules | — |
| `data` | RDS, its security group and subnet group, the database secret, the backend IAM role | `app-sg` security group as `backend_security_group_id`, so RDS accepts only the app instance |
| `app` | `aws_instance.app`, `aws_eip.app`, Session Manager policy on the backend role | `app-sg` security group; `data` instance profile and role |

## Requirements

- Terraform >= 1.10 (S3 native locking) and Terragrunt 1.x, with
  `export TG_TF_PATH=terraform`. Terragrunt 1.x otherwise runs OpenTofu when it
  is installed, which rewrites the lock files for another registry.
- AWS CLI >= 2.32 and credentials for the **course account only**
  (`869093593722`). Team members sign in through the AWS access portal
  ("SJSU 180B Project Group"), which grants `AccountFullAccessRole`; that role
  cannot manage IAM Identity Center, so `aws configure sso` finds no account.
  Use the console sign-in instead:

  ```bash
  aws login --profile cmpe180b --region us-east-2   # pick "SJSU 180B Project Group" in the browser
  ```

  The Terraform AWS provider (5.100) cannot read `aws login` credentials yet,
  so add a second profile that asks the CLI for them:

  ```ini
  # ~/.aws/config
  [profile cmpe180b-tf]
  credential_process = aws configure export-credentials --profile cmpe180b --format process
  region = us-east-2
  ```

  The `cmpe180b` profile needs `region = us-east-2` too, or refreshing the
  login fails after the first token expires. Check before every run:
  `aws sts get-caller-identity --profile cmpe180b-tf` shows `869093593722`.
- State bucket: `cmpe180b-tfstate-869093593722` (us-east-2; versioned,
  KMS-encrypted, public access blocked; created with `--backend-bootstrap`).

## Usage

```bash
cd Team_Project/infra/live/dev
export AWS_PROFILE=cmpe180b-tf TG_TF_PATH=terraform TG_STATE_BUCKET=cmpe180b-tfstate-869093593722

terragrunt run --all -- init          # the bucket exists; --backend-bootstrap only for a new one

terragrunt run --all -- plan          # review every unit
terragrunt run --all -- apply         # applies app-sg, then data, then app

# One unit at a time
cd app && terragrunt plan && terragrunt apply

# Values for .env (AWS_PLAN.md §4.3)
cd ../data && terragrunt output -raw db_address
terragrunt output -raw db_secret_arn  # DB_SECRET_ARN for the API
cd ../app && terragrunt output -raw app_url
```

`run --all -- apply` asks for confirmation once for the whole stack. Plan with
`run --all` shows **mock values** for dependencies that are not applied yet
(for example `sg-00000000000000000`); apply never uses mocks.

## Current deployment

Applied 2026-11-20 from branch `infra/us-east-2-apply`: 20 resources, and a
follow-up `terragrunt run --all -- plan` reports no changes.

| Unit | Created |
| --- | --- |
| `app-sg` | `cmpe180b-team-project-app-sg`: in 80 and 443 from anywhere; out 443/80 to anywhere and 5432 to the VPC |
| `data` | RDS `cmpe180b-team-project-db` (PostgreSQL 18.3, `db.t4g.micro`, 20 GB gp3, **encrypted**, private, database `appdb`, user `appadmin`); security group allowing 5432 only from the app security group; secret `cmpe180b-team-project/db-credentials`; role and instance profile `cmpe180b-team-project-backend-*` |
| `app` | EC2 `t3.small` (Amazon Linux 2023, Docker 25, Compose v5.8.0, Buildx v0.39.1) with an Elastic IP; Session Manager enabled, SSH closed |

Values for `.env` and the deploy (`terragrunt output` in each unit):
`db_address`, `db_secret_arn`, `app_url`, `instance_id`.

Checked after apply: the instance passes status checks and is online in
Session Manager; its role can read the database secret and nothing else
(`ListSecrets` is denied).

2026-11-25: inbound 443 added to `app-sg` for HTTPS (Caddy on the instance),
and the app deployed: https://3-136-227-184.sslip.io
([`docs/evidence/deployment.md`](../docs/evidence/deployment.md)).

There was no Terraform state to migrate. The earlier `infra/*.tf` had no
backend block, and nothing they define existed in the account. The database
the team has used so far, **`book-metaverse-database`**, was created in the
console (user `postgres`, gp2, the VPC's default security group). Terraform
does not manage it and did not change it. Two things the team should decide:

- Its default security group allows port 5432 from `0.0.0.0/0`, and the
  instance is publicly accessible. Restrict the rule to known IPs, or stop the
  instance once nobody needs it.
- Data in it is not in the new database. The app's data comes from migrations
  and seeds, so a fresh deploy needs no copy; move anything else with
  `pg_dump`/`pg_restore` if it matters.

## Secrets and IAM

- **`aws_secretsmanager_secret.db`** stores `{engine, host, port, dbname, username, password}`
  as one JSON secret; the generated password exists only there and in the state.
- **`aws_iam_role.backend`** + **`aws_iam_instance_profile.backend`** may read exactly
  that secret (`secretsmanager:GetSecretValue`). The `app` unit attaches the
  instance profile to the EC2 instance and, with `enable_ssm = true`, adds
  `AmazonSSMManagedInstanceCore` to the same role for Session Manager.
- The API reads the secret at startup when `DB_SECRET_ARN` is set
  (`backend/README.md`); otherwise it uses `DB_*` from `.env`. Never pass database
  credentials to the frontend.
- State contains the database password in plain text. The state bucket is
  encrypted and private; never commit `*.tfstate`.

## App instance

- Amazon Linux 2023 (latest at creation; later AMI releases are ignored so the
  instance is not replaced), `t3.small`, 30 GB encrypted gp3 root volume.
- IMDSv2 required, hop limit 2 so the API container can use the instance role.
- First boot installs Docker, git, and the Compose v5.8.0 and Buildx v0.39.1
  plugins (pinned, SHA-256 verified). Deploying the app is manual (AWS_PLAN.md §4.3).
- Access: Session Manager (`aws ssm start-session --profile cmpe180b --target "$(terragrunt output -raw instance_id)"`,
  which needs the Session Manager plugin for the AWS CLI),
  or SSH when `key_name` and `ssh_cidr_blocks` are set. Port 8080 is never opened.

## Known findings (Trivy)

`trivy config modules` reports, for the team's RDS in `modules/data`:
backup retention, IAM database authentication, and deletion protection
(MEDIUM). These are the optional hardening items in `AWS_PLAN.md` §1.3.
Storage encryption is on (`storage_encrypted = true`); it was set before the
first apply, so it replaced nothing. Changing it later would replace the
database. The app security group's outbound
HTTPS/HTTP to `0.0.0.0/0` is also flagged; the instance needs it for packages,
images, Open Library, and AWS APIs (there is no NAT or VPC endpoint in this
course setup).

## Cost and teardown

Stop the instance between sessions; RDS can be stopped for up to 7 days. The
Elastic IP is billed hourly. After grading, take a manual RDS snapshot if the
data matters (`skip_final_snapshot = true`), then
`terragrunt run --all -- destroy` (app, then data, then app-sg). Empty and
delete the state bucket last.

## Variables worth customizing

- `modules/data/variables.tf`: `db_engine`, `db_engine_version` (PostgreSQL 18),
  `db_instance_class` (`db.t4g.micro`), `db_allocated_storage_gb` (20). Changing the
  engine, `db_name`, or `project_name` on an existing database replaces it.
- `live/dev/app/terragrunt.hcl`: `instance_type`, `root_volume_gb`, `key_name`,
  `create_eip`, `enable_ssm`.
- `live/dev/app-sg/terragrunt.hcl`: `ssh_cidr_blocks` (never `0.0.0.0/0`).
