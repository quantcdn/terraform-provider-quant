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
  proxy = {
    to      = "https://backend.example.com"
    host    = "backend.example.com"
  }
  failover = {
    failover_mode = "true"
    failover_lifetime = "1h"
    failover_origin_status_codes = ["200", "201"]
  }
  waf_enabled = true
  waf_config = {
    mode           = "report"
    paranoia_level = 1
    allow_rules    = []
    block_ip       = []
    block_ua       = []
    block_referer  = []
    notify_email   = []
    httpbl = {
      enabled = false
    }
  }
  notify = "slack"
  notify_config = {
    origin_status_codes = ["200", "201"]
    period = "60"
    slack_webhook = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
  }
  thresholds = [{
    cooldown = 60
    hits = 10
    minutes = 1
    mode = "block"
    notify_slack = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
    rps = 1000
    type = "block"
    value = "192.168.1.1"
  }]
}
```

## Rule Selection Criteria

The following parameters are used to inspect the request and determine if the rule should be applied.

- `country` - Should be `country_is` or `country_is_not`. This tells the rules engine which variable to inspect when applying the rule.
- `country_is` - A list of country codes to match.
- `country_is_not` - A list of country codes to not match.
- `ip` - Should be `ip_is` or `ip_is_not`. This tells the rules engine which variable to inspect when applying the rule.
- `ip_is` - A list of IP addresses to match.
- `ip_is_not` - A list of IP addresses to not match.
- `method` - AnyOf `method_is` or `method_is_not`. This tells the rules engine which variable to inspect when applying the rule.
- `method_is` - A list of HTTP methods to match.
- `method_is_not` - A list of HTTP methods to not match.

## Argument Reference

- `name` - (Required) The name of the rule.
- `project` - (Required) The machine name of the project.
- `domain` - (Required) The domain to apply the rule to.
- `to` - (Required) The target URL to proxy requests to.
- `host` - (Required) The host to apply the rule to.
- `waf_enabled` - (Required) Whether to enable WAF for the rule.
- `waf_config` - (Required) The WAF configuration for the rule.

## Attributes Reference

- `id` - The ID of the rule.
