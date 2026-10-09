# Stores the DB master credentials + connection info so the backend never
# needs the password baked into an env var, image, or committed file — it
# fetches this at runtime via its IAM role instead.
resource "aws_secretsmanager_secret" "db" {
  name        = "${var.project_name}/db-credentials"
  description = "RDS master credentials and connection info for ${var.project_name}."

  tags = {
    Project = var.project_name
  }
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id
  secret_string = jsonencode({
    engine   = var.db_engine
    host     = aws_db_instance.this.address
    port     = aws_db_instance.this.port
    dbname   = aws_db_instance.this.db_name
    username = var.db_username
    password = local.db_password
  })
}
