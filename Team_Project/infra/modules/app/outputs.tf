output "instance_id" {
  description = "For Session Manager: aws ssm start-session --target <id>."
  value       = aws_instance.app.id
}

output "app_public_ip" {
  description = "Public address of the app (Elastic IP when create_eip is true)."
  value       = var.create_eip ? aws_eip.app[0].public_ip : aws_instance.app.public_ip
}

output "app_url" {
  description = "The site over plain HTTP."
  value       = "http://${var.create_eip ? aws_eip.app[0].public_ip : aws_instance.app.public_ip}/"
}
