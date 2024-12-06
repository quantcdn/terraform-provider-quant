# Header Resource

Manages a Quant custom headers.

## Example Usage

```hcl
resource "quant_header" "test" {
  name = "test-header"
  project = "default"
  headers = {
    "X-Quant-Header" = "test"
  }
}
```

## Argument Reference

- `name` - (Required) The name of the header.
- `project` - (Required) The name of the project.
- `headers` - (Required) The headers to set.
