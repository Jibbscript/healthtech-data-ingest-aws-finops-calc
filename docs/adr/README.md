# Architecture Decision Records

| ADR | Decision |
|---|---|
| [0001](0001-sqs-over-kinesis.md) | SQS standard queues over Kinesis/MSK for v1 async processing |
| [0002](0002-postgres-over-dynamodb.md) | RDS Postgres over DynamoDB for metadata and findings |
| [0003](0003-grpc-primary-rest-gateway.md) | gRPC as canonical API with grpc-gateway REST |
| [0004](0004-fargate-over-eks.md) | ECS/Fargate over EKS for PoC and early production |
| [0005](0005-queue-per-stage.md) | One SQS queue and DLQ per processing stage |
| [0006](0006-kms-per-data-class.md) | KMS key per data class and environment |
| [0007](0007-s3-lifecycle-in-terraform.md) | S3 lifecycle policies live in Terraform |
| [0008](0008-load-generator-first-class.md) | Synthetic load generator is a first-class artifact |
