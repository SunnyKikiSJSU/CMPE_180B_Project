# Settings for the dev (course demo) environment.
locals {
  environment = "dev"
  # The course organization's service control policy denies every region
  # except us-east-2 (checked 2026-11-20).
  aws_region = "us-east-2"
  # Prefixes every resource name. It must stay the same as when the data tier
  # was first applied, or Terraform would replace the database.
  project_name = "cmpe180b-team-project"
}
