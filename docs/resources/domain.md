# Domain Resource

Manages domains within a Quant project.

## Example Usage

### Basic Domain

```hcl
resource "quant_domain" "example" {
  domain  = "example.com"
  project = "my-project"
}
```

### Multiple Domains

```hcl
resource "quant_project" "website" {
  name = "My Website"
}

resource "quant_domain" "main" {
  domain  = "example.com"
  project = quant_project.website.machine_name
}

resource "quant_domain" "www" {
  domain  = "www.example.com"
  project = quant_project.website.machine_name
}

resource "quant_domain" "staging" {
  domain  = "staging.example.com"
  project = quant_project.website.machine_name
}
```

## Argument Reference

### Required Arguments

- `domain` - (Required) The domain name to manage (e.g., "example.com", "www.example.com").

### Optional Arguments

- `project` - (Optional, Computed) The project machine name to create the domain in. If not specified, uses the provider's default project.
- `organization` - (Optional, Computed) The organization identifier. Typically inherited from the provider configuration.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The numeric ID of the domain.
- `dns_engaged` - DNS engagement status (0 = not engaged, 1 = engaged).

## Import

Domains can be imported using the format `project_name/domain_id`:

```shell
terraform import quant_domain.example my-project/9555
```

### Import Format

The import ID must follow the pattern `project_name/domain_id` where:
- `project_name` is the machine name of your project (not the display name)
- `domain_id` is the numeric ID of the domain

### Finding the Domain ID

To find the domain ID:
1. Use the QuantCDN dashboard - Navigate to the domains section
2. Use the API directly - Call the domains list endpoint
3. Check existing Terraform state - If you have other domains already managed

### Step-by-Step Import Process

1. **Add the resource to your Terraform configuration**:
   ```hcl
   resource "quant_domain" "example" {
     domain  = "example.com"
     project = "my-project"
   }
   ```

2. **Run the import command**:
   ```shell
   terraform import quant_domain.example "my-project/9555"
   ```

3. **Verify the import**:
   ```shell
   terraform plan
   ```

### Troubleshooting

- **Invalid Import ID**: Ensure the format is exactly `project_name/domain_id`
- **Domain not found**: Verify the domain ID exists in the specified project
- **Project not found**: Ensure the project machine name is correct

## Notes

- Domain updates are not supported by the V2 API. To change a domain, you must delete and recreate it.
- The `dns_engaged` field indicates whether DNS management is active for the domain.
- Domains must be verified before they can be used for content delivery.
