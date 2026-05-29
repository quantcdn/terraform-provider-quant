package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
)

func setupCrawlerScheduleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	crawlerUUID := "test-crawler-uuid"
	scheduleID := "1"

	// Dynamic closure state
	var currentName = "test-schedule"
	var currentCron = "0 0 * * *"

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"id":                   1,
			"name":                 currentName,
			"project_id":           17,
			"schedule_cron_string": currentCron,
			"crawler_schedule":     currentCron,
			"crawler_uuid":         crawlerUUID,
			"crawler_config_id":    5,
			"crawler_last_run_id":  0,
			"created_at":           "2024-06-28T03:33:02.000000Z",
			"updated_at":           "2024-06-28T03:50:26.000000Z",
		}
	}

	// Log unhandled requests
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// Create schedule
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/%s/schedules", baseUrl, organizationID, projectID, crawlerUUID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Create crawler schedule request received")
			var requestBody map[string]interface{}
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
					if cron, ok := requestBody["schedule_cron_string"].(string); ok {
						currentCron = cron
					}
				}
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// Read schedule
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/%s/schedules/%s", baseUrl, organizationID, projectID, crawlerUUID, scheduleID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Get crawler schedule request received")
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// Update schedule (PATCH)
	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/%s/schedules/%s", baseUrl, organizationID, projectID, crawlerUUID, scheduleID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Update crawler schedule request received")
			var requestBody map[string]interface{}
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
					if cron, ok := requestBody["schedule_cron_string"].(string); ok {
						currentCron = cron
					}
				}
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// Delete schedule
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/%s/schedules/%s", baseUrl, organizationID, projectID, crawlerUUID, scheduleID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Delete crawler schedule request received")
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func testAccCrawlerScheduleResourceConfig(org, project, name, cron string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer       = "testtoken"
	organization = %[1]q
}

resource "quant_crawler_schedule" "test" {
	name                = %[3]q
	project             = %[2]q
	crawler             = "test-crawler-uuid"
	schedule_cron_string = %[4]q
}
`, org, project, name, cron)
}

func TestAccCrawlerScheduleResource_basic(t *testing.T) {
	setupCrawlerScheduleServer(t, "test-org", "test-project")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccCrawlerScheduleResourceConfig("test-org", "test-project", "test-schedule", "0 0 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "name", "test-schedule"),
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "schedule_cron_string", "0 0 * * *"),
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "project", "test-project"),
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "crawler_uuid", "test-crawler-uuid"),
					testAccCheckCrawlerScheduleExists("quant_crawler_schedule.test"),
				),
			},
			// Step 2: Import
			{
				ResourceName:            "quant_crawler_schedule.test",
				ImportState:             true,
				ImportStateId:           "test-project:test-crawler-uuid:1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"crawler"},
			},
			// Step 3: Update
			{
				Config: testAccCrawlerScheduleResourceConfig("test-org", "test-project", "updated-schedule", "0 12 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "name", "updated-schedule"),
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "schedule_cron_string", "0 12 * * *"),
					resource.TestCheckResourceAttr("quant_crawler_schedule.test", "project", "test-project"),
					testAccCheckCrawlerScheduleExists("quant_crawler_schedule.test"),
				),
			},
			// Step 4: Delete (implicit)
		},
	})
}

func testAccCheckCrawlerScheduleExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.Attributes["name"] == "" {
			return fmt.Errorf("No Crawler Schedule name is set")
		}

		if rs.Primary.Attributes["project"] == "" {
			return fmt.Errorf("No Crawler Schedule project is set")
		}

		return nil
	}
}

// ---------- Crawler Schedule HTTP error-path tests ----------

func setupCrawlerScheduleErrorResponder(t *testing.T, org string, project string, crawlerUUID string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/%s/schedules", baseUrl, org, project, crawlerUUID),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testCrawlerScheduleErrorConfig() string {
	return `
provider "quant" {
  organization = "test-org"
  bearer = "testtoken"
}

resource "quant_crawler_schedule" "test" {
  name                 = "error-schedule"
  project              = "test-project"
  crawler              = "test-crawler-uuid"
  schedule_cron_string = "0 0 * * *"
}
`
}

func TestAccCrawlerScheduleResource_CreateError(t *testing.T) {
	setupCrawlerScheduleErrorResponder(t, "test-org", "test-project", "test-crawler-uuid", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testCrawlerScheduleErrorConfig(),
				ExpectError: regexp.MustCompile(`Unable to create crawler schedule`),
			},
		},
	})
}
