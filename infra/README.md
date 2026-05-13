# Infrastructure

Terraform modules and environment compositions for the Throne ingest PoC.

- `modules/network`: VPC, subnets, endpoints, and baseline security groups.
- `modules/storage`: S3 buckets encrypted with KMS per data class.
- `modules/queue`: SQS queue plus DLQ and redrive policy.
- `modules/data`: RDS Postgres, subnet/parameter groups, and SSM password.
- `modules/service`: reusable ECS/Fargate service scaffold.
- `modules/observability`: CloudWatch alarms and Grafana provisioning artifacts.
- `envs/dev`: single sandbox/dev composition.

Run from `infra/envs/dev`:

```sh
terraform init -backend=false
terraform validate
terraform plan -var='project=throne' -var='env=dev'
```
