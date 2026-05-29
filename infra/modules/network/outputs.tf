output "vpc_id" { value = aws_vpc.this.id }
output "public_subnet_ids" { value = aws_subnet.public[*].id }
output "private_subnet_ids" { value = aws_subnet.private[*].id }
output "data_subnet_ids" { value = aws_subnet.data[*].id }
output "security_group_ids" { value = { tasks = aws_security_group.tasks.id, endpoints = aws_security_group.endpoints.id } }
