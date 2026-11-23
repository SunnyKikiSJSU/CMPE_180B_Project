# EC2 instance that runs the web and API containers with Docker Compose.
include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_parent_terragrunt_dir("root")}/modules/app"
}

dependency "app_sg" {
  config_path = "../app-sg"

  mock_outputs                            = { security_group_id = "sg-00000000000000000" }
  mock_outputs_allowed_terraform_commands = ["init", "validate", "plan"]
}

dependency "data" {
  config_path = "../data"

  mock_outputs = {
    backend_instance_profile_name = "mock-backend-profile"
    backend_iam_role_name         = "mock-backend-role"
  }
  mock_outputs_allowed_terraform_commands = ["init", "validate", "plan"]
}

inputs = {
  app_security_group_id = dependency.app_sg.outputs.security_group_id
  instance_profile_name = dependency.data.outputs.backend_instance_profile_name
  instance_role_name    = dependency.data.outputs.backend_iam_role_name

  instance_type  = "t3.small"
  root_volume_gb = 30
  create_eip     = true
  enable_ssm     = true
  # Name of an existing EC2 key pair for SSH, or null for Session Manager only.
  key_name = null
}
