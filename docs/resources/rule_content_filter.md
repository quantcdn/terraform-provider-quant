# Rule Content Filter Resource

The `quant_rule_content_filter` resource manages content filter rules that apply custom functions to filter and transform content in a Quant project.

## Example Usage

### Basic Content Filter Rule

```hcl
provider "quant" {
  organization = "test-organization"
  bearer       = "testtoken"
}

resource "quant_rule_content_filter" "basic" {
  project = "test-project"
  name    = "api-content-filter"
  domain  = ["any"]
  url     = ["/api/*"]
  
  fn_uuid  = "function-uuid-12345"
  disabled = false
}
```

### Content Filter with Conditional Matching

```hcl
resource "quant_rule_content_filter" "conditional" {
  project = "test-project"
  name    = "geo-content-filter"
  domain  = ["example.com"]
  url     = ["/content/*"]
  
  fn_uuid  = "function-uuid-67890"
  disabled = false
  
  # Apply only to specific countries
  country    = "country_is"
  country_is = ["US", "CA", "GB"]
  
  # Apply only to specific HTTP methods
  method    = "method_is"
  method_is = ["GET", "POST"]
}
```

### Content Filter with IP Restrictions

```hcl
resource "quant_rule_content_filter" "ip_restricted" {
  project = "test-project"
  name    = "admin-content-filter"
  domain  = ["admin.example.com"]
  url     = ["/admin/*"]
  
  fn_uuid  = "function-uuid-admin"
  disabled = false
  
  # Apply only to specific IP addresses
  ip    = "ip_is"
  ip_is = ["192.168.1.100", "10.0.0.50"]
}
```

### Disabled Content Filter Rule

```hcl
resource "quant_rule_content_filter" "disabled" {
  project = "test-project"
  name    = "maintenance-filter"
  domain  = ["any"]
  url     = ["/maintenance"]
  
  fn_uuid  = "function-uuid-maintenance"
  disabled = true  # Rule is disabled
}
```

## Argument Reference

The following arguments are supported:

### Required Arguments

* `name` - (Required) The name of the content filter rule.
* `project` - (Required) The project machine name this rule belongs to.
* `domain` - (Required) List of domains this rule applies to. Use `["any"]` to match all domains.
* `url` - (Required) List of URL patterns this rule applies to. Supports wildcard patterns like `/api/*`.
* `fn_uuid` - (Required) The UUID of the function to apply for content filtering.

### Optional Arguments

* `disabled` - (Optional) Whether the content filter rule is disabled. Defaults to `false`.

### Rule Selection Criteria

* `country` - (Optional) The country condition type. One of `country_is`, `country_is_not`, or `any`.
* `country_is` - (Optional) List of country codes to match when `country` is set to `country_is`.
* `country_is_not` - (Optional) List of country codes to exclude when `country` is set to `country_is_not`.
* `method` - (Optional) The HTTP method condition type. One of `method_is`, `method_is_not`, or `any`.
* `method_is` - (Optional) List of HTTP methods to match when `method` is set to `method_is`.
* `method_is_not` - (Optional) List of HTTP methods to exclude when `method` is set to `method_is_not`.
* `ip` - (Optional) The IP address condition type. One of `ip_is`, `ip_is_not`, or `any`.
* `ip_is` - (Optional) List of IP addresses to match when `ip` is set to `ip_is`.
* `ip_is_not` - (Optional) List of IP addresses to exclude when `ip` is set to `ip_is_not`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `uuid` - The UUID of the rule.
* `rule_id` - The unique identifier of the rule.
* `organization` - The organization this rule belongs to.
* `action` - The action type, always set to `content_filter`.
* `weight` - The weight of the rule for ordering purposes.
* `only_with_cookie` - The cookie condition for the rule (if any).

## Import

Content filter rules can be imported using the following format:

```bash
terraform import quant_rule_content_filter.resource_name "project_name/rule_uuid"
```

### Example

```bash
terraform import quant_rule_content_filter.api_filter "my-project/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0"
```

### Import Format

The import ID must follow the pattern `project_name/rule_uuid` where:
- `project_name` is the machine name of your project (not the display name)
- `rule_uuid` is the UUID of the content filter rule in format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

### Finding the Rule UUID

To find the rule UUID, you can:
1. Use the QuantCDN dashboard - Look in the rules section for your content filter rule
2. Use the API directly - Call the rules list endpoint
3. Check existing Terraform state - If you have other rules already managed

### Step-by-Step Import Process

1. **Add the resource to your Terraform configuration**:
   ```hcl
   resource "quant_rule_content_filter" "api_filter" {
     name    = "api-content-filter"
     project = "my-project"
     # Other required fields will be populated from the import
   }
   ```

2. **Run the import command**:
   ```bash
   terraform import quant_rule_content_filter.api_filter "my-project/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0"
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