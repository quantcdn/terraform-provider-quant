package provider_test

import (
	"fmt"
	"testing"
	"terraform-provider-quant/internal/provider"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var redirectRuleResponse = map[string]interface{}{
	"uuid": "111111-1111-1111-1111-111111111111",
	"rule_id": "111111-1111-1111-1111-111111111111",
	"name": "test-redirect",
	"domain": []string{"example.com"},
	"url": []string{"/old"},
	"action": "redirect",
	"action_config": map[string]interface{}{
		"to": "https://example.com/new",
		"status_code": "301",
	},
	"method_is": []string{},
	"method_is_not": []string{},
	"ip_is": []string{},
	"ip_is_not": []string{},
	"country_is": []string{"AU"},
	"country_is_not": []string{},

	"disabled": false,
	"ip": "any",
	"method": "any",
	"country": "country_is",
	"only_with_cookie": "",
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"quant": providerserver.NewProtocol6WithError(provider.New()()),
}

func testAccPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupRedirectRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"
	// https://dashboard.quantcdn.io/api/v2/organizations/test-organization/projects/test-redirect/rules/redirect
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{redirectRuleResponse})
	})
	
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect/111111-1111-1111-1111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, redirectRuleResponse)
	})
	
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, redirectRuleResponse)
	})
	
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/redirect/111111-1111-1111-1111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, redirectRuleResponse)
	})
}


func TestAccRuleRedirectResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupRedirectRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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