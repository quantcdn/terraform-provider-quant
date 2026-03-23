package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var environmentResponse = map[string]interface{}{
	"envName":      "staging",
	"status":       "ACTIVE",
	"runningCount": 1,
	"desiredCount": 1,
	"minCapacity":  1,
	"maxCapacity":  2,
}

func mockEnvironmentServer(t *testing.T, org string, app string, envName string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	envDeleted := false

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create environment
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments", baseUrl, org, app),
		func(req *http.Request) (*http.Response, error) {
			envDeleted = false
			return httpmock.NewJsonResponse(200, environmentResponse)
		})

	// GET environment (used by polling + read)
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s", baseUrl, org, app, envName),
		func(req *http.Request) (*http.Response, error) {
			if envDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, environmentResponse)
		})

	// PUT update environment
	httpmock.RegisterResponder("PUT",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s", baseUrl, org, app, envName),
		func(req *http.Request) (*http.Response, error) {
			if envDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			var requestBody map[string]interface{}
			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			if v, ok := requestBody["minCapacity"]; ok {
				environmentResponse["minCapacity"] = v
			}
			if v, ok := requestBody["maxCapacity"]; ok {
				environmentResponse["maxCapacity"] = v
			}

			return httpmock.NewJsonResponse(200, environmentResponse)
		})

	// DELETE environment
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s", baseUrl, org, app, envName),
		func(req *http.Request) (*http.Response, error) {
			if envDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			envDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccEnvironmentResource(t *testing.T) {
	org := "test-org"
	app := "test-app"
	envName := "staging"
	mockEnvironmentServer(t, org, app, envName)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfig(org, app, envName, 1, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_environment.test", "env_name", "staging"),
					resource.TestCheckResourceAttr("quant_environment.test", "application", "test-app"),
					resource.TestCheckResourceAttr("quant_environment.test", "status", "ACTIVE"),
					resource.TestCheckResourceAttr("quant_environment.test", "running_count", "1"),
					resource.TestCheckResourceAttr("quant_environment.test", "desired_count", "1"),
					resource.TestCheckResourceAttr("quant_environment.test", "min_capacity", "1"),
					resource.TestCheckResourceAttr("quant_environment.test", "max_capacity", "2"),
				),
			},
			// Import testing
			{
				ResourceName:  "quant_environment.test",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s/%s", app, envName),
				ImportStateVerifyIgnore: []string{
					"clone_configuration_from",
					"image_suffix",
					"spot_configuration",
					"environment_variables",
					"merge_environment",
					"compose_definition",
				},
			},
			// Update and Read testing
			{
				Config: testAccEnvironmentResourceConfig(org, app, envName, 2, 4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_environment.test", "env_name", "staging"),
					resource.TestCheckResourceAttr("quant_environment.test", "min_capacity", "2"),
					resource.TestCheckResourceAttr("quant_environment.test", "max_capacity", "4"),
				),
			},
		},
	})
}

func testAccEnvironmentResourceConfig(org string, app string, envName string, minCap int, maxCap int) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_environment" "test" {
  organization = %[1]q
  application  = %[2]q
  env_name     = %[3]q
  min_capacity = %[4]d
  max_capacity = %[5]d
}
`, org, app, envName, minCap, maxCap)
}

// ---------- HTTP error-path integration tests ----------

func setupEnvironmentErrorResponder(t *testing.T, org string, app string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments", baseUrl, org, app),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testEnvironmentErrorConfig() string {
	return `
provider "quant" {
  organization = "test-org"
  bearer = "testtoken"
}

resource "quant_environment" "test" {
  organization = "test-org"
  application  = "test-app"
  env_name     = "error-test"
  min_capacity = 1
  max_capacity = 2
}
`
}

func TestAccEnvironmentResource_CreateError401(t *testing.T) {
	setupEnvironmentErrorResponder(t, "test-org", "test-app", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testEnvironmentErrorConfig(),
				ExpectError: regexp.MustCompile(`Authentication Failed`),
			},
		},
	})
}

func TestAccEnvironmentResource_CreateError500(t *testing.T) {
	setupEnvironmentErrorResponder(t, "test-org", "test-app", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testEnvironmentErrorConfig(),
				ExpectError: regexp.MustCompile(`Unable to Create Environment`),
			},
		},
	})
}
