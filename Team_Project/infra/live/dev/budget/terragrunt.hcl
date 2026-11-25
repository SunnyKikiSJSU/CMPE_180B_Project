# Account cost budget: emails when spend passes 80% of the limit or the
# month's forecast passes it. Independent of the other units.
include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${get_parent_terragrunt_dir("root")}/modules/budget"
}

inputs = {
  # About $36 a month when running all the time; $20 catches a forgotten
  # instance or an unexpected charge early.
  monthly_limit_usd = 20
  alert_emails      = ["ahmed.abdelrahman@sjsu.edu"]
}
