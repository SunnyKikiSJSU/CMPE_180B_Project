# Shared Terragrunt settings for every unit under live/.
#
# Remote state: one S3 object per unit, at <path of the unit>/terraform.tfstate,
# with S3-native locking (use_lockfile, Terraform >= 1.10); no DynamoDB table.
# Set TG_STATE_BUCKET to the state bucket's name. To create it the first time,
# run with --backend-bootstrap (versioned, encrypted, public access blocked).

locals {
  env = read_terragrunt_config(find_in_parent_folders("env.hcl")).locals
}

# The team uses Terraform (the lock files list registry.terraform.io
# providers). Terragrunt 1.x runs OpenTofu by default and ignores a
# terraform_binary setting here, so set TG_TF_PATH=terraform (infra/README.md).

remote_state {
  backend = "s3"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = {
    bucket       = get_env("TG_STATE_BUCKET")
    key          = "${path_relative_to_include()}/terraform.tfstate"
    region       = local.env.aws_region
    encrypt      = true
    use_lockfile = true
  }
}

generate "provider" {
  path      = "provider.tf"
  if_exists = "overwrite_terragrunt"
  contents  = <<-EOT
    provider "aws" {
      region = "${local.env.aws_region}"
    }
  EOT
}

inputs = {
  project_name = local.env.project_name
}
