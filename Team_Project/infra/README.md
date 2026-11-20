# Infra (Terraform + Terragrunt)

AWS for the Book Metaverse Portal in `us-west-2`: the RDS data tier written by
the team, plus the EC2 app tier from [`AWS_PLAN.md`](../AWS_PLAN.md) §4.2.
Terraform modules hold the resources; Terragrunt supplies remote state, the
provider, and the wiring between units.

## Layout

```text
infra/
├── root.hcl                    # S3 remote state (one key per unit, S3 lock file), aws provider, shared inputs
├── modules/
│   ├── data/                   # The team's RDS tier, unchanged: RDS PostgreSQL 18, DB security group,
│   │                           # random password, Secrets Manager secret, backend IAM role + instance profile
│   ├── app_security_group/     # App security group: in HTTP 80 (+ SSH from listed CIDRs); out 443, 80, 5432 to the VPC
│   └── app/                    # EC2 (Amazon Linux 2023, Docker + Compose), optional Elastic IP, Session Manager
└── live/
    └── dev/
        ├── env.hcl             # environment, region, project_name
        ├── app-sg/terragrunt.hcl
        ├── data/terragrunt.hcl # + .terraform.lock.hcl (aws 5.100.0, random 3.9.1, as originally applied)
        └── app/terragrunt.hcl
```

Order (Terragrunt resolves it from the `dependency` blocks): **app-sg → data → app**.
The app security group is its own unit so the database security group can allow
it without a cycle (the app instance needs the data tier's instance profile).

| Unit | Creates | Uses |
| --- | --- | --- |
| `app-sg` | `aws_security_group.app` and its rules | — |
| `data` | Everything that was in `infra/*.tf` before | `app-sg` security group as `backend_security_group_id`, so RDS accepts only the app instance |
| `app` | `aws_instance.app`, `aws_eip.app`, Session Manager policy on the backend role | `app-sg` security group; `data` instance profile and role |

## Requirements

- Terraform >= 1.10 (S3 native locking) and Terragrunt 1.x. `root.hcl` sets
  `terraform_binary = "terraform"`; Terragrunt would otherwise run OpenTofu.
- AWS credentials for the **course account only**, for example
  `export AWS_PROFILE=<course-profile>`. Check before every run:
  `aws sts get-caller-identity`.
- `export TG_STATE_BUCKET=<globally unique bucket name>` for the state bucket.

## Usage

```bash
cd Team_Project/infra/live/dev
export AWS_PROFILE=<course-profile> TG_STATE_BUCKET=<state-bucket>

# First time only: create the state bucket (versioned, encrypted, public access blocked).
terragrunt run --all --backend-bootstrap -- init

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

## Moving the existing state

The data tier was applied before this layout existed. The `data` unit uses the
same module code with the same resource addresses (`aws_db_instance.this`,
`aws_secretsmanager_secret.db`, ...), so only the **state** has to move. Done
correctly, AWS sees no change to the database, secret, or IAM role.

1. Freeze changes: nobody runs Terraform in the old `infra/` directory from now on.
2. Export the current state with the commit **before** this change (where the
   `.tf` files were directly in `infra/`), using whatever backend it was applied
   with:

   ```bash
   git checkout <commit before the Terragrunt change> -- Team_Project/infra
   cd Team_Project/infra && terraform init && terraform state pull > /tmp/data.tfstate
   terraform state list                     # note the resources
   cd - && git checkout HEAD -- Team_Project/infra
   ```

   Also note any variable values it was applied with (a local `*.tfvars` or
   `-var` flags, for example `allowed_cidr_blocks`). Put them in
   `live/dev/data/terragrunt.hcl` `inputs`, or the plan will try to change them.
3. Create the app security group (new resources only):

   ```bash
   cd Team_Project/infra/live/dev/app-sg
   terragrunt run --backend-bootstrap -- init && terragrunt apply
   ```

4. Push the state into the `data` unit and check it:

   ```bash
   cd ../data
   terragrunt init
   terragrunt state list                    # must be empty before the push
   terragrunt state push /tmp/data.tfstate
   terragrunt state list                    # same resources as in step 2
   terragrunt plan
   ```

   The plan must show **no destroy and no replace**. Expected: `1 to add`
   (`aws_security_group_rule.from_backend[0]`, RDS now allows the app security
   group) and possibly in-place changes you can explain. If anything would be
   replaced, stop: do not apply, and compare the inputs with step 2.
5. `terragrunt apply` in `data`, then `cd ../app && terragrunt apply`.
6. Keep `/tmp/data.tfstate` until the app works, then delete it: it contains
   the database password in plain text.

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
- Access: Session Manager (`aws ssm start-session --target "$(terragrunt output -raw instance_id)"`),
  or SSH when `key_name` and `ssh_cidr_blocks` are set. Port 8080 is never opened.

## Known findings (Trivy)

`trivy config modules` reports, for the team's RDS in `modules/data`:
storage not encrypted (HIGH), backup retention, IAM database authentication,
and deletion protection (MEDIUM). These are the optional hardening items in
`AWS_PLAN.md` §1.3. **`storage_encrypted = true` replaces an existing database**,
so decide before relying on its data. The app security group's outbound
HTTPS/HTTP to `0.0.0.0/0` is also flagged; the instance needs it for packages,
images, Google Books, and AWS APIs (there is no NAT or VPC endpoint in this
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
