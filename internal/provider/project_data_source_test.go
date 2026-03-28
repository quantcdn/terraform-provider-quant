package provider_test

import (
	"fmt"
	"net/http"
	"github.com/quantcdn/terraform-provider-quant/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var projectDataSourceResponse = map[string]interface{}{
	"id":              123,
	"name":            "test-project",
	"uuid":            "test-uuid-123",
	"machine_name":    "default",
	"created_at":      "2024-01-01T00:00:00Z",
	"updated_at":      "2024-01-01T00:00:00Z",
	"region":          "au",
	"organization_id": 456,
	"security_score":  "A",
	"git_url":         "https://github.com/test/repo.git",
	"write_token":     "test-write-token-123",
}

var projectDataSourceResponseWithoutToken = map[string]interface{}{
	"id":              123,
	"name":            "test-project",
	"uuid":            "test-uuid-123",
	"machine_name":    "default",
	"created_at":      "2024-01-01T00:00:00Z",
	"updated_at":      "2024-01-01T00:00:00Z",
	"region":          "au",
	"organization_id": 456,
	"security_score":  "A",
	"git_url":         "https://github.com/test/repo.git",
	"write_token":     "",
}

func mockProjectDataSourceServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// Project read with token
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			withToken := req.URL.Query().Get("with_token")
			t.Logf("Project data source request received with with_token=%s", withToken)

			if withToken == "true" {
				return httpmock.NewJsonResponse(200, projectDataSourceResponse)
			}
			return httpmock.NewJsonResponse(200, projectDataSourceResponseWithoutToken)
		})
}

func testProjectDataSourceFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"quant": providerserver.NewProtocol6WithError(provider.New()()),
	}
}

func TestAccProjectDataSource(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	mockProjectDataSourceServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProjectDataSourceFactories(t),
		Steps: []resource.TestStep{
			// Read testing with token
			{
				Config: testAccProjectDataSourceConfig(organizationID, projectID, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.quant_project.test", "id", "123"),
					resource.TestCheckResourceAttr("data.quant_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("data.quant_project.test", "uuid", "test-uuid-123"),
					resource.TestCheckResourceAttr("data.quant_project.test", "machine_name", "default"),
					resource.TestCheckResourceAttr("data.quant_project.test", "write_token", "test-write-token-123"),
					resource.TestCheckResourceAttr("data.quant_project.test", "with_token", "true"),
				),
			},
		},
	})
}

func TestAccProjectDataSourceWithoutToken(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	mockProjectDataSourceServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProjectDataSourceFactories(t),
		Steps: []resource.TestStep{
			// Read testing without token
			{
				Config: testAccProjectDataSourceConfig(organizationID, projectID, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.quant_project.test", "id", "123"),
					resource.TestCheckResourceAttr("data.quant_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("data.quant_project.test", "uuid", "test-uuid-123"),
					resource.TestCheckResourceAttr("data.quant_project.test", "machine_name", "default"),
					resource.TestCheckResourceAttr("data.quant_project.test", "with_token", "false"),
					resource.TestCheckResourceAttr("data.quant_project.test", "write_token", ""),
				),
			},
		},
	})
}

func testAccProjectDataSourceConfig(organization string, projectName string, withToken bool) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

data "quant_project" "test" {
  machine_name = %[2]q
  with_token   = %[3]t
}
`, organization, projectName, withToken)
}
