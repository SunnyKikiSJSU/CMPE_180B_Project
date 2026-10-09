# Auto-generates the master password when var.db_password is left null, so
# nobody has to hand-pick or type a secret to stand this environment up.
resource "random_password" "db" {
  count   = var.db_password == null ? 1 : 0
  length  = 24
  special = false
}

locals {
  db_password = var.db_password != null ? var.db_password : random_password.db[0].result
}
