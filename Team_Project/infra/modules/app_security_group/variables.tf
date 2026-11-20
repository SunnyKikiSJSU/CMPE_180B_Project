variable "project_name" {
  description = "Short name used to prefix/tag all resources."
  type        = string
}

variable "ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH to the instance (for example [\"203.0.113.7/32\"]). Empty means no SSH port is open; use Session Manager instead."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for cidr in var.ssh_cidr_blocks : can(cidrhost(cidr, 0)) && cidr != "0.0.0.0/0"])
    error_message = "ssh_cidr_blocks must be valid IPv4 CIDRs and must not include 0.0.0.0/0."
  }
}
