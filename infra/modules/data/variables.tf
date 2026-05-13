variable "name_prefix" { type = string }
variable "project" { type = string }
variable "env" { type = string }
variable "vpc_id" { type = string }
variable "data_subnet_ids" { type = list(string) }
variable "app_security_group_id" { type = string }
variable "db_username" {
  type    = string
  default = "throne"
}
variable "instance_class" {
  type    = string
  default = "db.t4g.small"
}
variable "tags" {
  type    = map(string)
  default = {}
}
