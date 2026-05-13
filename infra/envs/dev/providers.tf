terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.86"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Phase-2 target backend. Keep commented for local validation/bootstrap-less PoC runs.
  # backend "s3" {
  #   bucket         = "throne-tfstate-dev"
  #   key            = "healthtech-data-ingest-finops-calc/dev.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "throne-tfstate-locks-dev"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = local.tags
  }
}
