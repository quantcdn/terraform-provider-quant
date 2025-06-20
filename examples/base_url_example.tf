# Example: Using custom base URL for different environments

# Production environment (default)
provider "quant" {
  alias = "prod"
  
  bearer       = var.prod_api_token
  organization = var.prod_organization
  # Uses default base URL: https://dashboard.quantcdn.io/api/v2
}

# Staging environment
provider "quant" {
  alias = "staging"
  
  bearer       = var.staging_api_token
  organization = var.staging_organization
  base_url     = "https://staging-dashboard.quantcdn.io/api/v2"
}

# Development environment
provider "quant" {
  alias = "dev"
  
  bearer       = var.dev_api_token
  organization = var.dev_organization
  base_url     = "https://dev-dashboard.quantcdn.io/api/v2"
}

# Custom API endpoint
provider "quant" {
  alias = "custom"
  
  bearer       = var.custom_api_token
  organization = var.custom_organization
  base_url     = "https://my-custom-api.example.com/api/v2"
}

# Using environment variables
provider "quant" {
  alias = "env"
  
  # These will use environment variables:
  # - QUANTCDN_API_TOKEN
  # - QUANTCDN_ORGANIZATION  
  # - QUANTCDN_BASE_URL
}

# Example resources using different providers
resource "quant_project" "prod_project" {
  provider = quant.prod
  name     = "Production Project"
}

resource "quant_project" "staging_project" {
  provider = quant.staging
  name     = "Staging Project"
}

resource "quant_project" "dev_project" {
  provider = quant.dev
  name     = "Development Project"
}

# Data source using custom base URL
data "quant_project" "custom_project" {
  provider     = quant.custom
  machine_name = "my-project"
  with_token   = true
}

# Variables
variable "prod_api_token" {
  description = "Production API token"
  type        = string
  sensitive   = true
}

variable "prod_organization" {
  description = "Production organization"
  type        = string
}

variable "staging_api_token" {
  description = "Staging API token"
  type        = string
  sensitive   = true
}

variable "staging_organization" {
  description = "Staging organization"
  type        = string
}

variable "dev_api_token" {
  description = "Development API token"
  type        = string
  sensitive   = true
}

variable "dev_organization" {
  description = "Development organization"
  type        = string
}

variable "custom_api_token" {
  description = "Custom API token"
  type        = string
  sensitive   = true
}

variable "custom_organization" {
  description = "Custom organization"
  type        = string
} 