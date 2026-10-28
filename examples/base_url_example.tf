# Example: Using a custom base URL
# Set base_url to the bare host of the dashboard. Do not add a path such as /api/v2.

# Public platform (default)
provider "quant" {
  alias = "public"

  bearer       = var.public_api_token
  organization = var.public_organization
  # Uses the default base URL: https://dashboard.quantcdn.io
}

# AU Government platform
provider "quant" {
  alias = "gov"

  bearer       = var.gov_api_token
  organization = var.gov_organization
  base_url     = "https://dash.quantgov.cloud"
}

# Custom host
provider "quant" {
  alias = "custom"

  bearer       = var.custom_api_token
  organization = var.custom_organization
  base_url     = "https://dashboard.example.com"
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
resource "quant_project" "public_project" {
  provider = quant.public
  name     = "Public Project"
}

resource "quant_project" "gov_project" {
  provider = quant.gov
  name     = "Gov Project"
}

# Data source using a custom base URL
data "quant_project" "custom_project" {
  provider     = quant.custom
  machine_name = "example-project"
  with_token   = true
}

# Variables
variable "public_api_token" {
  description = "Public platform API token"
  type        = string
  sensitive   = true
}

variable "public_organization" {
  description = "Public platform organization"
  type        = string
}

variable "gov_api_token" {
  description = "AU Government platform API token"
  type        = string
  sensitive   = true
}

variable "gov_organization" {
  description = "AU Government platform organization"
  type        = string
}

variable "custom_api_token" {
  description = "Custom host API token"
  type        = string
  sensitive   = true
}

variable "custom_organization" {
  description = "Custom host organization"
  type        = string
}
