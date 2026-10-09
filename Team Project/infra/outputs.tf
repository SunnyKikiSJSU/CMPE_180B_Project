output "db_endpoint" {
  description = "Host:port the backend should connect to."
  value       = aws_db_instance.this.endpoint
}

output "db_address" {
  description = "Hostname only (no port)."
  value       = aws_db_instance.this.address
}

output "db_port" {
  value = aws_db_instance.this.port
}

output "db_name" {
  value = aws_db_instance.this.db_name
}

output "db_security_group_id" {
  description = "Pass this as backend_security_group_id's source, or attach the backend's own SG to backend_security_group_id so it's granted ingress."
  value       = aws_security_group.db.id
}

output "db_secret_arn" {
  description = "ARN of the Secrets Manager secret holding DB host/port/username/password. The backend reads this at runtime via its IAM role — never pass these values as plain env vars."
  value       = aws_secretsmanager_secret.db.arn
}

output "backend_iam_role_name" {
  description = "IAM role name to attach to the backend's compute (EC2 instance profile already created as backend_instance_profile_name)."
  value       = aws_iam_role.backend.name
}

output "backend_instance_profile_name" {
  description = "Attach this instance profile to the backend's EC2 instance(s) so it can read db_secret_arn without static AWS credentials."
  value       = aws_iam_instance_profile.backend.name
}
