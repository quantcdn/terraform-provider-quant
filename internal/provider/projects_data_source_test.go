package provider_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var projectsListResponse = []map[string]interface{}{
	{
		"id":              1,
		"name":            "Project Alpha",
		"machine_name":    "project-alpha",
		"uuid":            "uuid-alpha",
		"created_at":      "2024-01-01T00:00:00Z",
		"updated_at":      "2024-01-01T00:00:00Z",
		"region":          "au",
		"organization_id": 100,
	},
	{
		"id":              2,
		"name":            "Project Beta",
		"machine_name":    "project-beta",
		"uuid":            "uuid-beta",
		"created_at":      "2024-02-01T00:00:00Z",
		"updated_at":      "2024-02-01T00:00:00Z",
		"region":          "us",
		"organization_id": 100,
	},
}

func mockProjectsDataSourceServer(t *testing.T, organizationID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects", baseUrl, organizationID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Projects list request received")
			return httpmock.NewJsonResponse(200, projectsListResponse)
		})
}

func TestAccProjectsDataSource(t *testing.T) {
	organizationID := "test-organization"
	mockProjectsDataSourceServer(t, organizationID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectsDataSourceConfig(organizationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.quant_projects.test", "projects.#", "2"),
					resource.TestCheckResourceAttr("data.quant_projects.test", "projects.0.name", "Project Alpha"),
					resource.TestCheckResourceAttr("data.quant_projects.test", "projects.0.machine_name", "project-alpha"),
					resource.TestCheckResourceAttr("data.quant_projects.test", "projects.1.name", "Project Beta"),
					resource.TestCheckResourceAttr("data.quant_projects.test", "projects.1.machine_name", "project-beta"),
				),
			},
		},
	})
}

func testAccProjectsDataSourceConfig(organization string) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

data "quant_projects" "test" {}
`, organization)
}
