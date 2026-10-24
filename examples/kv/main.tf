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
