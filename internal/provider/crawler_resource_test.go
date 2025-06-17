package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"net/http"
)

func testAccCrawlerPreCheck(t *testing.T) {
	// You can add any additional setup here
}

var crawlerResponse = map[string]interface{}{
	"id":         5,
	"project_id": 17,
	"uuid":       "29f1141b-ded6-483b-9a14-4439db01bc22",
	"name":       "SDK TF crawler 1719545580",
	"config":     "config:\n    user_agent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.83 Safari/537.36'\n    browser_mode: true\n    workers: 2\n    depth: -1\n    max_hits: 0\n    max_html: 500\n    cache: false\n    delay: 4\n    status_ok: [200]\n    quant: { options: { enabled: true, max_errors: 100 } }\n    start_url: [/start-here]\n    headers: { x-test-heaer: 'true', x-test-other-header: 'false' }\ndomain: 'https://www.quantcdn.io'\nheaders: {  }\n",
	"urls_list":  "single_url: {  }\n",
	"created_at": "2024-06-28T03:33:02.000000Z",
	"updated_at": "2024-06-28T03:50:26.000000Z",
	"domain":     "https://www.quantcdn.io",
	"domain_verified": 0,
	"webhook_url": nil,
	"webhook_auth_header": nil,
	"webhook_extra_vars": nil,
}

var crawlerResponseWithExclude = map[string]interface{}{
	"id":         5,
	"project_id": 17,
	"uuid":       "29f1141b-ded6-483b-9a14-4439db01bc22",
	"name":       "SDK TF crawler 1719545580",
	"config":     "config:\n    user_agent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.83 Safari/537.36'\n    browser_mode: true\n    workers: 2\n    depth: -1\n    max_hits: 0\n    max_html: 500\n    cache: false\n    delay: 4\n    status_ok: [200]\n    quant: { options: { enabled: true, max_errors: 100 } }\n    start_url: [/start-here]\n    exclude: [/exclude-path]\n    headers: { x-test-heaer: 'true', x-test-other-header: 'false' }\ndomain: 'https://www.quantcdn.io'\nheaders: {  }\n",
	"urls_list":  "single_url: {  }\n",
	"created_at": "2024-06-28T03:33:02.000000Z",
	"updated_at": "2024-06-28T03:50:26.000000Z",
	"domain":     "https://www.quantcdn.io",
	"domain_verified": 0,
	"webhook_url": nil,
	"webhook_auth_header": nil,
	"webhook_extra_vars": nil,
}

func setupCrawlerServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Log all requests for debugging
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// List crawlers
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("List crawlers request received")
			return httpmock.NewJsonResponse(200, []map[string]interface{}{crawlerResponse})
		})

	// Get crawler by UUID
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/29f1141b-ded6-483b-9a14-4439db01bc22", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Get crawler request received")
			return httpmock.NewJsonResponse(200, crawlerResponse)
		})

	// Create crawler
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Create crawler request received")
			return httpmock.NewJsonResponse(200, crawlerResponse)
		})

	// Update crawler - make sure this exactly matches the URL pattern used by the API
	httpmock.RegisterResponder("PUT", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/29f1141b-ded6-483b-9a14-4439db01bc22", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Update crawler request received: %s", req.URL.String())
			return httpmock.NewJsonResponse(200, crawlerResponse)
		})

	// Also register with PATCH in case the API uses PATCH instead of PUT
	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/29f1141b-ded6-483b-9a14-4439db01bc22", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Patch crawler request received: %s", req.URL.String())
			return httpmock.NewJsonResponse(200, crawlerResponseWithExclude)
		})

	// Delete crawler - using UUID as shown in the OpenAPI spec
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/crawlers/29f1141b-ded6-483b-9a14-4439db01bc22", baseUrl, organizationID, projectID),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Delete crawler request received")
			// Return the crawler object as per OpenAPI spec (delete returns the deleted crawler)
			return httpmock.NewJsonResponse(200, crawlerResponse)
		})
}

func TestAccCrawlerResourceMock(t *testing.T) {
	setupCrawlerServer(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccCrawlerPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCrawlerResourceConfigMock(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_crawler.test", "name", "SDK TF crawler 1719545580"),
					resource.TestCheckResourceAttr("quant_crawler.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_crawler.test", "domain", "https://www.quantcdn.io"),
					resource.TestCheckResourceAttr("quant_crawler.test", "browser_mode", "true"),
					resource.TestCheckResourceAttr("quant_crawler.test", "domain_verified", "0"),
					resource.TestCheckResourceAttr("quant_crawler.test", "headers.x-test-heaer", "true"),
					resource.TestCheckResourceAttr("quant_crawler.test", "headers.x-test-other-header", "false"),
					testAccCheckCrawlerExists("quant_crawler.test"),
				),
			},
			{
				// Test update
				Config: testAccCrawlerResourceConfigUpdateMock(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_crawler.test", "name", "SDK TF crawler 1719545580"),
					resource.TestCheckResourceAttr("quant_crawler.test", "domain", "https://www.quantcdn.io"),
					resource.TestCheckResourceAttr("quant_crawler.test", "browser_mode", "true"),
					resource.TestCheckResourceAttr("quant_crawler.test", "exclude.#", "1"),
					resource.TestCheckResourceAttr("quant_crawler.test", "exclude.0", "/exclude-path"),
					testAccCheckCrawlerExists("quant_crawler.test"),
				),
			},
		},
	})
}

func testAccCrawlerResourceConfigMock() string {
	return `
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_crawler" "test" {
	name    = "SDK TF crawler 1719545580"
	project = "default"
	domain  = "https://www.quantcdn.io"
	browser_mode = true
	headers = {
		"x-test-heaer" = "true"
		"x-test-other-header" = "false"
	}
	urls = ["/start-here"]
}
`
}

func testAccCrawlerResourceConfigUpdateMock() string {
	return `
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_crawler" "test" {
	name    = "SDK TF crawler 1719545580"
	project = "default"
	domain  = "https://www.quantcdn.io"
	browser_mode = true
	headers = {
		"x-test-heaer" = "true"
		"x-test-other-header" = "false"
	}
	urls = ["/start-here"]
	exclude = ["/exclude-path"]
}
`
}

func testAccCheckCrawlerExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		// In Plugin Framework v6, check for the UUID attribute instead of the generic ID
		if rs.Primary.Attributes["uuid"] == "" {
			return fmt.Errorf("No Crawler UUID is set")
		}

		// Also verify we have the basic required attributes
		if rs.Primary.Attributes["project"] == "" {
			return fmt.Errorf("No Crawler project is set")
		}

		// Here you would typically make an API call to verify the resource exists
		// Instead, we'll just return nil since we're mocking
		return nil
	}
}
