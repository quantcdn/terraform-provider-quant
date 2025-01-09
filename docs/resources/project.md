# Project Resource

Manages a Quant project.

## Example Usage

```hcl
resource "quant_project" "example" {
  # Required
  name = "My Project Name"

  # Optional
  region            = "au"  # Defaults to "au" if not specified
  allow_query_params = true

  # Optional - Basic Authentication
  # Note: If setting basic auth, both username and password must be provided
  basic_auth_username     = "admin"
  basic_auth_password     = "secret123"
  basic_auth_preview_only = "enabled"  # Optional basic auth setting
}
```

## Argument Reference

- `name` - (Required) The name of the project.
- `region` - (Optional) The region where the project is hosted. Defaults to "au".
- `allow_query_params` - (Optional) Whether to allow query parameters in URLs. Defaults to false.
- `basic_auth_username` - (Optional) The username for basic authentication. Must be provided together with `basic_auth_password` if basic auth is desired.
- `basic_auth_password` - (Optional) The password for basic authentication. Must be provided together with `basic_auth_username` if basic auth is desired.
- `basic_auth_preview_only` - (Optional) Whether basic authentication applies only to preview environments. Valid values are "enabled" or "disabled".


## Attributes Reference

- `id` - The ID of the project.
- `machine_name` - The machine-readable name of the project.
- `created_at` - The timestamp when the project was created.
- `deleted_at` - The timestamp when the project was deleted, if applicable.
- `organization` - The organization that owns the project.
- `parent_project_id` - The ID of the parent project, if this is a child project.
- `project` - The project identifier.
- `custom_s3_sync_access_key` - The access key for custom S3 sync configuration, if configured.
- `custom_s3_sync_secret_key` - The secret key for custom S3 sync configuration, if configured.
- `custom_s3_sync_region` - The region for custom S3 sync configuration, if configured.
- `custom_s3_sync_bucket` - The bucket name for custom S3 sync configuration, if configured.
