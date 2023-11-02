terraform {
  required_providers {
    quant = {
      source = "registry.terraform.io/quantcdn/quant"
    }
  }
}

provider "quant" {}

data "quant_projects" "all" {}

output "all_projects" {
  value = data.quant_projects.all
}
