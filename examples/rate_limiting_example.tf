# Example of using QuantCDN Terraform Provider with custom rate limiting

terraform {
  required_providers {
    quant = {
      source = "local/quantcdn/quant"
    }
  }
}

# Basic configuration with default rate limiting (10 req/sec, 3 retries)
provider "quant" {
  bearer       = var.quantcdn_api_token
  organization = var.quantcdn_organization
}

# Custom rate limiting configuration
provider "quant" {
  alias        = "high_throughput"
  bearer       = var.quantcdn_api_token
  organization = var.quantcdn_organization
  
  # Custom rate limiting settings
  requests_per_second = 20.0        # Allow 20 requests per second
  max_retries        = 5            # Retry up to 5 times
  base_delay_ms      = 1000         # Start with 1 second delay
  max_delay_ms       = 60000        # Maximum 60 second delay
  enable_jitter      = true         # Add random jitter to delays
}

# Conservative rate limiting for sensitive operations
provider "quant" {
  alias        = "conservative"
  bearer       = var.quantcdn_api_token
  organization = var.quantcdn_organization
  
  # Conservative settings
  requests_per_second = 2.0         # Only 2 requests per second
  max_retries        = 10           # More retries for resilience
  base_delay_ms      = 2000         # Start with 2 second delay
  max_delay_ms       = 120000       # Allow up to 2 minute delays
  enable_jitter      = true
}

# Example resources using different rate limiting configurations

# Domain using default rate limiting
resource "quant_domain" "example" {
  domain  = "example.com"
  project = "default"
}

# Project using high throughput rate limiting
resource "quant_project" "high_volume" {
  provider = quant.high_throughput
  
  name        = "High Volume Project"
  region      = "us-east-1"
  description = "Project with high API throughput requirements"
}

# Crawler using conservative rate limiting
resource "quant_crawler" "sensitive" {
  provider = quant.conservative
  
  name         = "Sensitive Crawler"
  project      = "sensitive-project"
  domain       = "https://sensitive-site.example.com"
  browser_mode = true
  
  urls = ["/critical-path"]
  
  headers = {
    "User-Agent" = "QuanTerraform-Conservative/1.0"
  }
}

# Variables
variable "quantcdn_api_token" {
  description = "QuantCDN API Token"
  type        = string
  sensitive   = true
}

variable "quantcdn_organization" {
  description = "QuantCDN Organization"
  type        = string
}

# Environment variables can also be used:
# export QUANTCDN_API_TOKEN="your-token"
# export QUANTCDN_ORGANIZATION="your-org"
# export QUANTCDN_REQUESTS_PER_SECOND="15.0"
# export QUANTCDN_MAX_RETRIES="5"
# export QUANTCDN_BASE_DELAY_MS="750"
# export QUANTCDN_MAX_DELAY_MS="45000"
# export QUANTCDN_ENABLE_JITTER="true" 