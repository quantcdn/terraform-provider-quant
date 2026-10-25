# KV Ledger Resource

Manages a tamper-evident ledger on a collection of a format 2 KV store (`quant_kv_store` with `format = 2`). The archive of a ledger is kept with S3 Object Lock, and the ledger cannot be dropped before its retention ends.

## Example Usage

```hcl
resource "quant_kv_store" "audit" {
  project = "my-project"
  name    = "audit"
  format  = 2
}

resource "quant_kv_ledger" "ai_use" {
  project         = "my-project"
  store_id        = quant_kv_store.audit.store_id
  name            = "aiUse"
  collection      = "ai-use:*"
  chains          = 4
  retention_days  = 2555
  lock_mode       = "governance"
  personal_fields = ["requesterName"]
  subject_field   = "requesterId"
  subject_mode    = "pseudonym"
  subject_case    = "fold"
  anchor          = { type = "rfc3161", tsa_url = "http://timestamp.digicert.com" }

  lifecycle {
    prevent_destroy = true
  }
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `store_id` - (Required) The KV store ID. The store must be format 2.
- `name` - (Required) The ledger name, unique in the store.
- `collection` - (Required) A collection, or a pattern where `*` as a whole segment matches exactly one segment. It must not overlap another ledger, and it must be empty when the ledger is declared.
- `chains` - (Required) Parallel hash chains (shards) per collection. Can only go up.
- `retention_days` - (Required) S3 Object Lock retention of the archive, in days. Can only go up.
- `lock_mode` - (Required) `governance` or `compliance`. `governance` can become `compliance`, never back.
- `personal_fields` - (Optional) Top-level value fields that are encrypted per data subject, so a shred can erase them. Needs `subject_field`.
- `subject_field` - (Optional) The top-level value field that names the data subject.
- `subject_mode` - (Optional) `pseudonym` or `clear`. Needs `subject_field`; `pseudonym` when left out. `pseudonym`: each entry stores a random subject token in the subject field and keeps the value as appended sealed, so after a shred no entry names the person. `clear`: each entry stores the value as appended, so after a shred the id and its activity stay readable. Cannot change.
- `subject_case` - (Optional) `exact` or `fold`. Needs `subject_field`; `exact` when left out. `exact`: `Alice@Example.com` and `alice@example.com` are two subjects, so a shred of one leaves the other. `fold`: ids that differ only in case are one subject. Use `fold` for emails and usernames. Cannot change.
- `anchor` - (Optional) External witness of the daily digests. `type` is `none` (default), `customer-bucket` (needs `bucket_arn` and `account_id`, optional `prefix`) or `rfc3161` (needs `tsa_url`).
- `writer_policy` - (Optional) `platform_principals`: the platform service principals that may append (1 to 16).
- `organization` - (Optional) The organization. Defaults to the provider organization.

## Attribute Reference

- `state` - `PENDING` for about 75 seconds after the declare, then `ACTIVE` or `FAILED`. `DROPPED` after a drop.
- `definition_hash` - Hex SHA-256 of the canonical definition.
- `retain_until` - The archive retention end. The ledger can be dropped after it.
- `created_at`, `active_after`, `updated_at`, `dropped_at` - Timestamps.
- `error` - `FAILED` only: why the ledger failed.
- `ledger_name` - The same value as `name`.
- `subject_mode`, `subject_case` - The values the API stored (`pseudonym` and `exact` when left out). Both are null when `subject_field` is not set.

## Plan-time checks

Terraform never replaces a ledger: a retained ledger cannot be dropped, and a new ledger needs an empty collection. So the plan fails, before anything reaches the API, when:

- `name`, `collection`, `personal_fields`, `subject_field`, `subject_mode`, `subject_case`, `anchor`, `writer_policy`, `store_id`, `project` or `organization` changes (`KV ledger <attribute> cannot change`). Declare a new ledger with a new name on an empty collection instead.
- `subject_mode` or `subject_case` without `subject_field`.
- `chains` or `retention_days` goes down (`<attribute> can only go up`).
- `lock_mode` goes from `compliance` to `governance`.
- The ledger in state is `DROPPED` and is still in the configuration. A dropped name cannot be used again.

An increase of `chains` or `retention_days`, or `governance` to `compliance`, is an in-place update that sends only the changed fields.

A ledger in state that is `FAILED` is declared again: the plan drops it (allowed for a `FAILED` ledger) and creates it. Before the drop, the apply reads the ledger again. If the server no longer shows it `FAILED` (for example it is `ACTIVE`), the apply refuses the drop. Run `terraform plan` again.

## Create

Create declares the ledger (the API answers `202` with a `PENDING` ledger) and waits until it leaves `PENDING`. The default wait is 5 minutes. Set the environment variable `QUANTCDN_KV_LEDGER_TIMEOUT` (a Go duration) to wait longer. A `FAILED` ledger is an error, and the next apply declares it again.

If the ledger is still `PENDING` when the wait ends, the apply fails and does not save the ledger in state. The ledger exists on the server. Import it (see [Import](#import)) before the next apply; otherwise the next apply declares it again and the API refuses. The provider does not save a `PENDING` ledger as tainted, because the next apply would then drop it.

## Rate limit

The KV API allows 10 index and ledger changes (declare, raise, drop) per minute per project. Above that, it answers `429` with a `Retry-After` header. The provider then waits the full `Retry-After` and sends the declare, the raise or the drop again, until the ledger wait timeout (`QUANTCDN_KV_LEDGER_TIMEOUT`, default 5 minutes) ends.

## Destroy

Destroy drops the ledger. While the archive retention runs, the API refuses with `LEDGER_RETAINED`, and the destroy fails with the `retainUntil` date. To stop managing the ledger without a drop, use a `removed` block:

```hcl
removed {
  from = quant_kv_ledger.ai_use

  lifecycle {
    destroy = false
  }
}
```

or run `terraform state rm quant_kv_ledger.ai_use`. In Pulumi, set the `retainOnDelete: true` resource option, or run `pulumi state delete`.

If the refusal reason is `unarchived` or `retention_unknown`, some entries are not archived yet. Try again in a few minutes.

## Import

```bash
terraform import quant_kv_ledger.ai_use "my-project/store-id/aiUse"
```

The import ID is `project/store_id/name`, or `organization/project/store_id/name`.
