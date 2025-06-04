package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a new provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"quant": providerserver.NewProtocol6WithError(New()()),
}

var customResponseResponse = map[string]interface{}{
	"uuid": "96e4f4f6-211a-4f4b-b7fb-03985df56dad",
	"rule_id": "96e4f4f6-211a-4f4b-b7fb-03985df56dad",
	"domain": []string{"any"},
	"country": "country_is",
	"country_is": []string{"AF"},
	"country_is_not": []string{},
	"method": "any",
	"method_is": []string{},
	"method_is_not": []string{},
	"ip": "any",
	"ip_is": []string{},
	"ip_is_not": []string{},
	"only_with_cookie": "",
	"url": []string{"/test"},
	"name": "custom response",
	"disabled": false,
	"action": "custom_response",
	"action_config": map[string]interface{}{
		"custom_response_status_code": 200,
		"custom_response_body": "<h1>test</h1>",
	},
}

func mockCustomResponseServer(t *testing.T, organizationID string, ruleID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"
	uuid := "96e4f4f6-211a-4f4b-b7fb-03985df56dad"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("No responder found for request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/custom-response", baseUrl, organizationID, organizationID),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, customResponseResponse)
		})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/custom-response/%s", baseUrl, organizationID, organizationID, uuid),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, customResponseResponse)
		})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/custom-response/%s", baseUrl, organizationID, organizationID, uuid),
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			var requestBody struct {
				CustomResponseStatusCode int32  `json:"custom_response_status_code"`
				CustomResponseBody      string `json:"custom_response_body"`
			}

			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			response := customResponseResponse
			response["action_config"] = map[string]interface{}{
				"custom_response_status_code": requestBody.CustomResponseStatusCode,
				"custom_response_body":        requestBody.CustomResponseBody,
			}
			return httpmock.NewJsonResponse(200, response)
		})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/custom-response/%s", baseUrl, organizationID, organizationID, uuid),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(204, ""), nil
		})
}

func testCustomResponseResourceConfigMock(organizationID string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

resource "quant_rule_custom_response" "test" {
  project = %[1]q
  name = "custom response"
  domain = ["any"]
  url = ["/test"]
  custom_response_status_code = 200
  custom_response_body = "<h1>test</h1>"
}
`, organizationID, name)
}

func testCustomResponseResourceConfigUpdateMock(organizationID string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

resource "quant_rule_custom_response" "test" {
  project = %[1]q
  name = "custom response"
  domain = ["any"]
  url = ["/test"]
  custom_response_status_code = 500
  custom_response_body = "<h1>Internal Server Error</h1>"
}
`, organizationID, name)
}

func TestAccRuleCustomResponseResource(t *testing.T) {
	organizationID := "test-organization"
	ruleID := "96e4f4f6-211a-4f4b-b7fb-03985df56dad"
	mockCustomResponseServer(t, organizationID, ruleID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testCustomResponseResourceConfigMock(organizationID, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "rule_id", ruleID),
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "custom_response_status_code", "200"),
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "custom_response_body", "<h1>test</h1>"),
				),
			},
			{
				ResourceName:            "quant_rule_custom_response.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s/%s", organizationID, ruleID),
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestAccRuleCustomResponseResource_Update(t *testing.T) {
	organizationID := "test-organization"
	ruleID := "96e4f4f6-211a-4f4b-b7fb-03985df56dad"
	mockCustomResponseServer(t, organizationID, ruleID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testCustomResponseResourceConfigMock(organizationID, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "custom_response_status_code", "200"),
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "custom_response_body", "<h1>test</h1>"),
				),
			},
			{
				Config: testCustomResponseResourceConfigUpdateMock(organizationID, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "custom_response_status_code", "500"),
					resource.TestCheckResourceAttr("quant_rule_custom_response.test", "custom_response_body", "<h1>Internal Server Error</h1>"),
				),
			},
		},
	})
} 