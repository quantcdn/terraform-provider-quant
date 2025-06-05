# Rule Custom Response Resource

The `quant_rule_custom_response` resource manages custom responses for rules in a Quant project.

## Example Usage

```hcl
provider "quant" {
  organization = "test-organization"
  bearer       = "testtoken"
}

resource "quant_rule_custom_response" "test" {
  project = "test-project"
  name    = "test-response"
  domain  = ["example.com"]
  url     = ["/test"]
  custom_response_status_code = 404
  custom_response_body        = "Not Found"
}
```

## Argument Reference

The following arguments are supported:

* `project` - (Required) The project ID this rule belongs to.
* `name` - (Required) The name of the rule.
* `domain` - (Required) List of domains this rule applies to.
* `url` - (Required) List of URLs this rule applies to.
* `custom_response_status_code` - (Required) The HTTP status code for the custom response.
* `custom_response_body` - (Required) The response body content.
* `country` - (Optional) The country condition type. Can be `country_is` or `country_is_not`.
* `country_is` - (Optional) List of countries to match when `country` is set to `country_is`.
* `country_is_not` - (Optional) List of countries to match when `country` is set to `country_is_not`.
* `method` - (Optional) The HTTP method condition type. Can be `method_is` or `method_is_not`.
* `method_is` - (Optional) List of HTTP methods to match when `method` is set to `method_is`.
* `method_is_not` - (Optional) List of HTTP methods to match when `method` is set to `method_is_not`.
* `ip` - (Optional) The IP condition type. Can be `ip_is` or `ip_is_not`.
* `ip_is` - (Optional) List of IPs to match when `ip` is set to `ip_is`.
* `ip_is_not` - (Optional) List of IPs to match when `ip` is set to `ip_is_not`.
* `disabled` - (Optional) Whether the rule is disabled. Defaults to `false`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `uuid` - The UUID of the rule.
* `rule_id` - The ID of the rule.
* `organization` - The organization ID this rule belongs to.
* `weight` - The weight of the rule.
* `action` - The action type, always set to `custom_response`.
* `only_with_cookie` - The cookie condition for the rule.

## Import

Custom response rules can be imported using their project and rule ID:

```bash
$ terraform import quant_rule_custom_response.test project_id/rule_id
``` 