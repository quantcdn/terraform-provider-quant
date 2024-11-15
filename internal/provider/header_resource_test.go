package provider_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-quant/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"quant": providerserver.NewProtocol6WithError(provider.New()()),
}

func TestHeaderResource(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	if bearer == "" {
		t.Skip("QUANT_BEARER not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testHeaderResourceConfig("test-project", map[string]string{
					"X-Custom-Header": "test-value",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_header.test", "project", "test-project"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Custom-Header", "test-value"),
				),
			},
			// Update and Read testing
			{
				Config: testHeaderResourceConfig("test-project", map[string]string{
					"X-Custom-Header": "updated-value",
					"Another-Header":  "new-value",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_header.test", "project", "test-project"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Custom-Header", "updated-value"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.Another-Header", "new-value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testHeaderResourceConfig(project string, headers map[string]string) string {
	headersStr := "{\n"
	for k, v := range headers {
		headersStr += fmt.Sprintf("    \"%s\" = \"%s\"\n", k, v)
	}
	headersStr += "  }"

	return fmt.Sprintf(`
resource "quant_header" "test" {
  project = "%s"
  headers = %s
}
`, project, headersStr)
}
