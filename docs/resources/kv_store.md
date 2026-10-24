# KV Store Resource

Manages a Quant KV store in a project.

## Example Usage

```hcl
resource "quant_kv_store" "audit" {
  project = "my-project"
  name    = "audit"
  format  = 2
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `name` - (Required) The store name. A change replaces the store.
- `organization` - (Optional) The organization. Defaults to the provider organization. A change replaces the store.
- `format` - (Optional) The store format, `1` or `2`. Default `1`. Format 2 enables indexes (`quant_kv_index`), ledgers (`quant_kv_ledger`), conditional writes, transactions and the cursor key list.

## Attribute Reference

- `store_id` - The store ID.
- `id` - The store ID.

## Format

The format is set when the store is created. Terraform never changes the format of an existing store:

- A plan that changes `format` on an existing store fails with `KV store format cannot change`. It does not replace the store, because a replacement deletes the data.
- To upgrade a format 1 store, use the upgrade API (`POST /api/v2/organizations/{organization}/projects/{project}/kv/{store_id}/upgrade`). When the upgrade is complete, set `format = 2`. The next plan shows no change.
- Terraform reads the real format from `GET .../kv/{store_id}/upgrade` on every refresh, so an upgrade outside Terraform is visible in the plan. If that endpoint answers `404`, the store is format 1. If it fails (for example `403` or `5xx`), the refresh keeps the format from state and gives a warning.
- The API sets `store_id`. A plan that sets `store_id` to another value fails with `KV store store_id cannot change`; it does not move the resource to another store.

## Import

```bash
terraform import quant_kv_store.audit "my-project/store-id"
```

The import reads `format` from the API.
