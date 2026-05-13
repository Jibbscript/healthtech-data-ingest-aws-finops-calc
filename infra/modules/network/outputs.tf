output "vpc_id" { value = aws_vpc.this.id }
output "public_subnet_ids" { value = aws_subnet.public[*].id }
output "private_subnet_ids" { value = aws_subnet.private[*].id }
output "data_subnet_ids" { value = aws_subnet.data[*].id }
output "security_group_ids" { value = { alb = aws_security_group.alb.id, tasks = aws_security_group.tasks.id, data = aws_security_group.data.id, endpoints = aws_security_group.endpoints.id } }
