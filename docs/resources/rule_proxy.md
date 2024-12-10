# Rule Proxy Resource

Manages a Quant proxy rule.

## Example Usage

```hcl
resource "quant_rule_proxy" "test" {
  name    = "test-proxy"
  project = quant_project.test.machine_name
  domain  = ["any"]
  url     = ["/proxy"]
  proxy = {
    to      = "https://backend.example.com"
    host    = "backend.example.com"
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
}
```

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
