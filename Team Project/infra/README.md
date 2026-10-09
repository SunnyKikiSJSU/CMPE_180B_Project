# Infra (Terraform) — RDS Data Tier

Provisions the Amazon RDS instance for the team project's data tier.

## Usage

```bash
cd "Team Project/infra"
terraform init
terraform plan   -var="db_password=<pick-a-strong-password>"
terraform apply  -var="db_password=<pick-a-strong-password>"
```

Or, to avoid typing the password every time, create a gitignored
`secrets.auto.tfvars` file next to these files:

```hcl
db_password = "<pick-a-strong-password>"
```

Terraform loads `*.auto.tfvars` automatically. **Never commit this file** —
it's already covered by `.gitignore` here, along with `.terraform/` and
`*.tfstate*` (the state file contains the password in plaintext).

## What this creates

- An `aws_db_instance` (RDS) using the account's **default VPC** and its
  subnets — no custom VPC needed for the class project.
- A dedicated security group for the database that only allows inbound
  traffic on the DB port from either:
  - the backend tier's security group (`backend_security_group_id`), or
  - specific CIDR blocks (`allowed_cidr_blocks`) — use this only for local
    development convenience, not for the real deployed backend.

## Wiring up the backend

1. Deploy this (`terraform apply`), note the `db_endpoint` output.
2. Give the backend's security group ID to `backend_security_group_id` (or
   add your dev IP's CIDR to `allowed_cidr_blocks` for local testing) and
   `terraform apply` again.
3. The backend is the **only** tier that should hold the DB
   endpoint/username/password — never pass these to the frontend.

## Variables worth customizing

See [`variables.tf`](variables.tf) — in particular `db_engine` (`mysql` or
`postgres`), `db_instance_class`, and `db_allocated_storage_gb`. Defaults are
sized for Free Tier eligibility (`db.t4g.micro`, 20 GB, single-AZ).
