# Crawler Resource

Manages web crawlers for content discovery and ingestion into Quant.

## Example Usage

### Basic Crawler

```hcl
resource "quant_crawler" "example" {
  project = "my-project"
  name    = "Main Site Crawler"
  domain  = "https://example.com"
  
  # Starting URLs
  start_urls = ["/", "/blog", "/products"]
  
  # Crawl configuration
  browser_mode = false
  depth        = -1  # Unlimited depth
  delay        = 4.0 # Seconds between requests
  
  # URL filtering
  exclude = [
    "^/admin",
    "^/private",
    "\\.pdf$"
  ]
}
```

### Advanced Crawler with Asset Harvesting

```hcl
resource "quant_crawler" "advanced" {
  project      = "my-project"
  name         = "Advanced Crawler"
  domain       = "https://app.example.com"
  browser_mode = true
  
  # Asset harvesting configuration
  assets = {
    network_intercept = {
      enabled    = true
      execute_js = true
      timeout    = 30
    }
    parser = {
      enabled = true
    }
  }
  
  # Crawl limits
  max_html   = 10000
  max_hits   = 50000
  max_errors = 100
  workers    = 5
  
  # Custom headers
  headers = {
    "Authorization" = "Bearer token123"
    "X-Custom"      = "value"
  }
  
  # Webhook notifications
  webhook_url         = "https://hooks.example.com/crawl-complete"
  webhook_auth_header = "Bearer webhook-secret"
}
```

### Multi-Domain Crawler with Sitemap

```hcl
resource "quant_crawler" "multidomain" {
  project = "my-project"
  name    = "Multi-Domain Crawler"
  domain  = "https://main.example.com"
  
  # Multi-domain crawling
  allowed_domains = [
    "https://blog.example.com",
    "https://shop.example.com"
  ]
  
  # Sitemap configuration
  sitemap = [
    {
      url       = "https://example.com/sitemap.xml"
      recursive = true
    }
  ]
  
  # Include patterns
  include = [
    "^/articles/",
    "^/products/"
  ]
  
  # Status codes to capture
  status_ok = [200, 201, 301, 302]
}
```

## Argument Reference

### Required Arguments

- `domain` - (Required) The primary domain to crawl. Must include protocol (http:// or https://).
- `project` - (Required) The machine name of the Quant project.

### Optional Arguments

#### Basic Configuration

- `name` - (Optional) A descriptive name for the crawler.
- `browser_mode` - (Optional) Enable browser mode for JavaScript rendering. Default: `false`.
- `depth` - (Optional) Maximum crawl depth. Use `-1` for unlimited. Must be >= -1.
- `delay` - (Optional) Delay between requests in seconds. Default: `4.0`. Non-default values may require verification.
- `workers` - (Optional) Number of concurrent workers. Default: `2`. Range: 1-20. Non-default values may require verification.
- `user_agent` - (Optional) Custom user agent string. Only used when `browser_mode = false`.

#### URL Configuration

- `start_urls` - (Optional) List of starting URLs for the crawl. Default: `["/"]`.
- `urls` - (Optional) Additional URLs to crawl.
- `include` - (Optional) List of regex patterns for URLs to include in the crawl.
- `exclude` - (Optional) List of regex patterns for URLs to exclude from the crawl.
- `status_ok` - (Optional) List of HTTP status codes that will result in content being captured. Default: `[200]`.

#### Multi-Domain Crawling

- `allowed_domains` - (Optional) List of additional domains to crawl. Automatically enables merge_domains mode.
- `sitemap` - (Optional) Sitemap configuration. Each entry has:
  - `url` - Sitemap URL
  - `recursive` - Whether to recursively follow sitemap links

#### Asset Harvesting

- `assets` - (Optional) Asset harvesting configuration:
  - `network_intercept` - Network intercept settings:
    - `enabled` - Enable network intercept
    - `execute_js` - Execute JavaScript during asset collection
    - `timeout` - Request timeout in seconds
  - `parser` - Parser configuration:
    - `enabled` - Enable parser for asset extraction

#### Crawl Limits

- `max_html` - (Optional) Maximum HTML pages to crawl. Use `0` for unlimited (subject to org limits). Non-default values may require verification. Must be >= 0.
- `max_hits` - (Optional) Maximum total requests. Use `0` for unlimited. Non-default values may require verification. Must be >= 0.
- `max_errors` - (Optional) Maximum errors before stopping the crawl. Must be >= 0.

#### Authentication & Headers

- `headers` - (Optional) Map of custom HTTP headers to send with requests.

#### Webhooks

- `webhook_url` - (Optional) Webhook URL for crawl notifications.
- `webhook_auth_header` - (Optional) Authorization header value for webhook requests.
- `webhook_extra_vars` - (Optional) Additional variables to include in webhook payload.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The numeric ID of the crawler.
- `uuid` - The UUID of the crawler.
- `created_at` - Timestamp when the crawler was created.
- `updated_at` - Timestamp when the crawler was last updated.
- `deleted_at` - Timestamp when the crawler was deleted (if applicable).
- `config` - Full crawler configuration in YAML format (computed).
- `urls_list` - URLs list in YAML format (computed).
- `project_id` - The numeric ID of the project.
- `domain_verified` - Domain verification status (0 = unverified, 1 = verified).
- `organization` - The organization identifier.
- `crawler` - Internal crawler identifier.

## Import

Crawlers can be imported using the format `organization/project/uuid`:

```shell
terraform import quant_crawler.example my-org/my-project/550e8400-e29b-41d4-a716-446655440000
```

## Notes

- **Verification Requirements**: Some settings like `delay`, `max_html`, `max_hits`, and `workers` may require verification from the Quant team for non-default values.
- **Browser Mode**: When `browser_mode = true`, the crawler uses a headless browser to render JavaScript. This is slower but captures dynamic content.
- **Domain Verification**: The `domain` must be verified before the crawler can run successfully. Check `domain_verified` status.
- **Rate Limiting**: The crawler respects the provider's rate limiting configuration.
- **Regex Patterns**: The `include` and `exclude` patterns use Go regex syntax.
- **Multi-Domain**: When using `allowed_domains`, ensure all domains are accessible and properly configured.
