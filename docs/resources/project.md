# Project Resource

Manages a Quant project.

## Example Usage

### Basic Project

```hcl
resource "quant_project" "example" {
  name = "My Project"
}
```

### Project with Configuration

```hcl
resource "quant_project" "configured" {
  name               = "My Configured Project"
  region             = "au"
  allow_query_params = true
  disable_revisions  = false
}
```

### Project with Basic Authentication

```hcl
resource "quant_project" "with_auth" {
  name = "Protected Project"
  
  # Basic authentication
  basic_auth_username      = "admin"
  basic_auth_password      = "secure-password-123"
  basic_auth_preview_only  = true  # Only apply to preview domain
}
```

## Argument Reference

### Required Arguments

- `name` - (Required) The display name of the project.

### Optional Arguments

- `machine_name` - (Optional, Computed) The machine-readable name of the project. If not provided, will be generated from the project name.
- `region` - (Optional, Computed) The region where the project is hosted (e.g., "au", "us"). 
- `allow_query_params` - (Optional, Computed) Whether to allow query parameters in URLs. Defaults to `false`.
- `disable_revisions` - (Optional, Computed) Whether to disable content revisions. Defaults to `false`.

#### Basic Authentication

- `basic_auth_username` - (Optional, Computed) The username for basic authentication. Must be provided together with `basic_auth_password`.
- `basic_auth_password` - (Optional, Computed) The password for basic authentication. Must be provided together with `basic_auth_username`.
- `basic_auth_preview_only` - (Optional, Computed) Whether basic authentication applies only to preview domains. Defaults to `false`.

#### Advanced Options

- `with_token` - (Optional) Whether to include the write token in the response. Defaults to `false`. Typically used with data sources rather than resources.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The numeric ID of the project.
- `uuid` - The UUID of the project.
- `write_token` - The write token for API access (only populated when `with_token = true`).
- `organization` - The organization identifier.
- `project` - The project identifier.

## Import

Projects can be imported using the machine name:

```shell
terraform import quant_project.example my-project-name
```

## Notes

- The `machine_name` is automatically generated from the `name` if not explicitly provided.
- Basic auth credentials (`basic_auth_username` and `basic_auth_password`) must be provided together or not at all.
- The `write_token` attribute is sensitive and will not be displayed in Terraform output by default.
- The `region` field is optional and will use the API's default if not specified.
