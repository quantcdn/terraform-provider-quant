package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

func testHeadersPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupHeadersRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Dynamic closure state for mock responses
	var currentName = "test-headers"
	var currentHeaders = map[string]interface{}{"X-Custom": "value"}
	var currentDomain = []string{"example.com"}
	var currentUrl = []string{"/api/*"}
	var currentDisabled = false

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":             "55555555-5555-5555-a555-555555555555",
			"rule_id":          "55555555-5555-5555-a555-555555555555",
			"name":             currentName,
			"domain":           currentDomain,
			"url":              currentUrl,
			"action":           "headers",
			"disabled":         currentDisabled,
			"weight":           0,
			"method":           "any",
			"method_is":        []string{},
			"method_is_not":    []string{},
			"country":          "any",
			"country_is":       []string{},
			"country_is_not":   []string{},
			"ip":               "any",
			"ip_is":            []string{},
			"ip_is_not":        []string{},
			"only_with_cookie": "",
			"action_config": map[string]interface{}{
				"headers": currentHeaders,
			},
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers/55555555-5555-5555-a555-555555555555", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
				if headers, ok := requestBody["headers"].(map[string]interface{}); ok {
					currentHeaders = headers
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers/55555555-5555-5555-a555-555555555555", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
				if headers, ok := requestBody["headers"].(map[string]interface{}); ok {
					currentHeaders = headers
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/headers/55555555-5555-5555-a555-555555555555", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
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
			// Create and Read testing
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
			// Update testing
			{
				Config: testAccRuleHeadersResourceConfigUpdateMock(organizationID, projectID, "test-headers-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_headers.test", "name", "test-headers-updated"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "headers.X-Updated", "new-value"),
					resource.TestCheckResourceAttr("quant_rule_headers.test", "disabled", "true"),
				),
			},
			// Import testing
			{
				ResourceName:                         "quant_rule_headers.test",
				ImportState:                          true,
				ImportStateId:                        "default/55555555-5555-5555-a555-555555555555",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{
					"action_config",
				},
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

func testAccRuleHeadersResourceConfigUpdateMock(organizationID string, projectID string, name string) string {
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
	disabled = true
	headers = {
		"X-Updated" = "new-value"
	}
}
`, organizationID, projectID, name)
}
