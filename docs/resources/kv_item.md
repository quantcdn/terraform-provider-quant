---
page_title: "Quant: quant_kv_item"
subcategory: "Key-Value Store"
description: |-
  Manages one item in a Quant key-value store.
---

# KV Item Resource

Manages one item in a Quant key-value store.

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
resource "quant_kv_store" "example" {
  project = "example-project"
  name    = "example-store"
}

resource "quant_kv_item" "config" {
  project  = "example-project"
  store_id = quant_kv_store.example.id
  key      = "feature_flag"
  value    = "enabled"
}

resource "quant_kv_item" "api_key" {
  project  = "example-project"
  store_id = quant_kv_store.example.id
  key      = "upstream_api_key"
  value    = var.upstream_api_key
  secret   = true
}
```

## Argument Reference

### Required

* `key` (String) - (Required) Item key.
* `value` (String) - (Required) Item value (can be JSON string).

### Optional

* `organization` (String) - (Optional) Organization identifier.
* `project` (String) - (Optional) Project identifier.
* `secret` (Boolean) - (Optional) Store as secret with KMS encryption. Secrets cannot be retrieved via GET operations (returns [ENCRYPTED]). Ideal for API keys, passwords, and credentials. Defaults to `false`.
* `store_id` (String) - (Optional) The ID of the key-value store.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.

## Notes

If `secret` is `true`, the platform encrypts the value. A read operation returns `[ENCRYPTED]` and not the value.

## Import

Import an item with the ID `project/store_id/key`.

```bash
terraform import quant_kv_item.config "example-project/example-store-id/feature_flag"
```
