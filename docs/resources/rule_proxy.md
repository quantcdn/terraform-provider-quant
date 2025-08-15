# Rule Proxy Resource

Manages a Quant proxy rule.

## Example Usage

### Basic Proxy Rule

```hcl
resource "quant_rule_proxy" "basic" {
  name    = "basic-proxy"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  host    = "backend.example.com"
  proxy_alert_enabled = true
}
```

### Cache Lifetime Examples

```hcl
# Respect origin headers (omit cache_lifetime)
resource "quant_rule_proxy" "respect_origin" {
  name    = "respect-origin"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  # cache_lifetime omitted - respects origin headers
}

# Disable caching
resource "quant_rule_proxy" "no_cache" {
  name    = "no-cache"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  cache_lifetime = 0  # Disables caching
}

# Respect origin headers using -1
resource "quant_rule_proxy" "unset_cache" {
  name    = "unset-cache"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  cache_lifetime = -1  # Backend will respect origin headers
}

# Set specific cache time
resource "quant_rule_proxy" "cached" {
  name    = "cached"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  cache_lifetime = 3600  # Cache for 1 hour
}
```

### Full Example with WAF

```hcl
resource "quant_rule_proxy" "full_example" {
  name    = "full-proxy"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/proxy"]
  country = "country_is"
  country_is = ["US", "CA"]
  method = "method_is"
  method_is = ["GET", "POST"]
  ip = "ip_is"
  ip_is = ["192.168.1.1", "192.168.1.2"]
  
  to = "https://backend.example.com"
  host = "backend.example.com"
  cache_lifetime = 1800  # 30 minutes
  proxy_alert_enabled = true  # Enable proxy monitoring alerts
  
  waf_enabled = true
  waf_config = {
    mode = "report"
    paranoia_level = 1
    allow_rules = []
    allow_ip = []
    block_ip = []
    block_ua = []
    block_referer = []
    notify_email = []
    notify_slack = ""
    notify_slack_hits_rpm = 100
    
    ip_ratelimit_mode = "disabled"
    ip_ratelimit_rps = 5
    ip_ratelimit_cooldown = 30
    
    request_header_name = ""
    request_header_ratelimit_mode = "disabled"
    request_header_ratelimit_rps = 5
    request_header_ratelimit_cooldown = 30
    
    waf_ratelimit_mode = "disabled"
    waf_ratelimit_rps = 5
    waf_ratelimit_hits = 10
    waf_ratelimit_cooldown = 300
  }

  notify = "slack"
  notify_config = {
    origin_status_codes = ["200", "201"]
    period = "60"
    slack_webhook = "https://hooks.slack.com/services/xxx/xxx/xxx"
  }
}
```

### Application Proxy Example

```hcl
resource "quant_rule_proxy" "app_proxy" {
  name    = "orders-proxy"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/orders/*"]

  # Note: `to` and `host` are computed by the backend when proxying to an application.

  application_proxy       = true
  application_name        = "orders"
  application_environment = "staging"
  application_container   = "orders-api"
  application_port        = 8080

  waf_config = {
    mode = "report"
  }

  # Optional origin timeout (milliseconds) – represented as a string by the API
  origin_timeout = "30000"
}
```

### Migrating Existing Rules to Use Origin Headers

If you have existing proxy rules that explicitly set `cache_lifetime = 0` but you want them to respect origin headers instead, you can use `-1`:

```hcl
# Before: Explicitly disabled caching
resource "quant_rule_proxy" "example" {
  name    = "example"
  project = quant_project.example.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  cache_lifetime = 0  # Disables caching
}

# After: Respect origin headers
resource "quant_rule_proxy" "example" {
  name    = "example"
  project = quant_project.example.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  cache_lifetime = -1  # Backend will respect origin headers
}
```

This is particularly useful when updating existing infrastructure where you want to change from explicit cache control to origin-based cache control.

## Argument Reference

* `name` - (Required) The name of the rule.
* `project` - (Required) The machine name of the project.
* `domain` - (Required) The domain to apply the rule to.
* `url` - (Required) List of URL patterns to match.
* `to` - (Conditionally Required) The target URL to proxy requests to. Required when `application_proxy` is not true. Must be omitted when `application_proxy` is true as it will be computed by the backend.
* `host` - (Optional) The host header to send to the backend, if `application_proxy` is true, this field must be omitted.
* `origin_timeout` - (Optional) Origin timeout in milliseconds. Represented as a string by the API (e.g. `"30000"`).
* `cache_lifetime` - (Optional) Cache lifetime in seconds. Different values have different behaviors:
  - **Omitted**: Respects origin headers (recommended for new resources)
  - **`0`**: Disables caching entirely  
  - **`-1`**: Backend will respect origin headers
  - **Positive number**: Sets specific cache time in seconds
* `disable_ssl_verify` - (Optional) Disable SSL verification for backend connections. Defaults to `false`.
* `only_proxy_404` - (Optional) Only proxy requests that would return 404. Defaults to `false`.
* `proxy_alert_enabled` - (Optional) Enable proxy alerts for monitoring and notifications. Defaults to `false`.
* `proxy_strip_headers` - (Optional) List of headers to strip from the response.
* `proxy_strip_request_headers` - (Optional) List of headers to strip from the request.

### Application Proxy (Request-only computation fields)

These fields instruct the backend to compute application proxy settings. They are provided on create/update and preserved in state for consistency.

* `application_proxy` - (Optional) Enable application proxy behaviour. Defaults to `false`.
* `application_name` - (Optional) The application name to route to (e.g. `"orders"`).
* `application_environment` - (Optional) The application environment (e.g. `"staging"`, `"prod"`).
* `application_container` - (Optional) The application container identifier/name.
* `application_port` - (Optional) The application container port number.

### Rule Selection Criteria

* `country` - (Optional) One of `country_is`, `country_is_not`, or `any`.
* `country_is` - (Optional) List of country codes to match.
* `country_is_not` - (Optional) List of country codes to not match.
* `ip` - (Optional) One of `ip_is`, `ip_is_not`, or `any`.
* `ip_is` - (Optional) List of IP addresses to match.
* `ip_is_not` - (Optional) List of IP addresses to not match.
* `method` - (Optional) One of `method_is`, `method_is_not`, or `any`.
* `method_is` - (Optional) List of HTTP methods to match.
* `method_is_not` - (Optional) List of HTTP methods to not match.

### WAF Configuration

* `waf_enabled` - (Optional) Enable WAF protection. Defaults to `false`.
* `waf_config` - (Optional) WAF configuration block.
  * `mode` - (Required) WAF mode. Either `block` or `report`.
  * `paranoia_level` - (Optional) WAF paranoia level (1-4). Defaults to `1`.
  * `allow_rules` - (Optional) List of WAF rule IDs to allow.
  * `allow_ip` - (Optional) List of IPs to allow.
  * `block_ip` - (Optional) List of IPs to block.
  * `block_ua` - (Optional) List of User Agents to block.
  * `block_referer` - (Optional) List of Referers to block.
  * `notify_email` - (Optional) List of email addresses for notifications.
  * `notify_slack` - (Optional) Slack webhook URL for notifications.
  * `notify_slack_hits_rpm` - (Optional) Threshold for Slack notifications (hits per minute).
  * `ip_ratelimit_mode` - (Optional) IP rate limiting mode (`disabled`, `block`, `report`). Defaults to `disabled`.
  * `ip_ratelimit_rps` - (Optional) IP rate limit requests per second. Defaults to `5`.
  * `ip_ratelimit_cooldown` - (Optional) IP rate limit cooldown period in seconds. Defaults to `30`.
  * `request_header_name` - (Optional) Header name for request-based rate limiting.
  * `request_header_ratelimit_mode` - (Optional) Header rate limiting mode. Defaults to `disabled`.
  * `request_header_ratelimit_rps` - (Optional) Header rate limit requests per second. Defaults to `5`.
  * `request_header_ratelimit_cooldown` - (Optional) Header rate limit cooldown period in seconds. Defaults to `30`.
  * `waf_ratelimit_mode` - (Optional) WAF rate limiting mode. Defaults to `disabled`.
  * `waf_ratelimit_rps` - (Optional) WAF rate limit requests per second. Defaults to `5`.
  * `waf_ratelimit_hits` - (Optional) WAF rate limit hit threshold. Defaults to `10`.
  * `waf_ratelimit_cooldown` - (Optional) WAF rate limit cooldown period in seconds. Defaults to `300`.
  * `httpbl_enabled` - (Optional) HTTP:BL configuration map.
  * `thresholds` - (Optional) List of threshold configurations.
    * `cooldown` - (Optional) Threshold cooldown period in seconds.
    * `hits` - (Optional) Number of hits to trigger threshold.
    * `minutes` - (Optional) Time window in minutes.
    * `mode` - (Optional) Threshold mode. Defaults to `disabled`.
    * `notify_slack` - (Optional) Slack webhook URL for threshold notifications.
    * `rps` - (Optional) Requests per second threshold.
    * `type` - (Optional) Threshold type.
    * `value` - (Optional) Threshold value.

### Failover Configuration

* `failover` - (Optional) Failover configuration block.
  * `failover_mode` - (Optional) Enable failover mode. Defaults to `false`.
  * `failover_lifetime` - (Optional) Failover cache lifetime. Defaults to `300`.
  * `failover_origin_status_codes` - (Optional) List of origin status codes to trigger failover.
  * `failover_origin_ttfb` - (Optional) Origin TTFB threshold in milliseconds. Defaults to `2000`.

### Notification Configuration

* `notify` - (Optional) Notification type. Either `none` or `slack`. Defaults to `none`.
* `notify_config` - (Optional) Notification configuration block.
  * `origin_status_codes` - (Optional) List of origin status codes to notify on.
  * `period` - (Optional) Notification period in seconds. Defaults to `60`.
  * `slack_webhook` - (Optional) Slack webhook URL.

## Import

Proxy rules can be imported using the following format:

```bash
terraform import quant_rule_proxy.resource_name "project_name/rule_uuid"
```

### Example

```bash
terraform import quant_rule_proxy.my_proxy "my-project/12345678-1234-1234-1234-123456789abc"
```

### Import Format

The import ID must follow the pattern `project_name/rule_uuid` where:
- `project_name` is the machine name of your project (not the display name)
- `rule_uuid` is the UUID of the proxy rule in format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

### Finding the Rule UUID

To find the rule UUID, you can:
1. Use the QuantCDN dashboard - Look in the rules section for your proxy rule
2. Use the API directly - Call the rules list endpoint
3. Check existing Terraform state - If you have other rules already managed

### Step-by-Step Import Process

1. **Add the resource to your Terraform configuration**:
   ```hcl
   resource "quant_rule_proxy" "my_proxy" {
     name    = "my-proxy-rule"
     project = "my-project"
     # Other required fields will be populated from the import
   }
   ```

2. **Run the import command**:
   ```bash
   terraform import quant_rule_proxy.my_proxy "my-project/12345678-1234-1234-1234-123456789abc"
   ```

3. **Verify the import**:
   ```bash
   terraform plan
   ```

### Troubleshooting

If you get an error like "Invalid UUID format", make sure:
- The UUID is in the correct format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
- There are no extra spaces or characters
- The project name is correct

The import will automatically handle the weight field and all other attributes from the existing rule.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `uuid` - The UUID of the rule.
* `rule_id` - The ID of the rule.
* `organization` - The organization ID this rule belongs to.
* `weight` - The weight of the rule.
* `action` - The action type, always set to `proxy`.
* `only_with_cookie` - The cookie condition for the rule.