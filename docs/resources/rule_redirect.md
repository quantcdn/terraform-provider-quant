# Rule Redirect Resource

Manages a Quant redirect rule.

## Example Usage

```hcl
resource "quant_rule_redirect" "test" {
  name = "test-redirect"
  project = quant_project.test.machine_name
  domain = ["any"]
  url = ["/redirect"]
  redirect_to = "https://backend.example.com"
  redirect_code = 301
}
```

## Argument Reference

- `name` - (Required) The name of the rule.
- `project` - (Required) The machine name of the project.
- `domain` - (Required) The domain to apply the rule to.
- `redirect_to` - (Required) The target URL to redirect requests to.
- `redirect_code` - (Required) The status code to use for the redirect.


## Import

Redirect rules can be imported using the following format:

```bash
terraform import quant_rule_redirect.resource_name "project_name/rule_uuid"
```

### Example

```bash
terraform import quant_rule_redirect.test "my-project/12345678-1234-1234-1234-123456789abc"
```

### Import Format

The import ID must follow the pattern `project_name/rule_uuid` where:
- `project_name` is the machine name of your project (not the display name)
- `rule_uuid` is the UUID of the redirect rule in format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

### Finding the Rule UUID

To find the rule UUID, you can:
1. Use the QuantCDN dashboard - Look in the rules section for your redirect rule
2. Use the API directly - Call the rules list endpoint
3. Check existing Terraform state - If you have other rules already managed

### Step-by-Step Import Process

1. **Add the resource to your Terraform configuration**:
   ```hcl
   resource "quant_rule_redirect" "test" {
     name = "test-redirect"
     project = "my-project"
     # Other required fields will be populated from the import
   }
   ```

2. **Run the import command**:
   ```bash
   terraform import quant_rule_redirect.test "my-project/12345678-1234-1234-1234-123456789abc"
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

## Attributes Reference

- `uuid` - The UUID of the rule.
- `rule_id` - The ID of the rule.
- `organization` - The organization ID this rule belongs to.
- `weight` - The weight of the rule.
- `action` - The action type, always set to `redirect`.
- `only_with_cookie` - The cookie condition for the rule.
