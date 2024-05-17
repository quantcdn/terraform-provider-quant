terraform {
  required_providers {
    quant = {
      source = "registry.terraform.io/quantcdn/quant"
    }
  }
}

provider "quant" {
  bearer = "<token>"
  organization = "quant"
}

data "quant_projects" "list" {}

output "quant_projects" {
  value = data.quant_projects.list
}
