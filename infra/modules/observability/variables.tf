variable "name_prefix" { type = string }
variable "queue_name" { type = string }
variable "dlq_name" { type = string }
variable "rds_instance_id" { type = string }
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
