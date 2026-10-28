---
page_title: "Quant: quant_kv_ledger_public_key"
subcategory: "Key-Value Store"
description: |-
  Reads the public key that verifies the KV ledger digest signatures of a store.
---

# KV Ledger Public Key Data Source

Reads the public key that verifies the KV ledger digest signatures of a store, as PEM, with the current and historical keys.

## Example Usage

```hcl
data "quant_kv_ledger_public_key" "audit" {
  project  = "my-project"
  store_id = quant_kv_store.audit.store_id
}

output "ledger_public_key_pem" {
  value = data.quant_kv_ledger_public_key.audit.public_key_pem
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `store_id` - (Required) The KV store ID.
- `organization` - (Optional) The organization. Defaults to the provider organization.

## Attribute Reference

- `key_id` - The identifier of the current signing key.
- `algorithm` - The signature algorithm, for example `ECDSA_SHA_256`.
- `curve` - The curve, for example `P-256`.
- `public_key_pem` - The current public key (SubjectPublicKeyInfo, PEM).
- `keys` - The current and historical verification keys: `key_id`, `public_key_pem`, `current`. Select a key by the `keyId` of a signature.
