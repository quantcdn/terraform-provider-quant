# Crawler Resource

Provides a crawler resource to manage crawlers in Quant.

## Example Usage

```hcl
resource "quant_crawler" "crawler" {
    project = quant_project.test.machine_name
    name = "test-crawler"
    domain = "example.com"
    urls = ["/"]
    browser_mode = false
    exclude = ["/admin"]
    headers = {
        "X-Header" = "value"
    }
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `name` - (Required) The name of the crawler.
- `domain` - (Required) The domain to apply the crawler to.
- `urls` - (Required) The URLs to apply the crawler to.
- `browser_mode` - (Required) Whether to use browser mode.
- `exclude` - (Required) The URLs to exclude from the crawler.
