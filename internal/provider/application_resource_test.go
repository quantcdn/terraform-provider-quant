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

func mockApplicationServer(t *testing.T, org string, appName string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	appDeleted := false

	// Dynamic closure state for mock responses
	var currentMinCapacity = 1
	var currentMaxCapacity = 2

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"appName":           appName,
			"organisation":      org,
			"status":            "active",
			"runningCount":      1,
			"desiredCount":      1,
			"minCapacity":       currentMinCapacity,
			"maxCapacity":       currentMaxCapacity,
			"composeDefinition": map[string]interface{}{},
			"containerNames":    []string{"web"},
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create application
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications", baseUrl, org),
		func(req *http.Request) (*http.Response, error) {
			appDeleted = false
			var requestBody map[string]interface{}
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if minCap, ok := requestBody["minCapacity"].(float64); ok {
						currentMinCapacity = int(minCap)
					}
					if maxCap, ok := requestBody["maxCapacity"].(float64); ok {
						currentMaxCapacity = int(maxCap)
					}
				}
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// GET application (used by polling + read)
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications/%s", baseUrl, org, appName),
		func(req *http.Request) (*http.Response, error) {
			if appDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// GET list applications
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications", baseUrl, org),
		func(req *http.Request) (*http.Response, error) {
			if appDeleted {
				return httpmock.NewJsonResponse(200, []interface{}{})
			}
			return httpmock.NewJsonResponse(200, []interface{}{createResponse()})
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
				Config: testAccApplicationResourceConfig(org, appName, 1, 2),
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
			// Update testing (ForceNew — triggers destroy+create because all fields require replacement)
			{
				Config: testAccApplicationResourceConfig(org, appName, 2, 4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_application.test", "app_name", "test-app"),
					resource.TestCheckResourceAttr("quant_application.test", "min_capacity", "2"),
					resource.TestCheckResourceAttr("quant_application.test", "max_capacity", "4"),
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

func testAccApplicationResourceConfig(org string, appName string, minCapacity int, maxCapacity int) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_application" "test" {
  app_name           = %[2]q
  compose_definition = "{}"
  min_capacity       = %[3]d
  max_capacity       = %[4]d
}
`, org, appName, minCapacity, maxCapacity)
}

// ---------- HTTP error-path integration tests ----------

func setupApplicationErrorResponder(t *testing.T, org string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/applications", baseUrl, org),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Application create request received — returning %d", statusCode)
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testApplicationErrorConfig() string {
	return `
provider "quant" {
  organization = "test-org"
  bearer       = "bad-token"
}

resource "quant_application" "test" {
  app_name           = "error-test-app"
  compose_definition = "{}"
}
`
}

func TestAccApplicationResource_CreateError401(t *testing.T) {
	setupApplicationErrorResponder(t, "test-org", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testApplicationErrorConfig(),
				ExpectError: regexp.MustCompile(`Authentication Failed`),
			},
		},
	})
}

func TestAccApplicationResource_CreateForbidden403(t *testing.T) {
	setupApplicationErrorResponder(t, "test-org", 403, map[string]interface{}{
		"error":   true,
		"message": "Insufficient permissions",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testApplicationErrorConfig(),
				ExpectError: regexp.MustCompile(`Authorization Failed`),
			},
		},
	})
}

func TestAccApplicationResource_CreateConflict409(t *testing.T) {
	setupApplicationErrorResponder(t, "test-org", 409, map[string]interface{}{
		"error":   true,
		"message": "Application already exists",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testApplicationErrorConfig(),
				ExpectError: regexp.MustCompile(`Application Already Exists`),
			},
		},
	})
}

func TestAccApplicationResource_CreateBadRequest400(t *testing.T) {
	setupApplicationErrorResponder(t, "test-org", 400, map[string]interface{}{
		"error":   true,
		"message": "Invalid application configuration",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testApplicationErrorConfig(),
				ExpectError: regexp.MustCompile(`Invalid Application Configuration`),
			},
		},
	})
}

func TestAccApplicationResource_CreateServerError500(t *testing.T) {
	setupApplicationErrorResponder(t, "test-org", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testApplicationErrorConfig(),
				ExpectError: regexp.MustCompile(`Unable to Create Application`),
			},
		},
	})
}
