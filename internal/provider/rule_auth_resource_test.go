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

func testAuthPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupAuthRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Dynamic closure state
	var currentName = "test-auth"
	var currentAuthUser = "admin"
	var currentAuthPass = "secret"
	var currentDisabled = false

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":    "33333333-3333-4333-a333-333333333333",
			"rule_id": "33333333-3333-4333-a333-333333333333",
			"name":    currentName,
			"domain":  []string{"example.com"},
			"url":     []string{"/admin/*"},
			"action":  "auth",
			"action_config": map[string]interface{}{
				"auth_user": currentAuthUser,
				"auth_pass": currentAuthPass,
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

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth/33333333-3333-4333-a333-333333333333", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if authUser, ok := requestBody["auth_user"].(string); ok {
					currentAuthUser = authUser
				}
				if authPass, ok := requestBody["auth_pass"].(string); ok {
					currentAuthPass = authPass
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth/33333333-3333-4333-a333-333333333333", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if authUser, ok := requestBody["auth_user"].(string); ok {
					currentAuthUser = authUser
				}
				if authPass, ok := requestBody["auth_pass"].(string); ok {
					currentAuthPass = authPass
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/auth/33333333-3333-4333-a333-333333333333", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
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
			// Create
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
			// Update
			{
				Config: testAccRuleAuthResourceConfigUpdateMock(organizationID, projectID, "test-auth-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_auth.test", "name", "test-auth-updated"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "auth_user", "newadmin"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "auth_pass", "newsecret"),
					resource.TestCheckResourceAttr("quant_rule_auth.test", "disabled", "true"),
				),
			},
			// Import
			{
				ResourceName:                         "quant_rule_auth.test",
				ImportState:                          true,
				ImportStateId:                        "default/33333333-3333-4333-a333-333333333333",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore: []string{
					"auth_pass",
				},
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

func testAccRuleAuthResourceConfigUpdateMock(organizationID string, projectID string, name string) string {
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
	auth_user = "newadmin"
	auth_pass = "newsecret"
	disabled = true
}
`, organizationID, projectID, name)
}
