# KV Index Resource

Manages a secondary index of a format 2 KV store (`quant_kv_store` with `format = 2`).

An index definition cannot change. Every input forces a replacement: Terraform drops the index, waits until it is gone, then declares it again.

## Example Usage

```hcl
resource "quant_kv_store" "orders" {
  project = "my-project"
  name    = "orders"
  format  = 2
}

resource "quant_kv_index" "orders_by_customer" {
  project    = "my-project"
  store_id   = quant_kv_store.orders.store_id
  name       = "byCustomer"
  collection = "order"
  fields     = ["customerId"]
  sort_by    = "createdAt"
}

resource "quant_kv_index" "orders_by_status" {
  project    = "my-project"
  store_id   = quant_kv_store.orders.store_id
  name       = "byStatus"
  collection = "order"
  fields     = ["status"]
  live_count = true
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `store_id` - (Required) The KV store ID. The store must be format 2.
- `name` - (Required) The index name, unique in the store. Pattern `^[A-Za-z][A-Za-z0-9_]{0,63}$`.
- `collection` - (Required) A collection, or a pattern where `*` as one whole segment matches exactly one segment (`customer:*:world`). At most 512 bytes.
- `fields` - (Optional) Top-level fields of a plaintext JSON object value (0 to 3). A unique index needs at least one. A field that a ledger on the collection marks as personal cannot be indexed.
- `sort_by` - (Optional) One sort field.
- `unique` - (Optional) One key per value. Enables the index lookup. Default `false`. Not allowed on a ledger collection.
- `live_count` - (Optional) Keeps a live count per value. Enables the index count. Default `false`. Not allowed on a ledger collection or on items with a ttl. The API field is `count`; Terraform reserves the name `count`.
- `organization` - (Optional) The organization. Defaults to the provider organization.

Every argument forces a replacement.

## Attribute Reference

- `state` - `ACTIVE` after a successful create.
- `slot` - The secondary index slot that serves the index.
- `created_at` - When the index was declared.
- `error` - `FAILED` only: the code of the first problem the build found.
- `conflicts` - `FAILED` only: up to 20 keys that caused the failure.
- `index_name` - The same value as `name`.

## Create and Delete

- Create declares the index (the API answers `202` with a `BUILDING` index) and waits until the build ends. The default wait is 10 minutes. Set the environment variable `QUANTCDN_KV_INDEX_TIMEOUT` (a Go duration, for example `30m`) to wait longer.
- If the build ends `FAILED`, the apply fails with the error code and the conflicting keys. The index stays in state as tainted, and the next apply drops it and declares it again.
- Delete drops the index and waits until it is gone. An index that is already gone (`404`) is not an error.
- A refresh removes an index that is gone from state. An index that is `DROPPING` stays in state until it is gone. The next plan replaces it (with a warning): the apply waits until the drop ends, without a second drop, and then declares the index again.
- An index in state that is `FAILED`, also one just imported, is replaced by the next plan (with a warning): the apply drops it and declares it again.

## Rate limit

The KV API allows 10 index and ledger changes (declare, raise, drop) per minute per project. Above that, it answers `429` with a `Retry-After` header. The provider then waits the full `Retry-After` and sends the declare or the drop again, until the wait timeout (`QUANTCDN_KV_INDEX_TIMEOUT`, default 10 minutes) ends. To declare many indexes in one apply, expect the apply to wait, or run `terraform apply -parallelism=` with a lower value.

## Import

```bash
terraform import quant_kv_index.orders_by_customer "my-project/store-id/byCustomer"
```

The import ID is `project/store_id/name`, or `organization/project/store_id/name`.
