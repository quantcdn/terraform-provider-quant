---
page_title: "Quant: quant_rule_bot_challenge"
subcategory: "Rules"
description: |-
  Manages a Quant bot challenge rule. The rule shows a challenge to matching visitors before it serves content.
---

# Rule Bot Challenge Resource

Manages a Quant bot challenge rule. The rule shows a challenge to matching visitors before it serves content.

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
resource "quant_rule_bot_challenge" "example" {
  name                             = "challenge-search"
  project                          = "example-project"
  domain                           = ["any"]
  url                              = ["/search*"]
  robot_challenge_type             = "invisible"
  robot_challenge_challenge_ttl    = 300
  robot_challenge_verification_ttl = 3600
}
```

## Argument Reference

### Required

* `domain` (List of String) - (Required) Domain patterns (default: any).
* `robot_challenge_type` (String) - (Required) Challenge type (invisible or checkbox).
* `url` (List of String) - (Required) URL patterns.

### Optional

* `robot_challenge_challenge_ttl` (Number) - (Optional) Challenge TTL in seconds. Defaults to `30`.
* `robot_challenge_verification_ttl` (Number) - (Optional) Verification TTL in seconds. Defaults to `10800`.

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
* `name` (String) - (Optional) Rule name.
* `organization` (String) - (Optional) Organization identifier.
* `project` (String) - (Optional) Project identifier.
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
terraform import quant_rule_bot_challenge.example "example-project/12345678-1234-1234-1234-123456789abc"
```
