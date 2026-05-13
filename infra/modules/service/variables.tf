variable "name" { type = string }
variable "project" { type = string }
variable "env" { type = string }
variable "cluster_name" { type = string }
variable "vpc_id" { type = string }
variable "subnet_ids" { type = list(string) }
variable "security_group_ids" { type = list(string) }
variable "image" { type = string }
variable "cpu" { type = number }
variable "memory" { type = number }
variable "container_port" { type = number }
variable "desired_count" {
  type    = number
  default = 1
}
variable "autoscaling_min" {
  type    = number
  default = 1
}
variable "autoscaling_max" {
  type    = number
  default = 3
}
variable "environment" {
  type    = map(string)
  default = {}
}
variable "secrets" {
  type    = map(string)
  default = {}
}
variable "target_group_arn" {
  type    = string
  default = null
}
variable "assign_public_ip" {
  type    = bool
  default = false
}
variable "tags" {
  type    = map(string)
  default = {}
}
