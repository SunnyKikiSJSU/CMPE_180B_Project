variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-west-2"
}

variable "project_name" {
  description = "Short name used to prefix/tag all resources."
  type        = string
  default     = "cmpe180b-team-project"
}

variable "db_engine" {
  description = "RDS engine (mysql or postgres)."
  type        = string
  default     = "mysql"
}

variable "db_engine_version" {
  description = "Engine version. Leave default unless a specific version is required."
  type        = string
  default     = "8.0"
}

variable "db_instance_class" {
  description = "RDS instance class. db.t3.micro/db.t4g.micro are Free Tier eligible."
  type        = string
  default     = "db.t4g.micro"
}

variable "db_allocated_storage_gb" {
  description = "Allocated storage in GB."
  type        = number
  default     = 20
}

variable "db_name" {
  description = "Initial database name created on the instance."
  type        = string
  default     = "appdb"
}

variable "db_username" {
  description = "Master username for the RDS instance."
  type        = string
  default     = "appadmin"
}

variable "db_password" {
  description = "Master password for the RDS instance. Leave null (default) to auto-generate a random password stored only in Secrets Manager. If set, pass via a gitignored *.auto.tfvars file or TF_VAR_db_password env var — never commit it."
  type        = string
  sensitive   = true
  default     = null
}

variable "backend_security_group_id" {
  description = "Security group ID of the backend tier, granted ingress to the database on the DB port. Leave null to instead allow ingress from allowed_cidr_blocks (e.g. for local development)."
  type        = string
  default     = null
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to reach the database directly (use only for local/dev access; prefer backend_security_group_id for the real backend tier)."
  type        = list(string)
  default     = []
}

variable "publicly_accessible" {
  description = "Whether the RDS instance gets a public IP. Keep false except for short-lived local development convenience."
  type        = bool
  default     = false
}

variable "multi_az" {
  description = "Whether to deploy a standby replica in a second AZ for high availability."
  type        = bool
  default     = false
}

variable "skip_final_snapshot" {
  description = "Skip the final snapshot on destroy. Fine for a class project; set false for anything with real data."
  type        = bool
  default     = true
}
