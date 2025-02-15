# Rule Proxy Resource

Manages a Quant proxy rule.

## Example Usage

```hcl
resource "quant_rule_proxy" "test" {
  name    = "test-proxy"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/proxy"]
  country = "country_is"
  country_is = ["US", "CA"]
  method = "method_is"
  method_is = ["GET", "POST"]
  ip = "ip_is"
  ip_is = ["192.168.1.1", "192.168.1.2"]
  
  # Updated proxy block to match new schema
  to = "https://backend.example.com"
  host = "backend.example.com"
  
  failover = {
    failover_mode = true
    failover_lifetime = "1h"
    failover_origin_status_codes = ["200", "201"]
    failover_origin_ttfb = "2000"
  }

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
    
    # New rate limiting configurations
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
    
    httpbl_enabled = {}
    
    thresholds = [{
      cooldown = 60
      hits = 10
      minutes = 1
      mode = "block"
      notify_slack = "https://hooks.slack.com/services/xxx/xxx/xxx"
      rps = 1000
      type = "block"
      value = "192.168.1.1"
    }]
  }

  notify = "slack"
  notify_config = {
    origin_status_codes = ["200", "201"]
    period = "60"
    slack_webhook = "https://hooks.slack.com/services/xxx/xxx/xxx"
  }
}
```

## Argument Reference

* `name` - (Required) The name of the rule.
* `project` - (Required) The machine name of the project.
* `domain` - (Required) The domain to apply the rule to.
* `url` - (Required) List of URL patterns to match.
* `to` - (Required) The target URL to proxy requests to.
* `host` - (Optional) The host header to send to the backend.

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
