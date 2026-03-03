package provider_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var volumeResponse = map[string]interface{}{
	"volumeId":         "vol-123",
	"volumeName":       "data",
	"description":      "test vol",
	"environmentEfsId": "efs-123",
	"createdAt":        "2024-01-01",
	"rootDirectory":    "/data",
	"accessPointId":    "ap-123",
	"accessPointArn":   "arn:aws:elasticfilesystem:us-east-1:123456789012:access-point/ap-123",
}

func mockVolumeServer(t *testing.T, org string, app string, env string, volumeId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	volDeleted := false

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create volume
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/volumes", baseUrl, org, app, env),
		func(req *http.Request) (*http.Response, error) {
			volDeleted = false
			return httpmock.NewJsonResponse(200, volumeResponse)
		})

	// GET volume by ID
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/volumes/%s", baseUrl, org, app, env, volumeId),
		func(req *http.Request) (*http.Response, error) {
			if volDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, volumeResponse)
		})

	// DELETE volume by ID
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/volumes/%s", baseUrl, org, app, env, volumeId),
		func(req *http.Request) (*http.Response, error) {
			if volDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			volDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccVolumeResource(t *testing.T) {
	org := "test-org"
	app := "test-app"
	env := "production"
	volumeId := "vol-123"
	mockVolumeServer(t, org, app, env, volumeId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVolumeResourceConfig(org, app, env),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_volume.test", "volume_name", "data"),
					resource.TestCheckResourceAttr("quant_volume.test", "volume_id", "vol-123"),
					resource.TestCheckResourceAttr("quant_volume.test", "application", "test-app"),
					resource.TestCheckResourceAttr("quant_volume.test", "environment", "production"),
					resource.TestCheckResourceAttr("quant_volume.test", "description", "test vol"),
					resource.TestCheckResourceAttr("quant_volume.test", "environment_efs_id", "efs-123"),
					resource.TestCheckResourceAttr("quant_volume.test", "root_directory", "/data"),
					resource.TestCheckResourceAttr("quant_volume.test", "access_point_id", "ap-123"),
					resource.TestCheckResourceAttr("quant_volume.test", "access_point_arn", "arn:aws:elasticfilesystem:us-east-1:123456789012:access-point/ap-123"),
					resource.TestCheckResourceAttr("quant_volume.test", "created_at", "2024-01-01"),
				),
			},
			// Import testing
			{
				ResourceName:  "quant_volume.test",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s/%s/%s", app, env, volumeId),
				ImportStateVerifyIgnore: []string{
					"organization",
				},
			},
		},
	})
}

func testAccVolumeResourceConfig(org string, app string, env string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_volume" "test" {
  organization = %[1]q
  application  = %[2]q
  environment  = %[3]q
  volume_name  = "data"
}
`, org, app, env)
}
