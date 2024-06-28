terraform {
  required_providers {
    quant = {
      source = "registry.terraform.io/quantcdn/quant"
    }
  }
}

provider "quant" {
  bearer = "fxj1eivEXhKdIEVGuKkfrcfv4WeEQ8uqqNqEeIEy4zEb6hlz8Tj1SdRxdc9x"
  organization = "quant"
}

resource "quant_project" "test-terraform" {
  name = "tf test 7"
}
