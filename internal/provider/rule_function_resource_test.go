package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var functionRuleResponse = map[string]interface{}{
	"uuid":    "222222-2222-2222-2222-222222222222",
	"rule_id": "222222-2222-2222-2222-222222222222",
	"name":    "test-function",
	"domain":  []string{"example.com"},
	"url":     []string{"/fn/*"},
	"action":  "function",
	"action_config": map[string]interface{}{
		"fn_uuid": "fn-123",
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

func testFunctionPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupFunctionRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{functionRuleResponse})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function/222222-2222-2222-2222-222222222222", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, functionRuleResponse)
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, functionRuleResponse)
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/function/222222-2222-2222-2222-222222222222", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, functionRuleResponse)
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
