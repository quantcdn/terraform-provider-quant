package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccHeaderResourceConfig = `
provider "quant" {
  bearer = "test-token"
  organization = "test-org"
}

resource "quant_header" "test" {
  project = "test-project"
  headers = {
    "X-Test-Header" = "test-value"
    "X-Another-Header" = "another-value"
  }
}
`

const testAccHeaderResourceConfigUpdated = `
provider "quant" {
  bearer = "test-token"
  organization = "test-org" 
}

resource "quant_header" "test" {
  project = "test-project"
  headers = {
    "X-Test-Header" = "updated-value"
    "X-New-Header" = "new-value"
  }
}
`

func TestAccHeaderResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccHeaderResourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_header.test", "project", "test-project"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Test-Header", "test-value"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Another-Header", "another-value"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "quant_header.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccHeaderResourceConfigUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_header.test", "project", "test-project"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Test-Header", "updated-value"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-New-Header", "new-value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
