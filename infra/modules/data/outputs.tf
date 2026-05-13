output "db_instance_id" { value = aws_db_instance.this.identifier }
output "endpoint" { value = aws_db_instance.this.endpoint }
output "password_parameter_name" { value = aws_ssm_parameter.password.name }
output "security_group_id" { value = aws_security_group.db.id }
