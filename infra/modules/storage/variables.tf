variable "name" { type = string }
variable "project" { type = string }
variable "env" { type = string }
variable "data_class" {
  type = string
  validation {
    condition     = contains(["phi", "standard", "telemetry"], var.data_class)
    error_message = "data_class must be phi, standard, or telemetry."
  }
}
variable "lifecycle_rules" {
  type = list(object({
    id              = string
    enabled         = bool
    transitions     = list(object({ days = number, storage_class = string }))
    expiration_days = number
  }))
  default = []
}
variable "tags" {
  type    = map(string)
  default = {}
}
