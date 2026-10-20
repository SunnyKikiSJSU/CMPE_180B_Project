# Uses the account's default VPC/subnets so the team doesn't need to stand up
# custom networking just to get a database running for the class project.
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.project_name}-db-subnet-group"
  subnet_ids = data.aws_subnets.default.ids

  tags = {
    Project = var.project_name
  }
}

resource "aws_security_group" "db" {
  name        = "${var.project_name}-db-sg"
  description = "Allows the backend tier (and optionally dev CIDRs) to reach RDS on the database port."
  vpc_id      = data.aws_vpc.default.id

  tags = {
    Project = var.project_name
  }
}

locals {
  db_port = var.db_engine == "postgres" ? 5432 : 3306
}

resource "aws_security_group_rule" "from_backend" {
  count = var.backend_security_group_id != null ? 1 : 0

  type                     = "ingress"
  security_group_id        = aws_security_group.db.id
  from_port                = local.db_port
  to_port                  = local.db_port
  protocol                 = "tcp"
  source_security_group_id = var.backend_security_group_id
}

resource "aws_security_group_rule" "from_allowed_cidrs" {
  count = length(var.allowed_cidr_blocks) > 0 ? 1 : 0

  type              = "ingress"
  security_group_id = aws_security_group.db.id
  from_port         = local.db_port
  to_port           = local.db_port
  protocol          = "tcp"
  cidr_blocks       = var.allowed_cidr_blocks
}

resource "aws_security_group_rule" "egress_all" {
  type              = "egress"
  security_group_id = aws_security_group.db.id
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
}
