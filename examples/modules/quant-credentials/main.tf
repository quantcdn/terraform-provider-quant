# Module: quant-credentials
# This module fetches Quant project credentials for use in other providers

terraform {
  required_providers {
    quant = {
      source = "quantcdn/quant"
    }
  }
}

# Fetch project details including write token
data "quant_project" "project" {
  machine_name = var.project
  with_token   = true
}

# Variables
variable "project" {
  description = "Quant project machine name"
  type        = string
}

# Outputs
output "token" {
  description = "Project write token"
  value       = data.quant_project.project.write_token
  sensitive   = true
}

output "project_details" {
  description = "Full project details"
  value = {
    id           = data.quant_project.project.id
    name         = data.quant_project.project.name
    uuid         = data.quant_project.project.uuid
    machine_name = data.quant_project.project.machine_name
    region       = data.quant_project.project.region
    created_at   = data.quant_project.project.created_at
    updated_at   = data.quant_project.project.updated_at
  }
} 