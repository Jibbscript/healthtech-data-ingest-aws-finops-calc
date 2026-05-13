resource "random_password" "db" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.name_prefix}-postgres"
  subnet_ids = var.data_subnet_ids
  tags       = merge(var.tags, { Name = "${var.name_prefix}-postgres" })
}

resource "aws_security_group" "db" {
  name        = "${var.name_prefix}-sg-postgres"
  description = "Postgres ingress from app tasks"
  vpc_id      = var.vpc_id
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [var.app_security_group_id]
    description     = "Postgres from ECS tasks"
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All egress"
  }
  tags = merge(var.tags, { Name = "${var.name_prefix}-sg-postgres" })
}

resource "aws_db_parameter_group" "this" {
  name   = "${var.name_prefix}-postgres16"
  family = "postgres16"
  parameter {
    name  = "log_statement"
    value = "ddl"
  }
  tags = merge(var.tags, { Name = "${var.name_prefix}-postgres16" })
}

resource "aws_ssm_parameter" "password" {
  name  = "/${var.project}/${var.env}/db/password"
  type  = "SecureString"
  value = random_password.db.result
  tags  = var.tags
}

resource "aws_db_instance" "this" {
  identifier                 = "${var.name_prefix}-postgres"
  engine                     = "postgres"
  engine_version             = "16"
  instance_class             = var.instance_class
  allocated_storage          = 50
  max_allocated_storage      = 200
  db_name                    = "throne"
  username                   = var.db_username
  password                   = random_password.db.result
  db_subnet_group_name       = aws_db_subnet_group.this.name
  vpc_security_group_ids     = [aws_security_group.db.id]
  parameter_group_name       = aws_db_parameter_group.this.name
  backup_retention_period    = 7
  deletion_protection        = false
  skip_final_snapshot        = true
  storage_encrypted          = true
  auto_minor_version_upgrade = true
  publicly_accessible        = false
  tags                       = merge(var.tags, { Name = "${var.name_prefix}-postgres" })
}
