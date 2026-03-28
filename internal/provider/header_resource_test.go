package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"io"
	"net/http"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/provider"
)

var customHeaderResponse = map[string]string{}

func testAccHeaderPreCheck(t *testing.T, org string, project string) {
	// Enable httpmock
	httpmock.Activate()

	// Base URL for the API
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Mock the headers list endpoint
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/%s/custom-headers", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, customHeaderResponse)
		})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/custom-headers", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			// Read the request body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			// Parse the JSON body
			var requestBody struct {
				Headers map[string]string `json:"headers"`
			}

			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			// Update the current headers with exactly what was sent
			customHeaderResponse = requestBody.Headers
			return httpmock.NewJsonResponse(200, customHeaderResponse)
		})

	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/projects/%s/custom-headers", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, customHeaderResponse)
		})
}

// testHeaderResourceFactories are used to instantiate a provider during
// acceptance testing.
var testHeaderResourceFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"quant": providerserver.NewProtocol6WithError(provider.New()()),
}

func TestAccHeaderResource(t *testing.T) {
	// Set up the mock server.
	project := "testproject"
	org := "testorg"

	testAccHeaderPreCheck(t, org, project)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testHeaderResourceFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccHeaderResourceConfig(org, project),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_header.test", "project", project),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Test", "test"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Another-Header", "test"),
					testAccHeaderResourceExists("quant_header.test"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "quant_header.test",
				ImportState:       true,
				ImportStateId:     project,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: testAccHeaderResourceConfigUpdate(org, project),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_header.test", "project", project),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-Test", "updated"),
					resource.TestCheckResourceAttr("quant_header.test", "headers.X-New-Header", "new"),
				),
			},
		},
	})
}

func testAccHeaderResourceConfig(org string, project string) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

resource "quant_header" "test" {
  project = %[2]q
  headers = {
    "X-Test"           = "test"
    "X-Another-Header" = "test"
  }
}
`, org, project)
}

func testAccHeaderResourceConfigUpdate(org string, project string) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

resource "quant_header" "test" {
  project = %[2]q
  headers = {
    "X-Test"      = "updated"
    "X-New-Header" = "new"
  }
}
`, org, project)
}

func testAccHeaderResourceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Header ID is set")
		}

		return nil
	}
}

// ---------- HTTP error-path integration tests ----------

func TestAccHeaderResource_CreateError500(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseUrl := "https://dashboard.quantcdn.io/api/v2"
	org := "test-org"
	project := "test-project"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/custom-headers", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(500, map[string]interface{}{
				"error":   true,
				"message": "Internal server error",
			})
		})

	// Register DELETE so the test framework's cleanup phase does not fail.
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/projects/%s/custom-headers", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, map[string]string{})
		})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "quant" {
  organization = "test-org"
  bearer = "testtoken"
}

resource "quant_header" "test" {
  project = "test-project"
  headers = {
    "X-Test" = "value"
  }
}
`,
				ExpectError: regexp.MustCompile(`Failed to add custom headers`),
			},
		},
	})
}
