# Project Data Source

Fetches details for a specific Quant project, including write tokens for use in other providers.

## Example Usage

### Basic Usage

```hcl
data "quant_project" "example" {
  machine_name = "my-project"
}

# Access project details
output "project_name" {
  value = data.quant_project.example.name
}

output "project_uuid" {
  value = data.quant_project.example.uuid
}
```

### With Write Token

```hcl
data "quant_project" "example" {
  machine_name = "my-project"
  with_token   = true
}

# Use write token in GitHub Actions
resource "github_actions_secret" "quant_token" {
  repository      = "my-org/my-repo"
  secret_name     = "QUANT_TOKEN"
  plaintext_value = data.quant_project.example.write_token
}
```

### Module Pattern

```hcl
# Create a reusable module for fetching credentials
# Module: modules/quant-credentials/main.tf
data "quant_project" "project" {
  machine_name = var.project_name
  with_token   = true
}

output "token" {
  value     = data.quant_project.project.write_token
  sensitive = true
}

# Usage in main configuration
module "quant_credentials" {
  source = "./modules/quant-credentials"
  
  project_name = "my-project"
}

resource "aws_ssm_parameter" "quant_token" {
  name  = "/app/quant/write_token"
  type  = "SecureString"
  value = module.quant_credentials.token
}
```

## Argument Reference

- `machine_name` - (Required) The machine name of the project to fetch details for.
- `with_token` - (Optional) Whether to include the write token in the response. Defaults to `true` for data sources since the primary use case is credential fetching.

## Attributes Reference

- `id` - The numeric ID of the project.
- `uuid` - The UUID of the project.
- `name` - The display name of the project.
- `machine_name` - The machine name of the project.
- `write_token` - (Sensitive) The project write token for API access. Only populated when `with_token = true`.
- `with_token` - Whether the write token was requested and included in the response.

## Notes

- The `write_token` attribute is marked as sensitive and will not be displayed in Terraform output by default.
- When `with_token = false`, the `write_token` attribute will be empty.
- This data source is particularly useful for integrating Quant projects with other providers such as GitHub, AWS, or Azure for credential management.
- The data source respects the provider's rate limiting configuration and will automatically retry failed requests using exponential backoff. 