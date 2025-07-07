package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
)

func testAccRuleContentFilterPreCheck(t *testing.T) {
	// You can add any additional setup here
}

var ruleContentFilterResponse = map[string]interface{}{
	"uuid":             "5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
	"rule_id":          "5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
	"domain":           []string{"any"},
	"url":              []string{"/api/*"},
	"name":             "test-content-filter",
	"action":           "content_filter",
	"disabled":         false,
	"weight":           0,
	"method":           "",
	"method_is":        []string{},
	"method_is_not":    []string{},
	"country":          "country_is",
	"country_is":       []string{"US", "CA"},
	"country_is_not":   []string{},
	"ip":               "",
	"ip_is":            []string{},
	"ip_is_not":        []string{},
	"only_with_cookie": "",
	"action_config": map[string]interface{}{
		"fn_uuid": "function-uuid-12345",
	},
}

func setupRuleContentFilterServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{ruleContentFilterResponse})
	})
	
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleContentFilterResponse)
	})
	
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		// Parse the request to verify the sent data
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				t.Logf("Create request body: %+v", requestBody)
				
				// Verify required fields are present
				if fnUuid, ok := requestBody["fn_uuid"].(string); ok && fnUuid != "" {
					// Update response with the sent data
					response := make(map[string]interface{})
					for k, v := range ruleContentFilterResponse {
						response[k] = v
					}
					if name, ok := requestBody["name"].(string); ok {
						response["name"] = name
					}
					if actionConfig, ok := response["action_config"].(map[string]interface{}); ok {
						actionConfig["fn_uuid"] = fnUuid
					}
					if disabled, ok := requestBody["disabled"].(bool); ok {
						response["disabled"] = disabled
					}
					return httpmock.NewJsonResponse(200, response)
				}
			}
		}
		return httpmock.NewJsonResponse(200, ruleContentFilterResponse)
	})
	
	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		// Parse the request to verify the sent data
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				t.Logf("Update request body: %+v", requestBody)
				
				// Update response with the sent data
				response := make(map[string]interface{})
				for k, v := range ruleContentFilterResponse {
					response[k] = v
				}
				if name, ok := requestBody["name"].(string); ok {
					response["name"] = name
				}
				if actionConfig, ok := response["action_config"].(map[string]interface{}); ok {
					if fnUuid, ok := requestBody["fn_uuid"].(string); ok {
						actionConfig["fn_uuid"] = fnUuid
					}
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					response["disabled"] = disabled
				}
				return httpmock.NewJsonResponse(200, response)
			}
		}
		return httpmock.NewJsonResponse(200, ruleContentFilterResponse)
	})
	
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
}

func TestAccRuleContentFilterResourceMock(t *testing.T) {
	setupRuleContentFilterServer(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccRuleContentFilterPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleContentFilterResourceConfigMock("test-content-filter"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "name", "test-content-filter"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "project", "default"),
					// Domain checks
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "domain.0", "any"),
					// URL checks
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "url.0", "/api/*"),
					// Content filter config checks
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "fn_uuid", "function-uuid-12345"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "disabled", "false"),
					// Country checks
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "country", "country_is"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "country_is.#", "2"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "country_is.0", "US"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "country_is.1", "CA"),
					// Basic rule attributes
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "action", "content_filter"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "weight", "0"),
					testAccCheckRuleContentFilterExists("quant_rule_content_filter.test"),
				),
			},
			// Test import
			{
				ResourceName:                         "quant_rule_content_filter.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "default/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func TestAccRuleContentFilterResourceUpdate(t *testing.T) {
	setupRuleContentFilterServerForUpdate(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccRuleContentFilterPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial resource
			{
				Config: testAccRuleContentFilterResourceConfigMock("test-content-filter"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "name", "test-content-filter"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "fn_uuid", "function-uuid-12345"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "disabled", "false"),
				),
			},
			// Update the resource
			{
				Config: testAccRuleContentFilterResourceConfigUpdateMock("test-content-filter-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "name", "test-content-filter-updated"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "fn_uuid", "function-uuid-67890"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "disabled", "true"),
				),
			},
		},
	})
}

func TestAccRuleContentFilterResourceWithMethodsAndIPs(t *testing.T) {
	setupRuleContentFilterServerForMethodsAndIPs(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccRuleContentFilterPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleContentFilterResourceConfigWithMethodsAndIPs("test-content-filter-complex"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "name", "test-content-filter-complex"),
					// Method checks
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "method", "method_is"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "method_is.#", "2"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "method_is.0", "GET"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "method_is.1", "POST"),
					// IP checks
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "ip", "ip_is"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "ip_is.#", "2"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "ip_is.0", "192.168.1.1"),
					resource.TestCheckResourceAttr("quant_rule_content_filter.test", "ip_is.1", "192.168.1.2"),
					testAccCheckRuleContentFilterExists("quant_rule_content_filter.test"),
				),
			},
		},
	})
}

func testAccRuleContentFilterResourceConfigMock(name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_rule_content_filter" "test" {
	name     = %[1]q
	project  = "default"
	domain   = ["any"]
	url      = ["/api/*"]
	
	fn_uuid  = "function-uuid-12345"
	disabled = false
	
	country    = "country_is"
	country_is = ["US", "CA"]
}
`, name)
}

func testAccRuleContentFilterResourceConfigUpdateMock(name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_rule_content_filter" "test" {
	name     = %[1]q
	project  = "default"
	domain   = ["any"]
	url      = ["/api/*"]
	
	fn_uuid  = "function-uuid-67890"
	disabled = true
	
	country    = "country_is"
	country_is = ["US", "CA"]
}
`, name)
}

func testAccRuleContentFilterResourceConfigWithMethodsAndIPs(name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_rule_content_filter" "test" {
	name     = %[1]q
	project  = "default"
	domain   = ["any"]
	url      = ["/api/*"]
	
	fn_uuid  = "function-uuid-complex"
	disabled = false
	
	method    = "method_is"
	method_is = ["GET", "POST"]
	
	ip    = "ip_is"
	ip_is = ["192.168.1.1", "192.168.1.2"]
}
`, name)
}

func testAccCheckRuleContentFilterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Rule Content Filter ID is set")
		}

		// Here you would typically make an API call to verify the resource exists
		// Instead, we'll just return nil since we're mocking
		return nil
	}
}

func setupRuleContentFilterServerForUpdate(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Store the current test state to make responses dynamic
	var currentName = "test-content-filter"
	var currentFnUuid = "function-uuid-12345"
	var currentDisabled = false

	// Create a dynamic response based on current state
	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":             "5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
			"rule_id":          "5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
			"domain":           []string{"any"},
			"url":              []string{"/api/*"},
			"name":             currentName,
			"action":           "content_filter",
			"disabled":         currentDisabled,
			"weight":           0,
			"method":           "",
			"method_is":        []string{},
			"method_is_not":    []string{},
			"country":          "country_is",
			"country_is":       []string{"US", "CA"},
			"country_is_not":   []string{},
			"ip":               "",
			"ip_is":            []string{},
			"ip_is_not":        []string{},
			"only_with_cookie": "",
			"action_config": map[string]interface{}{
				"fn_uuid": currentFnUuid,
			},
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// Dynamic responses that change based on request
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})
	
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})
	
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		// Parse the request to extract values
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
	
	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		// Parse the request to extract values
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
	
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
}

func setupRuleContentFilterServerForMethodsAndIPs(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	responseWithMethodsAndIPs := map[string]interface{}{
		"uuid":             "5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
		"rule_id":          "5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0",
		"domain":           []string{"any"},
		"url":              []string{"/api/*"},
		"name":             "test-content-filter-complex",
		"action":           "content_filter",
		"disabled":         false,
		"weight":           0,
		"method":           "method_is",
		"method_is":        []string{"GET", "POST"},
		"method_is_not":    []string{},
		"country":          "",
		"country_is":       []string{},
		"country_is_not":   []string{},
		"ip":               "ip_is",
		"ip_is":            []string{"192.168.1.1", "192.168.1.2"},
		"ip_is_not":        []string{},
		"only_with_cookie": "",
		"action_config": map[string]interface{}{
			"fn_uuid": "function-uuid-complex",
		},
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{responseWithMethodsAndIPs})
	})
	
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, responseWithMethodsAndIPs)
	})
	
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, responseWithMethodsAndIPs)
	})
	
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/content-filter/5bf0b98f-d2f6-49dd-b5f6-5908623a9bc0", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
} 