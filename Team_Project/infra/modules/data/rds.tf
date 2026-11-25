resource "aws_db_instance" "this" {
  identifier     = "${var.project_name}-db"
  engine         = var.db_engine
  engine_version = var.db_engine_version
  instance_class = var.db_instance_class

  allocated_storage = var.db_allocated_storage_gb
  storage_type      = "gp3"
  storage_encrypted = true

  db_name  = var.db_name
  username = var.db_username
  password = local.db_password

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.db.id]
  publicly_accessible    = var.publicly_accessible
  multi_az               = var.multi_az

  # Daily automated backups with point-in-time restore. Backup storage up to
  # the allocated storage is free.
  backup_retention_period = var.backup_retention_days
  copy_tags_to_snapshot   = true

  skip_final_snapshot = var.skip_final_snapshot
  deletion_protection = var.deletion_protection
  apply_immediately   = var.apply_immediately

  tags = {
    Project = var.project_name
  }
}
