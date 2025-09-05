# Example configuration for handling API deadlocks and timeouts
# This configuration provides resilience against API deadlocks and slow responses

provider "quant" {
  bearer      = var.quant_bearer_token
  organization = var.quant_organization
  
  # Timeout configuration for deadlock resilience
  timeout_seconds = 120  # Total timeout per request (default: 120 seconds)
  
  # Retry configuration for handling transient failures
  max_retries     = 5    # Maximum retry attempts (default: 3)
  base_delay_ms   = 1000 # Base delay between retries in milliseconds (default: 500)
  max_delay_ms    = 30000 # Maximum delay between retries in milliseconds (default: 30000)
  enable_jitter   = true  # Add randomness to retry delays to avoid thundering herd (default: true)
  
  # Rate limiting to avoid overwhelming the API
  requests_per_second = 5  # Slower rate for stability (default: 10)
}

# Alternatively, you can set these via environment variables:
# export QUANTCDN_TIMEOUT_SECONDS=120
# export QUANTCDN_MAX_RETRIES=5
# export QUANTCDN_BASE_DELAY_MS=1000
# export QUANTCDN_MAX_DELAY_MS=30000
# export QUANTCDN_ENABLE_JITTER=true
# export QUANTCDN_REQUESTS_PER_SECOND=5

# Example crawler resource that benefits from deadlock resilience
resource "quant_crawler" "example" {
  name    = "Resilient Crawler"
  project = "my-project"
  domain  = "https://example.com"
  
  browser_mode = true
  
  urls = [
    "/",
    "/about",
    "/contact"
  ]
  
  headers = {
    "User-Agent" = "Quant Crawler"
  }
  
  exclude = [
    "/admin/*",
    "/private/*"
  ]
}

variable "quant_bearer_token" {
  description = "QuantCDN API Bearer Token"
  type        = string
  sensitive   = true
}

variable "quant_organization" {
  description = "QuantCDN Organization"
  type        = string
}
