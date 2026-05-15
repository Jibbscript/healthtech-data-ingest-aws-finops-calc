locals {
  tags = {
    Project   = var.project
    Env       = var.env
    ManagedBy = "terraform"
  }

  services = {
    ingest    = { image = "public.ecr.aws/docker/library/nginx:stable-alpine", port = 8080, cpu = 256, memory = 512, desired = 1, min = 1, max = 3 }
    processor = { image = "public.ecr.aws/docker/library/busybox:latest", port = 8080, cpu = 1024, memory = 2048, desired = 1, min = 1, max = 10 }
    inference = { image = "public.ecr.aws/docker/library/nginx:stable-alpine", port = 8443, cpu = 2048, memory = 4096, desired = 1, min = 1, max = 5 }
    api       = { image = "public.ecr.aws/docker/library/nginx:stable-alpine", port = 8080, cpu = 256, memory = 512, desired = 1, min = 1, max = 3 }
  }
}

data "aws_caller_identity" "current" {}

module "network" {
  source             = "../../modules/network"
  name_prefix        = var.name_prefix
  project            = var.project
  env                = var.env
  vpc_cidr           = "10.42.0.0/16"
  availability_zones = var.availability_zones
  tags               = local.tags
}

module "raw_storage" {
  source     = "../../modules/storage"
  name       = "${var.name_prefix}-raw-${data.aws_caller_identity.current.account_id}"
  project    = var.project
  env        = var.env
  data_class = "phi"
  tags       = local.tags

  lifecycle_rules = [
    {
      id      = "raw-tiering"
      enabled = true
      transitions = [
        { days = 30, storage_class = "STANDARD_IA" },
        { days = 90, storage_class = "GLACIER_IR" }
      ]
      expiration_days = 0
    }
  ]
}

module "derived_storage" {
  source     = "../../modules/storage"
  name       = "${var.name_prefix}-derived-${data.aws_caller_identity.current.account_id}"
  project    = var.project
  env        = var.env
  data_class = "standard"
  tags       = local.tags

  lifecycle_rules = [
    {
      id              = "derived-retention"
      enabled         = true
      transitions     = []
      expiration_days = 365
    }
  ]
}

module "ingest_queue" {
  source                     = "../../modules/queue"
  name                       = "${var.name_prefix}-ingest-jobs"
  project                    = var.project
  env                        = var.env
  visibility_timeout_seconds = 180
  message_retention_seconds  = 345600
  max_receive_count          = 3
  tags                       = local.tags
}

module "database" {
  source                = "../../modules/data"
  name_prefix           = var.name_prefix
  project               = var.project
  env                   = var.env
  vpc_id                = module.network.vpc_id
  data_subnet_ids       = module.network.data_subnet_ids
  app_security_group_id = module.network.security_group_ids.tasks
  db_username           = var.db_username
  instance_class        = "db.t4g.small"
  tags                  = local.tags
}

module "service" {
  for_each = local.services

  source             = "../../modules/service"
  name               = "${var.name_prefix}-${each.key}"
  project            = var.project
  env                = var.env
  cluster_name       = "${var.name_prefix}-${each.key}-cluster"
  vpc_id             = module.network.vpc_id
  subnet_ids         = module.network.private_subnet_ids
  security_group_ids = [module.network.security_group_ids.tasks]
  image              = each.value.image
  cpu                = each.value.cpu
  memory             = each.value.memory
  container_port     = each.value.port
  desired_count      = each.value.desired
  autoscaling_min    = each.value.min
  autoscaling_max    = each.value.max
  environment = {
    ENV                  = var.env
    RAW_BUCKET           = module.raw_storage.bucket_name
    DERIVED_BUCKET       = module.derived_storage.bucket_name
    INGEST_QUEUE_URL     = module.ingest_queue.queue_url
    DB_PASSWORD_SSM_PATH = module.database.password_parameter_name
  }
  tags = local.tags
}

module "observability" {
  source                   = "../../modules/observability"
  name_prefix              = var.name_prefix
  project                  = var.project
  env                      = var.env
  queue_name               = module.ingest_queue.queue_name
  dlq_name                 = module.ingest_queue.dlq_name
  rds_instance_id          = module.database.db_instance_id
  alb_arn_suffix           = null
  alert_sns_topic_arn      = null
  postgres_max_connections = 100
  tags                     = local.tags
}
