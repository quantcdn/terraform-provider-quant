package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var authRuleResponse = map[string]interface{}{
	"uuid":    "333333-3333-3333-3333-333333333333",
	"rule_id": "333333-3333-3333-3333-333333333333",
	"name":    "test-auth",
	"domain":  []string{"example.com"},
	"url":     []string{"/admin/*"},
	"action":  "auth",
	"action_config": map[string]interface{}{
		"auth_user": "admin",
		"auth_pass": "secret",
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

func testAuthPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupAuthRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{authRuleResponse})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth/333333-3333-3333-3333-333333333333", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, authRuleResponse)
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, authRuleResponse)
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth/333333-3333-3333-3333-333333333333", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, authRuleResponse)
	})
}

func TestAccRuleAuthResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupAuthRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAuthPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleAuthResourceConfigMock(organizationID, projectID, "test-auth"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_auth.test", "name", "test-auth"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "auth_user", "admin"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "auth_pass", "secret"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "url.0", "/admin/*"),
				),
			},
		},
	})
}

func testAccRuleAuthResourceConfigMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`

provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_auth" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/admin/*"]
	auth_user = "admin"
	auth_pass = "secret"
}
`, organizationID, projectID, name)
}
