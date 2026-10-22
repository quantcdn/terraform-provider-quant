---
page_title: "Quant: quant_ai_custom_tool"
subcategory: "AI"
description: |-
  Manages a custom AI tool. The tool runs JavaScript code as an edge function that AI agents can call.
---

# AI Custom Tool Resource

Manages a custom AI tool. The tool runs JavaScript code as an edge function that AI agents can call.

## Example Usage

```hcl
provider "quant" {
  bearer       = var.quant_api_token
  organization = "example-org"
  # AU Government customers: uncomment to use the QuantGov dashboard.
  # base_url   = "https://dash.quantgov.cloud"
}
```

```hcl
resource "quant_ai_custom_tool" "lookup" {
  name        = "lookup_order"
  description = "Look up an order by its number."
  category    = "orders"

  input_schema = jsonencode({
    type       = "object"
    properties = { order_number = { type = "string" } }
    required   = ["order_number"]
  })

  edge_function_code = <<-EOT
    export default async function handler(input) {
      return { status: "shipped", order: input.order_number };
    }
  EOT

  timeout_seconds = 10
}
```

## Argument Reference

### Required

* `description` (String) - (Required) Human-readable description of what the tool does.
* `edge_function_code` (String) - (Required) JavaScript source code for the edge function.
* `input_schema` (String) - (Required) JSON-encoded JSON Schema object defining the tool's input parameters.
* `name` (String) - (Required) Unique tool name (alphanumeric and underscores only).

### Optional

* `category` (String) - (Optional) Category to group related tools.
* `is_async` (Boolean) - (Optional) Whether this tool runs asynchronously (>5 seconds). Defaults to `false`.
* `organisation` (String) - (Optional) The organisation ID.
* `output_schema` (String) - (Optional) JSON-encoded JSON Schema object defining the tool's output structure.
* `output_schema_description` (String) - (Optional) Human-readable description of the tool's output.
* `response_mode` (String) - (Optional) How the tool response is handled: llm (passed back to model) or direct (returned to user). Valid values: `llm`, `direct`.
* `timeout_seconds` (Number) - (Optional) Tool execution timeout. Range: 5 to 300. Defaults to `30`.
* `tool_name` (String) - (Optional) The tool name to retrieve.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `edge_function_url` (String) - (Read-Only) Computed edge function URL (read-only).
* `is_update` (Boolean) - (Read-Only) Whether this was an update to an existing tool.
* `message` (String) - (Read-Only) Status message returned by the last API operation.
* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.
* `tool` (Object) - (Read-Only) Tool details as returned by the API.

## Import

Import a tool with its name.

```bash
terraform import quant_ai_custom_tool.lookup lookup_order
```
