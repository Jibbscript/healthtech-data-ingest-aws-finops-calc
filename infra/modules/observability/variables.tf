variable "name_prefix" { type = string }
variable "project" { type = string }
variable "env" { type = string }
variable "queue_name" { type = string }
variable "dlq_name" { type = string }
variable "rds_instance_id" { type = string }
variable "alb_arn_suffix" {
  type    = string
  default = null
}
variable "log_group_names" {
  type    = map(string)
  default = {}
}
variable "alert_sns_topic_arn" {
  type    = string
  default = null
}
variable "postgres_max_connections" {
  type    = number
  default = 100
}
variable "tags" {
  type    = map(string)
  default = {}
}
