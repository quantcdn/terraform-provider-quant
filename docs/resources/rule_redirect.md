# Rule Redirect Resource

Manages a Quant redirect rule.

## Example Usage

```hcl
resource "quant_rule_redirect" "test" {
  name = "test-rule"
  project = "default"
  domain = ["example.com"]
  to = "https://example.com"
  status_code = 301
}
```

## Argument Reference

- `name` - (Required) The name of the rule.
- `project` - (Required) The name of the project.
- `domain` - (Required) The domain to apply the rule to.
- `to` - (Required) The target URL to redirect requests to.
- `status_code` - (Required) The status code to use for the redirect.


## Attributes Reference

- `id` - The ID of the rule.
