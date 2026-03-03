package provider_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var applicationResponse = map[string]interface{}{
	"appName":           "test-app",
	"organisation":      "test-org",
	"status":            "active",
	"runningCount":      1,
	"desiredCount":      1,
	"minCapacity":       1,
	"maxCapacity":       2,
	"composeDefinition": map[string]interface{}{},
	"containerNames":    []string{"web"},
}

func mockApplicationServer(t *testing.T, org string, appName string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	appDeleted := false

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create application
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications", baseUrl, org),
		func(req *http.Request) (*http.Response, error) {
			appDeleted = false
			return httpmock.NewJsonResponse(200, applicationResponse)
		})

	// GET application (used by polling + read)
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications/%s", baseUrl, org, appName),
		func(req *http.Request) (*http.Response, error) {
			if appDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, applicationResponse)
		})

	// GET list applications
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications", baseUrl, org),
		func(req *http.Request) (*http.Response, error) {
			if appDeleted {
				return httpmock.NewJsonResponse(200, []interface{}{})
			}
			return httpmock.NewJsonResponse(200, []interface{}{applicationResponse})
		})

	// DELETE application
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/applications/%s", baseUrl, org, appName),
		func(req *http.Request) (*http.Response, error) {
			if appDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			appDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccApplicationResource(t *testing.T) {
	org := "test-org"
	appName := "test-app"
	mockApplicationServer(t, org, appName)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfig(org, appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_application.test", "app_name", "test-app"),
					resource.TestCheckResourceAttr("quant_application.test", "organization", "test-org"),
					resource.TestCheckResourceAttr("quant_application.test", "status", "active"),
					resource.TestCheckResourceAttr("quant_application.test", "running_count", "1"),
					resource.TestCheckResourceAttr("quant_application.test", "desired_count", "1"),
					resource.TestCheckResourceAttr("quant_application.test", "min_capacity", "1"),
					resource.TestCheckResourceAttr("quant_application.test", "max_capacity", "2"),
				),
			},
			// Import testing
			{
				ResourceName:  "quant_application.test",
				ImportState:   true,
				ImportStateId: appName,
				ImportStateVerifyIgnore: []string{
					"compose_definition",
					"container_names",
				},
			},
		},
	})
}

func testAccApplicationResourceConfig(org string, appName string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_application" "test" {
  app_name           = %[2]q
  compose_definition = "{}"
  min_capacity       = 1
  max_capacity       = 2
}
`, org, appName)
}
