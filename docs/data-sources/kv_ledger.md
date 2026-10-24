# KV Ledger Data Source

Reads a KV ledger: its definition, its state, its archive retention end and the last verification.

## Example Usage

```hcl
data "quant_kv_ledger" "ai_use" {
  project     = "my-project"
  store_id    = quant_kv_store.audit.store_id
  ledger_name = "aiUse"
}

output "ai_use_retained_until" {
  value = data.quant_kv_ledger.ai_use.retain_until
}

output "ai_use_last_verify_ok" {
  value = data.quant_kv_ledger.ai_use.last_verify.ok
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `store_id` - (Required) The KV store ID.
- `ledger_name` - (Required) The ledger name.
- `organization` - (Optional) The organization. Defaults to the provider organization.

## Attribute Reference

- `name`, `collection`, `chains`, `retention_days`, `lock_mode`, `personal_fields`, `subject_field`, `anchor`, `writer_policy` - The ledger definition.
- `state` - `PENDING`, `ACTIVE`, `FAILED` or `DROPPED`.
- `definition_hash` - Hex SHA-256 of the canonical definition.
- `retain_until` - The archive retention end.
- `created_at`, `active_after`, `updated_at`, `dropped_at`, `last_digest_at` - Timestamps.
- `error` - `FAILED` only: why the ledger failed.
- `last_verify` - The last verification: `id`, `at`, `started_at`, `duration_ms`, `ok`, `entries`, `shards_failed`, `complete`, `pages`, `trigger`, `scope` and `first_failure`.
