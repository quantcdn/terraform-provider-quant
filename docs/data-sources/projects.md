# Projects Data Source

Lists all projects in the Quant organization.

## Example Usage

### Basic Usage

```hcl
data "quant_projects" "all" {}

# Output all project names
output "project_names" {
  value = [for project in data.quant_projects.all.projects : project.name]
}

# Output all project machine names
output "project_machine_names" {
  value = [for project in data.quant_projects.all.projects : project.machine_name]
}
```

### Using with for_each

```hcl
data "quant_projects" "all" {}

# Create a domain for each project
resource "quant_domain" "project_domains" {
  for_each = {
    for project in data.quant_projects.all.projects : 
    project.machine_name => project.name
  }
  
  domain  = "${each.key}.example.com"
  project = each.key
}
```

### Filtering Projects

```hcl
data "quant_projects" "all" {}

locals {
  # Filter projects by naming pattern
  production_projects = [
    for project in data.quant_projects.all.projects : project
    if startswith(project.name, "prod-")
  ]
  
  # Get machine names only
  prod_machine_names = [
    for project in local.production_projects : project.machine_name
  ]
}

output "production_projects" {
  value = local.production_projects
}
```

## Argument Reference

This data source takes no arguments.

## Attributes Reference

- `projects` - A list of projects in the organization. Each project has the following attributes:
  - `name` - The display name of the project.
  - `machine_name` - The machine-readable name of the project.

## Notes

- This data source fetches all projects accessible to the authenticated user within the organization.
- The data source respects the provider's rate limiting configuration and will automatically retry failed requests using exponential backoff.
- For fetching detailed information about a specific project including write tokens, use the `quant_project` data source instead. 