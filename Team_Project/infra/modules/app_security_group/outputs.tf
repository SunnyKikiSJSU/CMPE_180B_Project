output "security_group_id" {
  description = "Attached to the app instance; the database security group allows it on the database port."
  value       = aws_security_group.app.id
}
