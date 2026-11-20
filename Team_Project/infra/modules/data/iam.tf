# Least-privilege IAM for the backend tier: read-only access to exactly this
# one secret, nothing else. Shaped as an EC2 instance profile since the
# backend's compute target isn't decided yet — if it ends up on ECS/Lambda
# instead, swap the assume-role principal and reuse aws_iam_policy.read_db_secret.

data "aws_iam_policy_document" "ec2_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "backend" {
  name               = "${var.project_name}-backend-role"
  assume_role_policy = data.aws_iam_policy_document.ec2_assume_role.json

  tags = {
    Project = var.project_name
  }
}

data "aws_iam_policy_document" "read_db_secret" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.db.arn]
  }
}

resource "aws_iam_policy" "read_db_secret" {
  name   = "${var.project_name}-read-db-secret"
  policy = data.aws_iam_policy_document.read_db_secret.json
}

resource "aws_iam_role_policy_attachment" "backend_read_db_secret" {
  role       = aws_iam_role.backend.name
  policy_arn = aws_iam_policy.read_db_secret.arn
}

resource "aws_iam_instance_profile" "backend" {
  name = "${var.project_name}-backend-profile"
  role = aws_iam_role.backend.name
}
