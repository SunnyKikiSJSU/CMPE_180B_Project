# Security group for the app instance (applied first; the database allows it).
include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_parent_terragrunt_dir("root")}/modules/app_security_group"
}

inputs = {
  # Your public IP as /32 to allow SSH; leave empty and use Session Manager.
  ssh_cidr_blocks = []
}
