# Header Resource

Manages a Quant custom headers.

## Example Usage

```hcl
resource "quant_header" "test" {
  project = "default"
  headers = {
    "X-Quant-Header" = "test"
  }
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `headers` - (Required) The headers to set.

## Import

Header resources can be imported using the project machine name as the import ID:

```bash
terraform import quant_header.resource_name "project_name"
```

### Example

```bash
terraform import quant_header.test "my-project"
```

### Import Format

The import ID is simply the project machine name where the headers are configured.

### Step-by-Step Import Process

1. **Add the resource to your Terraform configuration**:
   ```hcl
   resource "quant_header" "test" {
     project = "my-project"
     # Headers will be populated from the import
   }
   ```

2. **Run the import command**:
   ```bash
   terraform import quant_header.test "my-project"
   ```

3. **Verify the import**:
   ```bash
   terraform plan
   ```

### How Import Works

When you import a header resource:
- The provider reads all existing headers for the specified project
- The headers are populated in the Terraform state
- You can then modify the configuration as needed

### Important Notes

- Header resources are project-scoped (one header configuration per project)
- Importing will read all existing headers for the project
- You can modify the headers after import by updating the configuration

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` - A computed ID based on the header values (SHA-256 hash).
