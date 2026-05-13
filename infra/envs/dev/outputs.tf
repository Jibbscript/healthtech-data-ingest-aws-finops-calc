output "vpc_id" { value = module.network.vpc_id }
output "raw_bucket_name" { value = module.raw_storage.bucket_name }
output "derived_bucket_name" { value = module.derived_storage.bucket_name }
output "ingest_queue_url" { value = module.ingest_queue.queue_url }
output "database_endpoint" { value = module.database.endpoint }
output "service_arns" { value = { for name, svc in module.service : name => svc.service_arn } }
output "alarm_names" { value = module.observability.alarm_names }
