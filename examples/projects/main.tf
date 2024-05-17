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

resource "quant_project" "test-terraform" {
  name = "tf test 7"
}
