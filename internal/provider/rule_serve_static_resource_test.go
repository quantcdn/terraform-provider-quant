package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var serveStaticRuleResponse = map[string]interface{}{
	"uuid":    "666666-6666-6666-6666-666666666666",
	"rule_id": "666666-6666-6666-6666-666666666666",
	"name":    "test-serve-static",
	"domain":  []string{"example.com"},
	"url":     []string{"/"},
	"action":  "serve_static",
	"action_config": map[string]interface{}{
		"static_file_path": "/index.html",
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

func testServeStaticPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupServeStaticRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{serveStaticRuleResponse})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static/666666-6666-6666-6666-666666666666", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, serveStaticRuleResponse)
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, serveStaticRuleResponse)
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/serve-static/666666-6666-6666-6666-666666666666", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, serveStaticRuleResponse)
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
