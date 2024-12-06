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
