terraform {
  required_providers {
    quant = {
      source = "registry.terraform.io/quantcdn/quant"
    }
  }
  required_version = ">= 1.1.0"
}

provider "quant" {
  # secret = "quantsecrettoken"
  # organization = "quant"
}

resource "quant_project" "tf" {
  name = "Terraform project 7"
  allow_query_params = false
  basic_auth_username = "quant"
  basic_auth_password = "qcdn"
  basic_auth_preview_only = false
}

resource "quant_domain" "d1" {
  project = tf.machine_name
  domain = "test.com"
}

output "tf_project" {
  value = quant_project.tf
}
