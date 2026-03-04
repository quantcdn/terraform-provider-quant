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

func testServeStaticPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupServeStaticRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Dynamic closure state for mock responses
	var currentName = "test-serve-static"
	var currentStaticFilePath = "/index.html"
	var currentDisabled = false

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":    "66666666-6666-5666-a666-666666666666",
			"rule_id": "66666666-6666-5666-a666-666666666666",
			"name":    currentName,
			"domain":  []string{"example.com"},
			"url":     []string{"/"},
			"action":  "serve_static",
			"action_config": map[string]interface{}{
				"static_file_path": currentStaticFilePath,
			},
			"method_is":      []string{},
			"method_is_not":  []string{},
			"ip_is":          []string{},
			"ip_is_not":      []string{},
			"country_is":     []string{},
			"country_is_not": []string{},

			"disabled":         currentDisabled,
			"weight":           0,
			"ip":               "any",
			"method":           "any",
			"country":          "any",
			"only_with_cookie": "",
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static/66666666-6666-5666-a666-666666666666", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
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
				if sfp, ok := requestBody["static_file_path"].(string); ok {
					currentStaticFilePath = sfp
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static/66666666-6666-5666-a666-666666666666", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
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
				if sfp, ok := requestBody["static_file_path"].(string); ok {
					currentStaticFilePath = sfp
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static/66666666-6666-5666-a666-666666666666", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
}

func TestAccRuleServeStaticResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupServeStaticRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testServeStaticPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRuleServeStaticResourceConfigMock(organizationID, projectID, "test-serve-static"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "name", "test-serve-static"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "static_file_path", "/index.html"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "url.0", "/"),
				),
			},
			// Update testing
			{
				Config: testAccRuleServeStaticResourceConfigUpdateMock(organizationID, projectID, "test-serve-static-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "name", "test-serve-static-updated"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "static_file_path", "/about.html"),
					resource.TestCheckResourceAttr("quant_rule_serve_static.test", "disabled", "true"),
				),
			},
			// Import testing
			{
				ResourceName:                         "quant_rule_serve_static.test",
				ImportState:                          true,
				ImportStateId:                        "default/66666666-6666-5666-a666-666666666666",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{
					"action_config",
				},
			},
		},
	})
}

func testAccRuleServeStaticResourceConfigMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_serve_static" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/"]
	static_file_path = "/index.html"
}
`, organizationID, projectID, name)
}

func testAccRuleServeStaticResourceConfigUpdateMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_serve_static" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/"]
	static_file_path = "/about.html"
	disabled = true
}
`, organizationID, projectID, name)
}
