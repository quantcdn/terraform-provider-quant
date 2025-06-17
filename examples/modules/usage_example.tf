# Example: Using the quant-credentials module to fetch tokens for GitHub Actions

terraform {
  required_providers {
    quant = {
      source = "quantcdn/quant"
    }
    github = {
      source = "integrations/github"
    }
  }
}

# Configure providers
provider "quant" {
  bearer       = var.quant_api_token
  organization = var.quant_organization
}

provider "github" {
  token = var.github_token
}

# Local variable to control Quant integration
locals {
  include_quant = var.enable_quant_integration
}

# Use the quant-credentials module to fetch project token
module "quant_credentials" {
  source = "./quant-credentials"
  count  = local.include_quant ? 1 : 0

  project = var.quant_project
}

# Set GitHub Actions secret with the Quant token
resource "github_actions_secret" "quant_token" {
  count = local.include_quant ? 1 : 0

  repository      = var.repository
  secret_name     = "QUANT_TOKEN"
  plaintext_value = module.quant_credentials[0].token
}

# Additional example: Set multiple secrets with project details
resource "github_actions_secret" "quant_project_id" {
  count = local.include_quant ? 1 : 0

  repository      = var.repository
  secret_name     = "QUANT_PROJECT_ID"
  plaintext_value = module.quant_credentials[0].project_details.uuid
}

# Variables
variable "quant_api_token" {
  description = "Quant API bearer token"
  type        = string
  sensitive   = true
}

variable "quant_organization" {
  description = "Quant organization machine name"
  type        = string
}

variable "quant_project" {
  description = "Quant project machine name"
  type        = string
  default     = "default"
}

variable "github_token" {
  description = "GitHub token for managing secrets"
  type        = string
  sensitive   = true
}

variable "repository" {
  description = "GitHub repository name"
  type        = string
}

variable "enable_quant_integration" {
  description = "Whether to enable Quant integration"
  type        = bool
  default     = true
}

# Outputs
output "quant_project_info" {
  description = "Information about the Quant project"
  value = local.include_quant ? {
    name         = module.quant_credentials[0].project_details.name
    machine_name = module.quant_credentials[0].project_details.machine_name
    region       = module.quant_credentials[0].project_details.region
    uuid         = module.quant_credentials[0].project_details.uuid
  } : null
} 