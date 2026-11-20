# One EC2 instance running Docker Compose (AWS_PLAN.md §3): the Next.js web
# container on port 80 and the Go API container behind it, talking to private RDS.

data "aws_vpc" "default" {
  default = true
}

# Default-VPC subnets in AZs that offer the instance type, so the launch does
# not fail in an AZ without it.
data "aws_ec2_instance_type_offerings" "available" {
  filter {
    name   = "instance-type"
    values = [var.instance_type]
  }
  location_type = "availability-zone"
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
  filter {
    name   = "default-for-az"
    values = ["true"]
  }
  filter {
    name   = "availability-zone"
    values = data.aws_ec2_instance_type_offerings.available.locations
  }
}

# Latest Amazon Linux 2023 (x86_64) from the public SSM parameter. ignore_changes
# below keeps a new AMI release from replacing a running instance.
data "aws_ssm_parameter" "al2023" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}

resource "aws_instance" "app" {
  ami                    = data.aws_ssm_parameter.al2023.value
  instance_type          = var.instance_type
  subnet_id              = sort(data.aws_subnets.default.ids)[0]
  vpc_security_group_ids = [var.app_security_group_id]
  iam_instance_profile   = var.instance_profile_name
  key_name               = var.key_name

  # IMDSv2 only. Hop limit 2 lets the API container reach the instance role
  # credentials (to read the database secret) through the Docker bridge.
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  root_block_device {
    volume_type           = "gp3"
    volume_size           = var.root_volume_gb
    encrypted             = true
    delete_on_termination = true
  }

  user_data                   = file("${path.module}/user_data.sh")
  user_data_replace_on_change = true

  tags = {
    Name    = "${var.project_name}-app"
    Project = var.project_name
  }

  lifecycle {
    ignore_changes = [ami]
  }
}

# Keeps the public address across stop/start. Public IPv4 addresses are billed
# hourly; release it (terraform destroy) when the project ends.
resource "aws_eip" "app" {
  count    = var.create_eip ? 1 : 0
  domain   = "vpc"
  instance = aws_instance.app.id

  tags = {
    Name    = "${var.project_name}-app"
    Project = var.project_name
  }
}

# Shell access through AWS Systems Manager Session Manager, so port 22 can stay
# closed. Attached to the backend role created by the data unit.
resource "aws_iam_role_policy_attachment" "ssm" {
  count      = var.enable_ssm ? 1 : 0
  role       = var.instance_role_name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}
