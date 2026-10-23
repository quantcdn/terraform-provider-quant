---
page_title: "Quant: quant_rule_auth"
subcategory: "Rules"
description: |-
  Manages a Quant authentication rule. The rule requires HTTP basic authentication for matching requests.
---

# Rule Auth Resource

Manages a Quant authentication rule. The rule requires HTTP basic authentication for matching requests.

## Example Usage

```hcl
provider "quant" {
  bearer       = var.quant_api_token
  organization = "example-org"
  # AU Government customers: uncomment to use the QuantGov dashboard.
  # base_url   = "https://dash.quantgov.cloud"
}
```

```hcl
resource "quant_rule_auth" "example" {
  name      = "protect-staging"
  project   = "example-project"
  domain    = ["any"]
  url       = ["/*"]
  auth_user = "example-user"
  auth_pass = var.basic_auth_password
}
```

## Argument Reference

### Required

* `auth_pass` (String) - (Required) Authentication password.
* `auth_user` (String) - (Required) Authentication username.
* `domain` (List of String) - (Required) Domain patterns (default: any).
* `url` (List of String) - (Required) URL patterns.

### Optional

#### Rule Selection Criteria

* `asn` (String) - (Optional) ASN filter type (asn_is, asn_is_not, any).
* `asn_is` (List of String) - (Optional) Allowed AS numbers.
* `asn_is_not` (List of String) - (Optional) Excluded AS numbers.
* `country` (String) - (Optional) Country filter type (country_is, country_is_not, any).
* `country_is` (List of String) - (Optional) Allowed countries.
* `country_is_not` (List of String) - (Optional) Excluded countries.
* `disabled` (Boolean) - (Optional) Whether rule is disabled. Defaults to `false`.
* `ip` (String) - (Optional) IP filter type (ip_is, ip_is_not, any).
* `ip_is` (List of String) - (Optional) Allowed IP addresses.
* `ip_is_not` (List of String) - (Optional) Excluded IP addresses.
* `method` (String) - (Optional) Method filter type (method_is, method_is_not, any).
* `method_is` (List of String) - (Optional) Allowed HTTP methods.
* `method_is_not` (List of String) - (Optional) Excluded HTTP methods.
* `name` (String) - (Optional, Computed) Rule name. Computed from the API when omitted; always set it explicitly.
* `organization` (String) - (Optional) Organization identifier.
* `project` (String) - (Optional, Computed) Project identifier. Computed from the API when omitted; always set it explicitly.
* `weight` (Number) - (Optional) Rule weight. Defaults to `0`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `action` (String) - (Read-Only) Rule action.
* `action_config` (Object) - (Read-Only) Action configuration as returned by the API.
* `only_with_cookie` (String) - (Read-Only) Only apply with cookie.
* `rule` (String) - (Read-Only) Rule identifier.
* `rule_id` (String) - (Read-Only) Rule ID.
* `uuid` (String) - (Read-Only) Rule UUID.

## Import

Rules can be imported with the ID `project_machine_name/rule_uuid`. The UUID must be a valid UUID.

```bash
terraform import quant_rule_auth.example "example-project/12345678-1234-1234-1234-123456789abc"
```
