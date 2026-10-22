---
page_title: "Quant: quant_kv_store"
subcategory: "Key-Value Store"
description: |-
  Manages a Quant key-value store in a project. Format 2 stores support indexes, ledgers, conditional writes and transactions.
---

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

## Destroy

Destroy deletes the store. While the store has a ledger that is `PENDING` or `ACTIVE`, or `DROPPED` but not yet finished by the archive job, the API refuses with `LEDGER_PRESENT` and the destroy fails. The provider shows the HTTP status only (for example `409 Conflict`). The error code (`DIGEST_OWED` or `LEDGER_PRESENT`) is in the body of the API answer. If the code is `DIGEST_OWED`, the last digest of the store is not written yet: run the destroy again in a few minutes.

Destroy never deletes the ledger archive of the store. The archive objects, digests and anchors stay after the store is deleted, also after their retention ends. While they exist, a new store with the same name cannot be created (`409 STORE_NAME_RETAINED`). To delete a retained archive, use the archive purge in the dashboard or the KV API (`/api/v2/organizations/{organization}/kv/archives`, see [Deleting a retained archive](https://docs.quantcdn.io/compute/kv-ledgers/#deleting-a-retained-archive)). The provider has no resource or data source for it: a purge is a one-time action that cannot be undone, not state that Terraform manages.

## Import

```bash
terraform import quant_kv_store.audit "my-project/store-id"
```

The import reads `format` from the API.
