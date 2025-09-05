# Quant Provider

The Quant provider allows you to manage resources in your Quant CDN environment with built-in API rate limiting and intelligent retry mechanisms.

## Example Usage

### Basic Configuration

```hcl
provider "quant" {
  bearer       = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  organization = "quant"
}
```

### Advanced Configuration with Rate Limiting and Timeout Management

```hcl
provider "quant" {
  bearer              = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  organization        = "quant"
  base_url            = "https://custom-api.example.com/api/v2"  # Optional custom base URL
  
  # Timeout configuration for deadlock resilience
  timeout_seconds     = 120     # Total timeout per request (default: 120 seconds)
  
  # Rate limiting configuration
  requests_per_second = 15.0    # Allow 15 requests per second
  max_retries        = 5        # Retry failed requests up to 5 times
  base_delay_ms      = 1000     # Start with 1 second delay for retries
  max_delay_ms       = 60000    # Maximum 60 second delay between retries
  enable_jitter      = true     # Add randomisation to prevent thundering herd
}
```

### Configuration for Deadlock-Prone Environments

For environments that experience API deadlocks or slow responses:

```hcl
provider "quant" {
  bearer              = var.quant_bearer_token
  organization        = var.quant_organization
  
  # Extended timeout and retry configuration for deadlock resilience
  timeout_seconds     = 180     # Extended timeout for slow operations
  max_retries        = 5        # More retry attempts for recovery
  base_delay_ms      = 2000     # Longer initial delays
  max_delay_ms       = 45000    # Extended maximum delay
  requests_per_second = 5.0     # Slower rate to reduce API pressure
}
```

### Multiple Provider Configurations

```hcl
# Default provider for most resources
provider "quant" {
  bearer       = var.quant_bearer_token
  organization = var.quant_organization
}

# High-throughput provider for bulk operations
provider "quant" {
  alias               = "bulk"
  bearer              = var.quant_bearer_token
  organization        = var.quant_organization
  requests_per_second = 25.0
  max_retries        = 2
}

# Conservative provider for critical resources with deadlock resilience
provider "quant" {
  alias               = "critical"
  bearer              = var.quant_bearer_token
  organization        = var.quant_organization
  timeout_seconds     = 300     # 5 minute timeout for critical operations
  requests_per_second = 2.0
  max_retries        = 10
  base_delay_ms      = 2000     # Longer initial delays
  max_delay_ms       = 300000   # 5 minutes maximum delay
}
```

## Schema

### Required Arguments

- `bearer` (String) - The bearer token for authentication with the Quant API.
- `organization` (String) - The organization name/identifier.

### Optional Arguments

- `base_url` (String) - The base URL for the QuantCDN API. Useful for pointing to staging, development, or custom API endpoints. Can also be set via QUANTCDN_BASE_URL environment variable.
- `timeout_seconds` (Number) - HTTP client timeout in seconds for API requests. Controls how long to wait for API responses before timing out. Particularly useful for handling slow operations and API deadlocks. Default: `120` seconds. Can also be set via QUANTCDN_TIMEOUT_SECONDS environment variable.
- `requests_per_second` (Number) - Maximum number of requests per second to send to the API. Controls the rate limiting behaviour. Default: `10.0`
- `max_retries` (Number) - Maximum number of retry attempts for failed requests. The provider will retry requests that fail due to rate limiting (429), server errors (5xx), network errors, or timeout errors. Default: `3`
- `base_delay_ms` (Number) - Base delay in milliseconds for exponential backoff retry logic. Each retry attempt increases the delay exponentially from this base value. Default: `500`
- `max_delay_ms` (Number) - Maximum delay in milliseconds for exponential backoff. Prevents retry delays from becoming excessively long. Default: `30000` (30 seconds)
- `enable_jitter` (Boolean) - Whether to add random jitter to retry delays. Helps prevent thundering herd effects when multiple provider instances are retrying simultaneously. Default: `true`

## Environment Variables

All configuration options can be set via environment variables as an alternative to explicit configuration:

- `QUANTCDN_API_TOKEN` - Bearer token (alternative to `bearer`)
- `QUANTCDN_ORGANIZATION` - Organization name (alternative to `organization`)
- `QUANTCDN_BASE_URL` - Base URL for the API (alternative to `base_url`)
- `QUANTCDN_TIMEOUT_SECONDS` - HTTP client timeout in seconds (alternative to `timeout_seconds`)
- `QUANTCDN_REQUESTS_PER_SECOND` - Requests per second limit
- `QUANTCDN_MAX_RETRIES` - Maximum retry attempts
- `QUANTCDN_BASE_DELAY_MS` - Base retry delay in milliseconds
- `QUANTCDN_MAX_DELAY_MS` - Maximum retry delay in milliseconds
- `QUANTCDN_ENABLE_JITTER` - Enable jitter (true/false)

## Rate Limiting, Retry Behaviour, and Deadlock Management

The provider includes sophisticated rate limiting, retry mechanisms, and deadlock resilience:

### Timeout Management
- **Configurable Timeouts**: Set custom timeout values to handle slow API operations
- **Deadlock Detection**: Automatically detects and recovers from API deadlocks
- **Per-Request Timeouts**: Fine-grained timeout control for individual requests

### Rate Limiting and Retry Logic
- **Token Bucket Rate Limiting**: Controls the number of API requests per second using a token bucket algorithm
- **Exponential Backoff**: Failed requests are retried with exponentially increasing delays
- **Smart Retry Logic**: Automatically retries requests that fail due to:
  - Rate limiting (HTTP 429)
  - Server errors (HTTP 5xx)
  - Network errors and connection issues
  - Timeout errors and deadlocks (`context deadline exceeded`, `Client.Timeout exceeded`)
  - Connection resets and refused connections
- **Jitter Support**: Adds randomisation to retry delays to prevent coordinated retry storms
- **Retry-After Support**: Respects `Retry-After` headers provided by the API
- **Context Awareness**: Properly handles request cancellation and timeouts

### Deadlock Resilience
The provider automatically handles common API deadlock scenarios:
- **Timeout Detection**: Recognises timeout patterns that indicate potential deadlocks
- **Automatic Recovery**: Retries requests with backoff when deadlocks are detected
- **Configurable Resilience**: Tune timeout and retry settings for your environment's needs

## Best Practices

### General Configuration
1. **Start with defaults**: The default configuration works well for most use cases
2. **Monitor API usage**: Adjust `requests_per_second` based on your API quota and actual usage patterns
3. **Use appropriate retry settings**: Increase `max_retries` for critical infrastructure deployments
4. **Leverage provider aliases**: Use different rate limiting profiles for different types of operations
5. **Test configurations**: Verify rate limiting settings in development environments before production use
6. **Respect API limits**: Don't exceed your API plan's rate limits with the `requests_per_second` setting

### Deadlock and Timeout Management
7. **Increase timeouts for slow environments**: Use `timeout_seconds = 180` or higher for environments with slow API responses
8. **Configure retries for deadlock recovery**: Set `max_retries = 5` or higher for environments prone to API deadlocks
9. **Use longer delays in unstable environments**: Increase `base_delay_ms` and `max_delay_ms` for better stability
10. **Monitor for timeout patterns**: If you frequently see "context deadline exceeded" errors, increase `timeout_seconds`
11. **Adjust rate limiting for stability**: Lower `requests_per_second` if experiencing frequent deadlocks or timeouts

### Environment-Specific Recommendations
- **Development**: Use default settings for fast feedback
- **Staging**: Use moderate timeout increases (`timeout_seconds = 150`)
- **Production**: Use conservative settings with higher timeouts and retries
- **High-traffic environments**: Lower `requests_per_second` and increase `max_retries`



## Resources and Data Sources

### Data Sources

#### quant_project

Fetches details for a specific Quant project, including the write token for use in other providers.

```hcl
data "quant_project" "example" {
  machine_name = "my-project"
  with_token   = true  # Default is true
}

# Use the write token in other providers
resource "github_actions_secret" "quant_token" {
  repository      = "my-org/my-repo"
  secret_name     = "QUANT_TOKEN"
  plaintext_value = data.quant_project.example.write_token
}
```

**Schema:**

- `machine_name` (String, Required) - The machine name of the project to fetch
- `with_token` (Boolean, Optional) - Whether to include the write token in the response. Default: `true`

**Attributes:**

- `id` (Number) - Numeric project ID
- `name` (String) - Project display name
- `uuid` (String) - Project UUID
- `machine_name` (String) - Project machine name
- `region` (String) - Deployment region
- `organization_id` (Number) - Organization ID
- `security_score` (String) - Project security score
- `git_url` (String) - Associated Git repository URL
- `write_token` (String, Sensitive) - Project write token (only when `with_token = true`)
- `created_at` (String) - Creation timestamp
- `updated_at` (String) - Last update timestamp

#### quant_projects

Lists all projects in the organisation.

```hcl
data "quant_projects" "all" {}

output "project_names" {
  value = [for project in data.quant_projects.all.projects : project.name]
}
```
