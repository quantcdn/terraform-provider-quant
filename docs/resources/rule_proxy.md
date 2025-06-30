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

# Explicitly unset caching (useful for updating existing rules)
resource "quant_rule_proxy" "unset_cache" {
  name    = "unset-cache"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/api/*"]
  to      = "https://backend.example.com"
  cache_lifetime = -1  # Explicitly unset - respects origin headers
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

### Migrating Existing Rules to Use Origin Headers

If you have existing proxy rules that explicitly set `cache_lifetime = 0` but you want them to respect origin headers instead, you can use the `-1` sentinel value:

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
  cache_lifetime = -1  # Respects origin headers
}
```

This is particularly useful when updating existing infrastructure where you want to change from explicit cache control to origin-based cache control.

## Argument Reference

* `name` - (Required) The name of the rule.
* `project` - (Required) The machine name of the project.
* `domain` - (Required) The domain to apply the rule to.
* `url` - (Required) List of URL patterns to match.
* `to` - (Required) The target URL to proxy requests to.
* `host` - (Optional) The host header to send to the backend.
* `cache_lifetime` - (Optional) Cache lifetime in seconds. Different values have different behaviors:
  - **Omitted**: Respects origin headers (recommended for new resources)
  - **`0`**: Disables caching entirely  
  - **`-1`**: Explicitly unset - respects origin headers (useful for migrating existing resources)
  - **Positive number**: Sets specific cache time in seconds
* `disable_ssl_verify` - (Optional) Disable SSL verification for backend connections. Defaults to `false`.
* `only_proxy_404` - (Optional) Only proxy requests that would return 404. Defaults to `false`.
* `proxy_strip_headers` - (Optional) List of headers to strip from the response.
* `proxy_strip_request_headers` - (Optional) List of headers to strip from the request.

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

## Attributes Reference

* `id` - The ID of the rule.
* `rule_id` - The unique identifier of the rule.
