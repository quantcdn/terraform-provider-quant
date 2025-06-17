# Terraform Provider Quant

## Quickstarts

- [Getting started with QuantCDN and terraform](https://docs.quantcdn.io/terraform/getting-started)

## Rate Limiting and Retry Configuration

The QuantCDN Terraform provider includes built-in API rate limiting and intelligent retry mechanisms with exponential backoff to ensure reliable operation and compliance with API limits.

### Features

- **Token Bucket Rate Limiting**: Controls the number of requests per second to the QuantCDN API
- **Exponential Backoff**: Automatically retries failed requests with increasing delays
- **Smart Retry Logic**: Handles rate limiting (429), server errors (5xx), and network errors
- **Jitter Support**: Adds randomisation to retry delays to prevent thundering herd effects
- **Retry-After Support**: Respects API-provided retry timing headers
- **Context-Aware**: Properly handles request cancellation and timeouts

### Configuration Options

All rate limiting parameters can be configured via the provider block:

```hcl
provider "quant" {
  bearer              = var.quantcdn_api_token
  organization        = var.quantcdn_organization
  
  # Rate limiting configuration (all optional)
  requests_per_second = 10.0    # Max requests per second (default: 10.0)
  max_retries        = 3        # Max retry attempts (default: 3)
  base_delay_ms      = 500      # Initial retry delay in ms (default: 500)
  max_delay_ms       = 30000    # Maximum retry delay in ms (default: 30000)
  enable_jitter      = true     # Add jitter to delays (default: true)
}
```

### Environment Variables

Configuration can also be set via environment variables:

```bash
export QUANTCDN_API_TOKEN="your-token"
export QUANTCDN_ORGANIZATION="your-org"
export QUANTCDN_REQUESTS_PER_SECOND="15.0"
export QUANTCDN_MAX_RETRIES="5"
export QUANTCDN_BASE_DELAY_MS="1000"
export QUANTCDN_MAX_DELAY_MS="60000"
export QUANTCDN_ENABLE_JITTER="true"
```

### Usage Examples

#### Default Configuration
Most users can simply use the default configuration:

```hcl
provider "quant" {
  bearer       = var.quantcdn_api_token
  organization = var.quantcdn_organization
}
```

#### High-Throughput Configuration
For environments requiring higher API throughput:

```hcl
provider "quant" {
  bearer              = var.quantcdn_api_token
  organization        = var.quantcdn_organization
  requests_per_second = 20.0
  max_retries        = 5
  base_delay_ms      = 1000
  max_delay_ms       = 60000
}
```

#### Conservative Configuration
For sensitive environments requiring maximum reliability:

```hcl
provider "quant" {
  bearer              = var.quantcdn_api_token
  organization        = var.quantcdn_organization
  requests_per_second = 2.0
  max_retries        = 10
  base_delay_ms      = 2000
  max_delay_ms       = 120000
}
```

#### Multiple Provider Configurations
Use aliases for different rate limiting profiles:

```hcl
# Default provider for most resources
provider "quant" {
  bearer       = var.quantcdn_api_token
  organization = var.quantcdn_organization
}

# High-throughput provider for bulk operations
provider "quant" {
  alias               = "bulk"
  bearer              = var.quantcdn_api_token
  organization        = var.quantcdn_organization
  requests_per_second = 25.0
  max_retries        = 2
}

# Conservative provider for critical resources
provider "quant" {
  alias               = "critical"
  bearer              = var.quantcdn_api_token
  organization        = var.quantcdn_organization
  requests_per_second = 1.0
  max_retries        = 15
  max_delay_ms       = 300000  # 5 minutes
}

# Use different providers for different resources
resource "quant_domain" "example" {
  domain  = "example.com"
  project = "default"
}

resource "quant_project" "bulk_import" {
  provider = quant.bulk
  name     = "Bulk Import Project"
}

resource "quant_crawler" "critical" {
  provider = quant.critical
  name     = "Mission Critical Crawler"
  domain   = "https://critical.example.com"
}
```

### Default Values

| Parameter | Default Value | Description |
|-----------|---------------|-------------|
| `requests_per_second` | `10.0` | Maximum requests per second to the API |
| `max_retries` | `3` | Maximum number of retry attempts |
| `base_delay_ms` | `500` | Initial delay in milliseconds for exponential backoff |
| `max_delay_ms` | `30000` | Maximum delay in milliseconds (30 seconds) |
| `enable_jitter` | `true` | Whether to add random jitter to retry delays |

### Best Practices

1. **Start with defaults**: The default configuration works well for most use cases
2. **Monitor API usage**: Adjust `requests_per_second` based on your API quota and usage patterns
3. **Increase retries for reliability**: Use higher `max_retries` for critical infrastructure
4. **Use provider aliases**: Configure different rate limiting profiles for different resource types
5. **Test in development**: Verify your rate limiting configuration in non-production environments
6. **Respect API limits**: Don't set `requests_per_second` higher than your API plan allows

### Troubleshooting

- **Frequent 429 errors**: Reduce `requests_per_second` or increase retry delays
- **Slow deployments**: Increase `requests_per_second` if within your API limits
- **Timeout errors**: Increase `max_delay_ms` for better resilience in unstable networks
- **Thundering herd**: Ensure `enable_jitter` is set to `true` when running multiple providers

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Developing the Provider

With terraform v0.14 and later, [development overrides for provider development](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers) can be used.

```
provider_installation {

  # Use /home/developer/tmp/terraform-null as an overridden package directory
  # for the hashicorp/null provider. This disables the version and checksum
  # verifications for this provider and forces Terraform to look for the
  # null provider plugin in the given directory.
  dev_overrides {
    "registry.terraform.io/quantcdn/quant" = "/path/to/quant-provider"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```