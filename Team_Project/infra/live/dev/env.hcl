# Settings for the dev (course demo) environment.
locals {
  environment = "dev"
  aws_region  = "us-west-2"
  # Prefixes every resource name. It must stay the same as when the data tier
  # was first applied, or Terraform would replace the database.
  project_name = "cmpe180b-team-project"
}
