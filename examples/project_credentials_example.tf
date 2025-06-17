# Example: Using quant_project data source to fetch write tokens for other providers

# Configure the Quant provider
provider "quant" {
  bearer = var.quant_api_token
  organization = var.quant_organization
}

# Fetch project details including write token
data "quant_project" "main" {
  machine_name = var.project_machine_name
  with_token   = true  # Default is true for data sources
}

# Example: Use the write token in GitHub Actions secrets
resource "github_actions_secret" "quant_token" {
  count = var.include_quant ? 1 : 0

  repository      = var.repository
  secret_name     = "QUANT_TOKEN"
  plaintext_value = data.quant_project.main.write_token
}

# Example: Use in other resources that need the token
resource "aws_ssm_parameter" "quant_token" {
  count = var.include_quant ? 1 : 0

  name  = "/app/quant/write_token"
  type  = "SecureString"
  value = data.quant_project.main.write_token

  tags = {
    Project = data.quant_project.main.name
    UUID    = data.quant_project.main.uuid
  }
}

# Module example (like your use case)
module "quant_credentials" {
  source = "../quant-credentials"
  count  = var.include_quant ? 1 : 0

  project_machine_name = var.quant_project
}

# Example outputs for the module
output "project_details" {
  description = "Full project details"
  value = {
    id           = data.quant_project.main.id
    name         = data.quant_project.main.name
    uuid         = data.quant_project.main.uuid
    machine_name = data.quant_project.main.machine_name
    region       = data.quant_project.main.region
  }
}

output "write_token" {
  description = "Project write token for API access"
  value       = data.quant_project.main.write_token
  sensitive   = true  # Mark as sensitive since it's an API token
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

variable "project_machine_name" {
  description = "Machine name of the Quant project"
  type        = string
}

variable "include_quant" {
  description = "Whether to include Quant resources"
  type        = bool
  default     = true
}

variable "repository" {
  description = "GitHub repository name"
  type        = string
  default     = ""
}

variable "quant_project" {
  description = "Quant project machine name"
  type        = string
  default     = "default"
} 