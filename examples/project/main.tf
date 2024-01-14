terraform {
  required_providers {
    quant = {
      source = "registry.terraform.io/quantcdn/quant"
    }
  }
  required_version = ">= 1.1.0"
}

resource "quant_project" "tf" {
  name = "Terraform project"
  allow_query_params = false
  basic_auth_username = "quant"
  basic_auth_password = "qcdn"
  basic_auth_preview_only = false
}
