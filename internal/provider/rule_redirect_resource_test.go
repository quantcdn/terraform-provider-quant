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

func testRedirectPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupRedirectRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Dynamic closure state for mock responses
	var currentName = "test-redirect"
	var currentRedirectTo = "https://example.com/new"
	var currentRedirectCode = "301"
	var currentDisabled = false

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":    "11111111-1111-1111-a111-111111111111",
			"rule_id": "11111111-1111-1111-a111-111111111111",
			"name":    currentName,
			"domain":  []string{"example.com"},
			"url":     []string{"/old"},
			"action":  "redirect",
			"action_config": map[string]interface{}{
				"to":          currentRedirectTo,
				"status_code": currentRedirectCode,
			},
			"method_is":      []string{},
			"method_is_not":  []string{},
			"ip_is":          []string{},
			"ip_is_not":      []string{},
			"country_is":     []string{"AU"},
			"country_is_not": []string{},

			"disabled":         currentDisabled,
			"weight":           0,
			"ip":               "any",
			"method":           "any",
			"country":          "country_is",
			"only_with_cookie": "",
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect/11111111-1111-1111-a111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
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
				if to, ok := requestBody["redirect_to"].(string); ok {
					currentRedirectTo = to
				}
				if code, ok := requestBody["redirect_code"].(string); ok {
					currentRedirectCode = code
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect/11111111-1111-1111-a111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
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
				if to, ok := requestBody["redirect_to"].(string); ok {
					currentRedirectTo = to
				}
				if code, ok := requestBody["redirect_code"].(string); ok {
					currentRedirectCode = code
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect/11111111-1111-1111-a111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
}

func TestAccRuleRedirectResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupRedirectRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testRedirectPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRuleRedirectResourceConfigMock(organizationID, projectID, "test-redirect"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "name", "test-redirect"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "url.0", "/old"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "redirect_to", "https://example.com/new"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "redirect_code", "301"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "only_with_cookie", ""),
				),
			},
			// Update testing
			{
				Config: testAccRuleRedirectResourceConfigUpdateMock(organizationID, projectID, "test-redirect-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "name", "test-redirect-updated"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "redirect_to", "https://example.com/updated"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "redirect_code", "302"),
					resource.TestCheckResourceAttr("quant_rule_redirect.test", "disabled", "true"),
				),
			},
			// Import testing
			{
				ResourceName:                         "quant_rule_redirect.test",
				ImportState:                          true,
				ImportStateId:                        "default/11111111-1111-1111-a111-111111111111",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{
					"action_config",
				},
			},
		},
	})
}

func testAccRuleRedirectResourceConfigMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_redirect" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/old"]
	redirect_to = "https://example.com/new"
	redirect_code = 301
}
`, organizationID, projectID, name)
}

func testAccRuleRedirectResourceConfigUpdateMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_redirect" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/old"]
	redirect_to = "https://example.com/updated"
	redirect_code = 302
	disabled = true
}
`, organizationID, projectID, name)
}
