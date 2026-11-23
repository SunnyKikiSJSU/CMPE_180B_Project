# Data tier written by the team: RDS PostgreSQL 18, its security group, the
# Secrets Manager secret, and the backend IAM role and instance profile.
#
# The module keeps the original resource addresses (aws_db_instance.this, ...),
# so moving the existing state here changes nothing in AWS. See infra/README.md
# "Moving the existing state".
include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_parent_terragrunt_dir("root")}/modules/data"
}

dependency "app_sg" {
  config_path = "../app-sg"

  # Only so `plan` and `validate` work before app-sg exists; never for apply.
  mock_outputs                            = { security_group_id = "sg-00000000000000000" }
  mock_outputs_allowed_terraform_commands = ["init", "validate", "plan"]
}

inputs = {
  # RDS accepts connections only from the app instance's security group
  # (AWS_PLAN.md §4.2).
  backend_security_group_id = dependency.app_sg.outputs.security_group_id
  # Temporary direct access for migrations from a laptop, for example
  # ["203.0.113.7/32"]. Keep empty otherwise.
  allowed_cidr_blocks = []
}
