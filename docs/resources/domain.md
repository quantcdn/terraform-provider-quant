# Domain Resource

The `quant_domain` resource manages domains within a Quant project.

## Example Usage

```hcl
provider "quant" {
  organization = "test-organization"
  bearer       = "testtoken"
}

resource "quant_domain" "test" {
  domain  = "example.com"
  project = "default"
}
```

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The domain name to manage.
* `project` - (Optional) The project machine name to create the domain in. Defaults to "default".

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the domain.
* `created_at` - The timestamp when the domain was created.
* `updated_at` - The timestamp when the domain was last updated.
* `deleted_at` - The timestamp when the domain was deleted, if applicable.
* `dns_engaged` - Whether DNS is engaged for this domain.
* `in_section` - Whether the domain is in a section.
* `project_id` - The ID of the project the domain belongs to.
* `section_message` - Any section-related message for the domain.
* `organization` - The organization the domain belongs to.

## Import

Domain resources can be imported using the following format:

```bash
terraform import quant_domain.resource_name "project_name/domain_id"
```

### Example

```bash
terraform import quant_domain.test "my-project/9555"
```

### Import Format

The import ID must follow the pattern `project_name/domain_id` where:
- `project_name` is the machine name of your project (not the display name)
- `domain_id` is the numeric ID of the domain

### Finding the Domain ID

To find the domain ID, you can:
1. Use the QuantCDN dashboard - Look in the domains section for your domain
2. Use the API directly - Call the domains list endpoint
3. Check existing Terraform state - If you have other domains already managed

### Step-by-Step Import Process

1. **Add the resource to your Terraform configuration**:
   ```hcl
   resource "quant_domain" "test" {
     domain  = "example.com"
     project = "my-project"
   }
   ```

2. **Run the import command**:
   ```bash
   terraform import quant_domain.test "my-project/9555"
   ```

3. **Verify the import**:
   ```bash
   terraform plan
   ```

### Troubleshooting

- **Invalid Import ID**: Ensure the format is exactly `project_name/domain_id`
- **Domain not found**: Verify the domain ID exists in the specified project
- **Project not found**: Ensure the project machine name is correct 