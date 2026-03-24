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

var cronJobResponse = map[string]interface{}{
	"name":               "backup",
	"scheduleExpression": "0 * * * *",
	"command":            []string{"echo", "hello"},
}

func mockCronJobServer(t *testing.T, org string, app string, env string, cronName string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	cronDeleted := false

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create cron job
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/cron", baseUrl, org, app, env),
		func(req *http.Request) (*http.Response, error) {
			cronDeleted = false
			return httpmock.NewJsonResponse(200, cronJobResponse)
		})

	// GET cron job
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/cron/%s", baseUrl, org, app, env, cronName),
		func(req *http.Request) (*http.Response, error) {
			if cronDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, cronJobResponse)
		})

	// PATCH update cron job
	httpmock.RegisterResponder("PATCH",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/cron/%s", baseUrl, org, app, env, cronName),
		func(req *http.Request) (*http.Response, error) {
			if cronDeleted {
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

			if v, ok := requestBody["scheduleExpression"]; ok {
				cronJobResponse["scheduleExpression"] = v
			}
			if v, ok := requestBody["command"]; ok {
				cronJobResponse["command"] = v
			}

			return httpmock.NewJsonResponse(200, cronJobResponse)
		})

	// DELETE cron job
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/cron/%s", baseUrl, org, app, env, cronName),
		func(req *http.Request) (*http.Response, error) {
			if cronDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			cronDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccCronJobResource(t *testing.T) {
	org := "test-org"
	app := "test-app"
	env := "production"
	cronName := "backup"
	mockCronJobServer(t, org, app, env, cronName)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCronJobResourceConfig(org, app, env, cronName, "0 * * * *", `["echo","hello"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_cron_job.test", "name", "backup"),
					resource.TestCheckResourceAttr("quant_cron_job.test", "application", "test-app"),
					resource.TestCheckResourceAttr("quant_cron_job.test", "environment", "production"),
					resource.TestCheckResourceAttr("quant_cron_job.test", "schedule", "0 * * * *"),
					resource.TestCheckResourceAttr("quant_cron_job.test", "command", `["echo","hello"]`),
				),
			},
			// Import testing
			{
				ResourceName:  "quant_cron_job.test",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s/%s/%s", app, env, cronName),
				ImportStateVerifyIgnore: []string{
					"schedule_expression",
					"description",
					"target_container_name",
					"is_enabled",
					"organization",
				},
			},
			// Update and Read testing
			{
				Config: testAccCronJobResourceConfig(org, app, env, cronName, "*/5 * * * *", `["echo","updated"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_cron_job.test", "name", "backup"),
					resource.TestCheckResourceAttr("quant_cron_job.test", "schedule_expression", "*/5 * * * *"),
					resource.TestCheckResourceAttr("quant_cron_job.test", "command", `["echo","updated"]`),
				),
			},
		},
	})
}

func testAccCronJobResourceConfig(org string, app string, env string, name string, schedule string, command string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_cron_job" "test" {
  organization        = %[1]q
  application         = %[2]q
  environment         = %[3]q
  name                = %[4]q
  schedule_expression = %[5]q
  command             = %[6]q
}
`, org, app, env, name, schedule, command)
}

// ---------- HTTP error-path integration tests ----------

func setupCronJobErrorResponder(t *testing.T, org string, app string, env string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v3"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/applications/%s/environments/%s/cron", baseUrl, org, app, env),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testCronJobErrorConfig() string {
	return `
provider "quant" {
  organization = "test-org"
  bearer = "testtoken"
}

resource "quant_cron_job" "test" {
  organization        = "test-org"
  application         = "test-app"
  environment         = "production"
  name                = "error-test"
  schedule_expression = "0 * * * *"
  command             = "[\"echo\",\"hello\"]"
}
`
}

func TestAccCronJobResource_CreateError401(t *testing.T) {
	setupCronJobErrorResponder(t, "test-org", "test-app", "production", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testCronJobErrorConfig(),
				ExpectError: regexp.MustCompile(`Authentication Failed`),
			},
		},
	})
}

func TestAccCronJobResource_CreateError500(t *testing.T) {
	setupCronJobErrorResponder(t, "test-org", "test-app", "production", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testCronJobErrorConfig(),
				ExpectError: regexp.MustCompile(`Unable to Create Cron Job`),
			},
		},
	})
}
