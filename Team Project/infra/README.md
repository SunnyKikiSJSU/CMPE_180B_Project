# Infra (Terraform) — RDS Data Tier

Provisions the Amazon RDS instance for the team project's data tier.

## Usage

```bash
cd "Team Project/infra"
terraform init
terraform plan
terraform apply
```

No password needs to be supplied — if `db_password` is left unset, a random
24-character password is generated automatically and stored **only** in AWS
Secrets Manager (see `db_secret_arn` output). If you want to pin a specific
password instead, pass `-var="db_password=<...>"` or set it in a gitignored
`secrets.auto.tfvars` file.

`.terraform/` and `*.tfstate*` are gitignored — the state file still contains
the password in plaintext, so never commit it regardless.

## What this creates

- An `aws_db_instance` (RDS) using the account's **default VPC** and its
  subnets — no custom VPC needed for the class project.
- A dedicated security group for the database that only allows inbound
  traffic on the DB port from either:
  - the backend tier's security group (`backend_security_group_id`), or
  - specific CIDR blocks (`allowed_cidr_blocks`) — use this only for local
    development convenience, not for the real deployed backend.

## Secrets & IAM

- **`aws_secretsmanager_secret.db`** stores `{engine, host, port, dbname, username, password}` as a single JSON secret — this is the only place the password lives outside Terraform state.
- **`aws_iam_role.backend`** + **`aws_iam_instance_profile.backend`** grant read-only access to exactly that one secret (`secretsmanager:GetSecretValue`), nothing else. Attach `backend_instance_profile_name` to whatever EC2 instance(s) run the backend.
- If the backend ends up on ECS/Lambda instead of EC2 rather than plain EC2, reuse `aws_iam_policy.read_db_secret` with a different assume-role principal instead of the EC2 instance profile.
- The backend's runtime code should call `secretsmanager:GetSecretValue` on `db_secret_arn` at startup — **no DB credentials should ever be passed as plain environment variables, command-line args, or committed config.**

## Wiring up the backend

1. Deploy this (`terraform apply`).
2. Attach `backend_instance_profile_name` to the backend's EC2 instance(s).
3. Give the backend's security group ID to `backend_security_group_id` (or
   add your dev IP's CIDR to `allowed_cidr_blocks` for local testing) and
   `terraform apply` again.
4. The backend is the **only** tier that should ever read `db_secret_arn` —
   never pass DB credentials to the frontend.

## Variables worth customizing

See [`variables.tf`](variables.tf) — in particular `db_engine` (`mysql` or
`postgres`), `db_instance_class`, and `db_allocated_storage_gb`. Defaults are
sized for Free Tier eligibility (`db.t4g.micro`, 20 GB, single-AZ).
