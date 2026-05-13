variable "project" {
  description = "Project tag and resource name prefix."
  type        = string
  default     = "throne"
}

variable "env" {
  description = "Environment name."
  type        = string
  default     = "dev"
}

variable "aws_region" {
  description = "AWS region for the dev sandbox."
  type        = string
  default     = "us-east-1"
}

variable "name_prefix" {
  description = "Optional resource name prefix."
  type        = string
  default     = "throne-dev"
}

variable "availability_zones" {
  description = "AZs to spread subnets across."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c"]
}

variable "db_username" {
  description = "RDS master username."
  type        = string
  default     = "throne"
}
