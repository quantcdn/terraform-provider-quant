terraform {
  required_providers {
    quant = {
      source = "quantcdn/quant"
    }
  }
}

provider "quant" {
  # Set QUANTCDN_API_TOKEN and QUANTCDN_ORGANIZATION in the environment.
}

variable "project" {
  type    = string
  default = "my-project"
}

# A format 2 store supports indexes and ledgers. The format is set at create;
# Terraform never changes it (use the upgrade API, then set format = 2).
resource "quant_kv_store" "orders" {
  project = var.project
  name    = "orders"
  format  = 2
}

# An index definition cannot change: any change replaces the index.
resource "quant_kv_index" "orders_by_customer" {
  project    = var.project
  store_id   = quant_kv_store.orders.store_id
  name       = "byCustomer"
  collection = "order"
  fields     = ["customerId"]
  sort_by    = "createdAt"
}

# A ledger definition never changes, and the ledger cannot be dropped while
# its archive retention runs. chains, retention_days and lock_mode can only go up.
resource "quant_kv_ledger" "ai_use" {
  project         = var.project
  store_id        = quant_kv_store.orders.store_id
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

# To stop managing the ledger without dropping it:
# removed {
#   from = quant_kv_ledger.ai_use
#   lifecycle {
#     destroy = false
#   }
# }

data "quant_kv_ledger" "ai_use" {
  project     = var.project
  store_id    = quant_kv_store.orders.store_id
  ledger_name = quant_kv_ledger.ai_use.name
}

data "quant_kv_ledger_public_key" "orders" {
  project  = var.project
  store_id = quant_kv_store.orders.store_id
}

output "ai_use_retained_until" {
  value = data.quant_kv_ledger.ai_use.retain_until
}

output "ledger_public_key_pem" {
  value = data.quant_kv_ledger_public_key.orders.public_key_pem
}
