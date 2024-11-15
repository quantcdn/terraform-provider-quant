package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProjectResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	if bearer == "" {
		t.Skip("QUANT_BEARER not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testProjectResourceConfig("test-project", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("quant_project.test", "allow_query_params", "false"),
					resource.TestCheckResourceAttr("quant_project.test", "region", "au"),
				),
			},
			// Update and Read testing
			{
				Config: testProjectResourceConfig("test-project-updated", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_project.test", "name", "test-project-updated"),
					resource.TestCheckResourceAttr("quant_project.test", "allow_query_params", "true"),
					resource.TestCheckResourceAttr("quant_project.test", "region", "au"),
				),
			},
			// Import testing
			{
				ResourceName:      "quant_project.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore auth fields as they're not returned by the API
				ImportStateVerifyIgnore: []string{
					"basic_auth_username",
					"basic_auth_password",
					"basic_auth_preview_only",
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testProjectResourceConfig(name string, allowQueryParams bool) string {
	return `
resource "quant_project" "test" {
  name = "` + name + `"
  allow_query_params = ` + boolToString(allowQueryParams) + `
  region = "au"
}
`
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
