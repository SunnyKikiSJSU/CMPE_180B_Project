# Security group for the app instance. It is its own unit so the database
# security group (data unit) can allow it without a dependency cycle:
# app-sg -> data -> app.

data "aws_vpc" "default" {
  default = true
}

resource "aws_security_group" "app" {
  name        = "${var.project_name}-app-sg"
  description = "Web traffic to the app instance; SSH only from listed CIDRs."
  vpc_id      = data.aws_vpc.default.id

  tags = {
    Project = var.project_name
  }
}

# Plain HTTP for the course demo (AWS_PLAN.md §2). The API port 8080 stays
# closed: Next.js forwards /api/* to the API over the Compose network.
resource "aws_vpc_security_group_ingress_rule" "http" {
  security_group_id = aws_security_group.app.id
  description       = "HTTP"
  ip_protocol       = "tcp"
  from_port         = 80
  to_port           = 80
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "ssh" {
  for_each = toset(var.ssh_cidr_blocks)

  security_group_id = aws_security_group.app.id
  description       = "SSH"
  ip_protocol       = "tcp"
  from_port         = 22
  to_port           = 22
  cidr_ipv4         = each.value
}

# Outbound only what the instance uses: HTTPS and HTTP (packages, container
# images, Go and npm modules, Google Books, AWS APIs) and PostgreSQL inside the
# VPC (RDS). DNS, NTP, and instance metadata use link-local addresses that
# security groups do not filter.
resource "aws_vpc_security_group_egress_rule" "https" {
  security_group_id = aws_security_group.app.id
  description       = "HTTPS"
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "http" {
  security_group_id = aws_security_group.app.id
  description       = "HTTP (package mirrors)"
  ip_protocol       = "tcp"
  from_port         = 80
  to_port           = 80
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "postgres" {
  security_group_id = aws_security_group.app.id
  description       = "PostgreSQL to RDS in the VPC"
  ip_protocol       = "tcp"
  from_port         = 5432
  to_port           = 5432
  cidr_ipv4         = data.aws_vpc.default.cidr_block
}
