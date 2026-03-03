package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var headersRuleResponse = map[string]interface{}{
	"uuid":    "555555-5555-5555-5555-555555555555",
	"rule_id": "555555-5555-5555-5555-555555555555",
	"name":    "test-headers",
	"domain":  []string{"example.com"},
	"url":     []string{"/api/*"},
	"action":  "headers",
	"action_config": map[string]interface{}{
		"headers": map[string]interface{}{
			"X-Custom": "value",
		},
	},
	"method_is":      []string{},
	"method_is_not":  []string{},
	"ip_is":          []string{},
	"ip_is_not":      []string{},
	"country_is":     []string{},
	"country_is_not": []string{},

	"disabled":         false,
	"ip":               "any",
	"method":           "any",
	"country":          "any",
	"weight":           0,
	"only_with_cookie": "",
}

func testHeadersPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupHeadersRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{headersRuleResponse})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers/555555-5555-5555-5555-555555555555", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, headersRuleResponse)
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, headersRuleResponse)
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers/555555-5555-5555-5555-555555555555", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, headersRuleResponse)
	})
}

func TestAccRuleHeadersResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupHeadersRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testHeadersPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleHeadersResourceConfigMock(organizationID, projectID, "test-headers"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_headers.test", "name", "test-headers"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "headers.X-Custom", "value"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "url.0", "/api/*"),
				),
			},
		},
	})
}

func testAccRuleHeadersResourceConfigMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`

provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_headers" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/api/*"]
	headers = {
		"X-Custom" = "value"
	}
}
`, organizationID, projectID, name)
}
