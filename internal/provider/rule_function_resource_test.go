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

func testFunctionPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupFunctionRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Dynamic closure state
	var currentName = "test-function"
	var currentFnUuid = "fn-123"
	var currentDisabled = false

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":    "22222222-2222-4222-a222-222222222222",
			"rule_id": "22222222-2222-4222-a222-222222222222",
			"name":    currentName,
			"domain":  []string{"example.com"},
			"url":     []string{"/fn/*"},
			"action":  "function",
			"action_config": map[string]interface{}{
				"fn_uuid": currentFnUuid,
			},
			"method_is":      []string{},
			"method_is_not":  []string{},
			"ip_is":          []string{},
			"ip_is_not":      []string{},
			"country_is":     []string{},
			"country_is_not": []string{},

			"disabled":         currentDisabled,
			"ip":               "any",
			"method":           "any",
			"country":          "any",
			"weight":           0,
			"only_with_cookie": "",
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function/22222222-2222-4222-a222-222222222222", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if fnUuid, ok := requestBody["fn_uuid"].(string); ok {
					currentFnUuid = fnUuid
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function/22222222-2222-4222-a222-222222222222", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if fnUuid, ok := requestBody["fn_uuid"].(string); ok {
					currentFnUuid = fnUuid
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function/22222222-2222-4222-a222-222222222222", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
}

func TestAccRuleFunctionResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupFunctionRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testFunctionPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccRuleFunctionResourceConfigMock(organizationID, projectID, "test-function"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_function.test", "name", "test-function"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "fn_uuid", "fn-123"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "url.0", "/fn/*"),
				),
			},
			// Update
			{
				Config: testAccRuleFunctionResourceConfigUpdateMock(organizationID, projectID, "test-function-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_function.test", "name", "test-function-updated"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "fn_uuid", "fn-456"),
					resource.TestCheckResourceAttr("quant_rule_function.test", "disabled", "true"),
				),
			},
			// Import
			{
				ResourceName:                         "quant_rule_function.test",
				ImportState:                          true,
				ImportStateId:                        "default/22222222-2222-4222-a222-222222222222",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func testAccRuleFunctionResourceConfigMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`

provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_function" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/fn/*"]
	fn_uuid = "fn-123"
}
`, organizationID, projectID, name)
}

func testAccRuleFunctionResourceConfigUpdateMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`

provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_function" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/fn/*"]
	fn_uuid = "fn-456"
	disabled = true
}
`, organizationID, projectID, name)
}
