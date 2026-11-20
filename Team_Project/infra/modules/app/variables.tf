variable "project_name" {
  description = "Short name used to prefix/tag all resources."
  type        = string
}

variable "instance_type" {
  description = "EC2 instance type. t3.small (2 GB) because building Next.js on 1 GB tends to run out of memory."
  type        = string
  default     = "t3.small"
}

variable "root_volume_gb" {
  description = "Root volume size in GB (gp3, encrypted)."
  type        = number
  default     = 30
}

variable "key_name" {
  description = "Existing EC2 key pair for SSH, or null to rely on Session Manager only."
  type        = string
  default     = null
}

variable "create_eip" {
  description = "Allocate an Elastic IP so the address survives stop/start."
  type        = bool
  default     = true
}

variable "enable_ssm" {
  description = "Attach AmazonSSMManagedInstanceCore to the backend role so Session Manager works without SSH."
  type        = bool
  default     = true
}

variable "app_security_group_id" {
  description = "Security group from the app-sg unit."
  type        = string
}

variable "instance_profile_name" {
  description = "Backend instance profile from the data unit (reads the database secret)."
  type        = string
}

variable "instance_role_name" {
  description = "Backend IAM role from the data unit (Session Manager policy is attached to it)."
  type        = string
}
