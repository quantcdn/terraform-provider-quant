package provider_test

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"io"
	"net/http"
	"regexp"
	"github.com/quantcdn/terraform-provider-quant/internal/provider"
	"testing"
)

var projectResponse = map[string]interface{}{
	"id":                 123,
	"name":               "test-project",
	"allow_query_params": false,
	"region":             "au",
	"uuid":               "123",
	"machine_name":       "test-project",
	"disable_revisions":  true,
	"fastly_migrated":    1,
	"project_type":       "normal",
	"organization":       "test-organization",
	"parent_project_id":  0,
	"write_token":        "test-write-token-123",
}

func mockProjectServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Track deletion state
	projectDeleted := false

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects", baseUrl, organizationID), func(req *http.Request) (*http.Response, error) {
		if projectDeleted {
			return httpmock.NewStringResponse(404, "Not Found"), nil
		}
		return httpmock.NewJsonResponse(200, projectResponse)
	})

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/%s", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
			if projectDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, projectResponse)
		})

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/0", baseUrl, organizationID), func(req *http.Request) (*http.Response, error) {
			if projectDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, projectResponse)
		})

	// Handle import by UUID (id field = "123")
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/123", baseUrl, organizationID), func(req *http.Request) (*http.Response, error) {
			if projectDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, projectResponse)
		})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects", baseUrl, organizationID),
		func(req *http.Request) (*http.Response, error) {
			projectDeleted = false // Reset deletion state on create
			return httpmock.NewJsonResponse(200, projectResponse)
		})

	httpmock.RegisterResponder("PATCH",
		fmt.Sprintf("%s/organizations/%s/projects/%s", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
			if projectDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			var requestBody struct {
				Name             string `json:"name"`
				AllowQueryParams bool   `json:"allow_query_params"`
				Region           string `json:"region"`
			}

			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			projectResponse["name"] = requestBody.Name
			projectResponse["allow_query_params"] = requestBody.AllowQueryParams
			projectResponse["region"] = requestBody.Region
			return httpmock.NewJsonResponse(200, projectResponse)
		})

	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/projects/%s", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
			if projectDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			projectDeleted = true // Mark project as deleted
			return httpmock.NewJsonResponse(200, projectResponse)
		})
}

func testProjectResourceFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"quant": providerserver.NewProtocol6WithError(provider.New()()),
	}
}

func TestProjectResource(t *testing.T) {
	organizationID := "test-organization"
	projectID := "test-project"
	mockProjectServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProjectResourceFactories(t),
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testProjectResourceConfig(organizationID, projectID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("quant_project.test", "allow_query_params", "false"),
					resource.TestCheckResourceAttr("quant_project.test", "region", "au"),
					resource.TestCheckResourceAttr("quant_project.test", "uuid", "123"),
					resource.TestCheckResourceAttr("quant_project.test", "machine_name", "test-project"),
				),
			},
			// Import testing
			{
				ResourceName: "quant_project.test",
				ImportState:  true,
				ImportStateVerifyIgnore: []string{
					"basic_auth_username",
					"basic_auth_password",
					"basic_auth_preview_only",
				},
			},
			// Update and Read testing
			{
				Config: testProjectResourceConfig(organizationID, fmt.Sprintf("%s-updated", projectID), true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_project.test", "name", "test-project-updated"),
					resource.TestCheckResourceAttr("quant_project.test", "allow_query_params", "true"),
					resource.TestCheckResourceAttr("quant_project.test", "region", "au"),
					resource.TestCheckResourceAttr("quant_project.test", "uuid", "123"),
					resource.TestCheckResourceAttr("quant_project.test", "machine_name", "test-project"),
				),
			},

			// Delete testing automatically occurs in TestCase
		},
	})
}

func testProjectResourceConfig(organization string, name string, allowQueryParams bool) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

resource "quant_project" "test" {
  name = %[2]q
  allow_query_params = %[3]t
  region = "au"
}
`, organization, name, allowQueryParams)
}

func TestAccProjectResourceCreateDuplicateNameError(t *testing.T) {
	setupProjectErrorServer(t, "test-organization")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectResourceConfigDuplicateName(),
				ExpectError: regexp.MustCompile(`Project name is not unique in this organisation\. Try another\.`),
			},
		},
	})
}

func setupProjectErrorServer(t *testing.T, organizationID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Log all requests for debugging
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// Mock project creation endpoint to return 400 error for duplicate name
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects", baseUrl, organizationID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Create project request received - returning duplicate name error")
			errorResponse := map[string]interface{}{
				"error":   true,
				"message": "Project name is not unique in this organisation. Try another.",
			}
			return httpmock.NewJsonResponse(400, errorResponse)
		})
}

func testAccProjectResourceConfigDuplicateName() string {
	return `
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_project" "test" {
	name = "Duplicate Project Name"
}
`
}

// ---------- HTTP error-path integration tests ----------

func setupProjectErrorResponder(t *testing.T, orgID string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects", baseUrl, orgID),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testProjectErrorConfig() string {
	return `
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_project" "test" {
	name = "error-test"
}
`
}

func TestProjectResource_CreateAuth401(t *testing.T) {
	setupProjectErrorResponder(t, "test-organization", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testProjectErrorConfig(),
				ExpectError: regexp.MustCompile(`Authentication Failed`),
			},
		},
	})
}

func TestProjectResource_CreateForbidden403(t *testing.T) {
	setupProjectErrorResponder(t, "test-organization", 403, map[string]interface{}{
		"error":   true,
		"message": "Insufficient permissions",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testProjectErrorConfig(),
				ExpectError: regexp.MustCompile(`Authorization Failed`),
			},
		},
	})
}

func TestProjectResource_CreateConflict409(t *testing.T) {
	setupProjectErrorResponder(t, "test-organization", 409, map[string]interface{}{
		"error":   true,
		"message": "Project already exists",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testProjectErrorConfig(),
				ExpectError: regexp.MustCompile(`Project Already Exists`),
			},
		},
	})
}
