# Example: Force refresh all crawler resources
#
# This example shows how to use the force_refresh attribute to trigger
# updates on all crawler resources, which is useful when the backend API
# has been fixed and you want to refresh the crawler configurations.

terraform {
  required_providers {
    quant = {
      source = "quantcdn/quant"
    }
  }
}

# Local variable to control when to force refresh all crawlers
# Change this value to trigger a refresh of all crawler resources
locals {
  force_refresh_timestamp = "2024-01-15T10:30:00Z"  # Change this to trigger refresh
}

# Example crawler resources using modules
module "acma-dr" {
  source = "./modules/crawler"
  
  project      = var.project
  name         = "acma-dr-crawler"
  domain       = "acma.gov.au"
  urls         = ["/"]
  browser_mode = true
  exclude      = ["/admin", "/private"]
  
  # This will trigger an update when changed
  force_refresh = local.force_refresh_timestamp
}

module "example-crawler" {
  source = "./modules/crawler"
  
  project      = var.project
  name         = "example-crawler"
  domain       = "example.com"
  urls         = ["/", "/docs"]
  browser_mode = false
  
  # This will trigger an update when changed
  force_refresh = local.force_refresh_timestamp
}

# Alternative: Direct crawler resource (not using modules)
resource "quant_crawler" "direct_crawler" {
  project      = var.project
  name         = "direct-crawler"
  domain       = "direct.example.com"
  urls         = ["/"]
  browser_mode = false
  
  # This will trigger an update when changed
  force_refresh = local.force_refresh_timestamp
}

# Variables
variable "project" {
  description = "Quant project machine name"
  type        = string
  default     = "default"
}

# Outputs
output "crawler_info" {
  description = "Information about the crawlers"
  value = {
    acma_dr_uuid     = module.acma-dr.crawler_uuid
    example_uuid     = module.example-crawler.crawler_uuid
    direct_uuid      = quant_crawler.direct_crawler.uuid
    refresh_trigger  = local.force_refresh_timestamp
  }
}
