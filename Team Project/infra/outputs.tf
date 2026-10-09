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
