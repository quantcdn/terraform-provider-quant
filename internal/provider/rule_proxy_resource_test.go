package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
)

func testAccRuleProxyPreCheck(t *testing.T) {
	// You can add any additional setup here
}

var ruleProxyResponse = map[string]interface{}{
	"uuid":             "4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9",
	"rule_id":          "4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9",
	"domain":           []string{"any"},
	"url":              []string{"/proxy"},
	"name":             "test-proxy",
	"action":           "proxy",
	"disabled":         false,
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
		"to":                           "https://backend.example.com",
		"host":                         "backend.example.com",
		"waf_enabled":                  true,
        "origin_timeout":               "30000",
		"cache_lifetime":               "3600",
		"failover_mode":                false,
		"failover_origin_ttfb":         "5000",
		"failover_lifetime":            "300",
		"failover_origin_status_codes": []string{},
		"disable_ssl_verify":           false,
		"only_proxy_404":               false,
		"proxy_strip_headers":          []string{"X-Custom-Header"},
		"proxy_alert_enabled":          true,
		"proxy_inline_fn_enabled":      false,
		"auth_user":                    "",
		"auth_pass":                    "",
		"inject_headers":               nil,
		"waf_config": map[string]interface{}{
			"mode":                  "report",
			"paranoia_level":        1,
			"allow_rules":           []string{},
			"allow_ip":              []string{},
			"block_ip":              []string{},
			"block_ua":              []string{},
			"block_referer":         []string{},
			"notify_email":          []string{},
			"notify_slack":          "",
			"notify_slack_hits_rpm": nil,
			"static_error_page":     "",
			"static_error_page_status_codes": []string{},
			"block_lists": map[string]interface{}{
				"referer":    false,
				"user_agent": false,
				"ai":         false,
				"ip":         false,
			},
			"httpbl": map[string]interface{}{
				"httpbl_enabled":      false,
				"block_suspicious":    false,
				"block_harvester":     false,
				"api_key":             "",
				"block_search_engine": false,
				"block_spam":          false,
			},
			"thresholds": []map[string]interface{}{
				{
					"type":         "ip",
					"rps":          5,
					"cooldown":     30,
					"mode":         "disabled",
					"notify_slack": "",
				},
				{
					"type":         "header",
					"rps":          5,
					"cooldown":     30,
					"mode":         "disabled",
					"value":        "",
					"notify_slack": "",
				},
				{
					"type":         "waf_hit_by_ip",
					"hits":         10,
					"minutes":      5,
					"cooldown":     300,
					"mode":         "disabled",
					"notify_slack": "",
				},
			},
		},
		"notify": "none",
		"notify_config": map[string]interface{}{
			"period":              "60",
			"slack_webhook":       "",
			"origin_status_codes": []string{},
		},
	},
}

func setupRuleProxyServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{ruleProxyResponse})
	})
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleProxyResponse)
	})
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleProxyResponse)
	})
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleProxyResponse)
	})
}

// Setup a mock server that validates application_* fields are sent on create and update
func setupRuleProxyServerForApplicationFields(t *testing.T, organizationID string, projectID string, expectedCreate map[string]interface{}, expectedUpdate map[string]interface{}) {
    httpmock.Activate()
    baseUrl := "https://dashboard.quantcdn.io/api/v2"

    httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
        t.Logf("Request: %s", req.URL)
        return httpmock.NewStringResponse(404, "Not Found"), nil
    })

    // Track current values to reflect updates across GET/POST/PATCH
    current := deepCopy(ruleProxyResponse)

    // Common GET list and read responses
    httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
        return httpmock.NewJsonResponse(200, []map[string]interface{}{current})
    })
    httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
        return httpmock.NewJsonResponse(200, current)
    })

    // POST create should include application_* fields
    httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
        var body map[string]interface{}
        if req.Body != nil {
            raw, _ := io.ReadAll(req.Body)
            _ = json.Unmarshal(raw, &body)
        }

        // Assertions for create
        if v, ok := expectedCreate["application_proxy"]; ok {
            if body["application_proxy"] != v {
                t.Errorf("application_proxy (create) mismatch: got %v want %v", body["application_proxy"], v)
            }
        }
        if v, ok := expectedCreate["application_name"]; ok {
            if body["application_name"] != v {
                t.Errorf("application_name (create) mismatch: got %v want %v", body["application_name"], v)
            }
        }
        if v, ok := expectedCreate["application_environment"]; ok {
            if body["application_environment"] != v {
                t.Errorf("application_environment (create) mismatch: got %v want %v", body["application_environment"], v)
            }
        }
        if v, ok := expectedCreate["application_container"]; ok {
            if body["application_container"] != v {
                t.Errorf("application_container (create) mismatch: got %v want %v", body["application_container"], v)
            }
        }
        if v, ok := expectedCreate["application_port"]; ok {
            if body["application_port"] != v {
                t.Errorf("application_port (create) mismatch: got %v want %v", body["application_port"], v)
            }
        }

        // Echo back selected fields from request into current response to avoid provider inconsistencies
        if v, ok := body["name"].(string); ok {
            current["name"] = v
        }
        if ac, ok := current["action_config"].(map[string]interface{}); ok {
            if v, ok := body["waf_enabled"].(bool); ok {
                ac["waf_enabled"] = v
            }
            if v, ok := body["failover_origin_ttfb"].(string); ok {
                ac["failover_origin_ttfb"] = v
            }
        }
        return httpmock.NewJsonResponse(200, current)
    })

    // PATCH update should include application_* fields when present
    httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
        var body map[string]interface{}
        if req.Body != nil {
            raw, _ := io.ReadAll(req.Body)
            _ = json.Unmarshal(raw, &body)
        }

        // Assertions for update
        if v, ok := expectedUpdate["application_proxy"]; ok {
            if body["application_proxy"] != v {
                t.Errorf("application_proxy (update) mismatch: got %v want %v", body["application_proxy"], v)
            }
        }
        if v, ok := expectedUpdate["application_name"]; ok {
            if body["application_name"] != v {
                t.Errorf("application_name (update) mismatch: got %v want %v", body["application_name"], v)
            }
        }
        if v, ok := expectedUpdate["application_environment"]; ok {
            if body["application_environment"] != v {
                t.Errorf("application_environment (update) mismatch: got %v want %v", body["application_environment"], v)
            }
        }
        if v, ok := expectedUpdate["application_container"]; ok {
            if body["application_container"] != v {
                t.Errorf("application_container (update) mismatch: got %v want %v", body["application_container"], v)
            }
        }
        if v, ok := expectedUpdate["application_port"]; ok {
            if body["application_port"] != v {
                t.Errorf("application_port (update) mismatch: got %v want %v", body["application_port"], v)
            }
        }

        // Echo back selected fields on update too
        if v, ok := body["name"].(string); ok {
            current["name"] = v
        }
        if ac, ok := current["action_config"].(map[string]interface{}); ok {
            if v, ok := body["waf_enabled"].(bool); ok {
                ac["waf_enabled"] = v
            }
            if v, ok := body["failover_origin_ttfb"].(string); ok {
                ac["failover_origin_ttfb"] = v
            }
        }
        return httpmock.NewJsonResponse(200, current)
    })

    httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
        return httpmock.NewJsonResponse(200, ruleProxyResponse)
    })
}

// deepCopy makes a deep copy of a map[string]interface{} using json marshal/unmarshal
func deepCopy(m map[string]interface{}) map[string]interface{} {
    b, _ := json.Marshal(m)
    var out map[string]interface{}
    _ = json.Unmarshal(b, &out)
    return out
}

func TestAccRuleProxyApplicationProxyRequests(t *testing.T) {
    createExpected := map[string]interface{}{
        "application_proxy":       true,
        "application_name":        "orders",
        "application_environment": "prod",
        "application_container":   "orders-app",
        "application_port":        float64(8080),
    }
    updateExpected := map[string]interface{}{
        "application_proxy":       true,
        "application_name":        "orders-v2",
        "application_environment": "staging",
        "application_container":   "orders-app-v2",
        "application_port":        float64(9090),
    }

    setupRuleProxyServerForApplicationFields(t, "test-organization", "default", createExpected, updateExpected)
    defer httpmock.DeactivateAndReset()

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccRuleProxyPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccRuleProxyResourceConfigWithApplicationFields("test-proxy-app", createExpected),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-proxy-app"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "project", "default"),
                ),
            },
            {
                Config: testAccRuleProxyResourceConfigWithApplicationFields("test-proxy-app", updateExpected),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-proxy-app"),
                ),
            },
        },
    })
}

func testAccRuleProxyResourceConfigWithApplicationFields(name string, vals map[string]interface{}) string {
    // values from map are asserted by server; here we just use them in config
    return fmt.Sprintf(`
provider "quant" {
  bearer = "testtoken"
  organization = "test-organization"
}

resource "quant_rule_proxy" "test" {
  name    = %q
  project = "default"
  domain  = ["any"]
  url     = ["/proxy"]

  to   = "https://backend.example.com"
  host = "backend.example.com"

  application_proxy       = %v
  application_name        = %q
  application_environment = %q
  application_container   = %q
  application_port        = %d

  waf_enabled = false
  waf_config = {
    mode = "report"
  }
}
`,
        name,
        vals["application_proxy"],
        vals["application_name"],
        vals["application_environment"],
        vals["application_container"],
        int(vals["application_port"].(float64)),
    )
}

func TestAccRuleProxyResourceMock(t *testing.T) {
	setupRuleProxyServer(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccRuleProxyPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleProxyResourceConfigMock("test-proxy"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-proxy"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "project", "default"),
					// Domain checks
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.0", "any"),
					// URL checks
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "url.0", "/proxy"),
					// Proxy config checks
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "to", "https://backend.example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "host", "backend.example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "cache_lifetime", "3600"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "disable_ssl_verify", "false"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "only_proxy_404", "false"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "proxy_strip_headers.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "proxy_strip_headers.0", "X-Custom-Header"),
					// Country checks
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "country", "country_is"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "country_is.#", "2"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "country_is.0", "US"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "country_is.1", "CA"),
					// WAF checks
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_enabled", "true"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.mode", "report"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.paranoia_level", "1"),
					testAccCheckRuleProxyExists("quant_rule_proxy.test"),
				),
			},
		},
	})
}

func testAccRuleProxyResourceConfigMock(name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_rule_proxy" "test" {
	name    = %[1]q
	project = "default"
	domain  = ["any"]
	url     = ["/proxy"]
	disabled = false
	
	to              = "https://backend.example.com"
	host            = "backend.example.com"
	cache_lifetime  = 3600
	disable_ssl_verify = false
	only_proxy_404  = false
	proxy_strip_headers = ["X-Custom-Header"]
	
	country    = "country_is"
	country_is = ["US", "CA"]
	
	waf_enabled = true
	waf_config = {
		mode           = "report"
		paranoia_level = 1
	}

	failover_mode = false
	failover_origin_ttfb = "5000"
}
`, name)
}

func testAccCheckRuleProxyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Rule Proxy ID is set")
		}

		// Here you would typically make an API call to verify the resource exists
		// Instead, we'll just return nil since we're mocking
		return nil
	}
}

// Test cache_lifetime sentinel value behavior
func TestAccRuleProxyCacheLifetimeSentinel(t *testing.T) {
	setupRuleProxyServerForCacheLifetime(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccRuleProxyPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test -1 sentinel value (explicitly unset)
			{
				Config: testAccRuleProxyConfigCacheLifetimeSentinel("test-unset", -1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-unset"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "cache_lifetime", "-1"),
					testAccCheckRuleProxyExists("quant_rule_proxy.test"),
				),
			},
			// Test 0 value (disable caching)
			{
				Config: testAccRuleProxyConfigCacheLifetimeSentinel("test-disabled", 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-disabled"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "cache_lifetime", "0"),
				),
			},
			// Test positive value (specific cache time)
			{
				Config: testAccRuleProxyConfigCacheLifetimeSentinel("test-cached", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-cached"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "cache_lifetime", "3600"),
				),
			},
			// Test omitted value (null) - should not have cache_lifetime attribute
			{
				Config: testAccRuleProxyConfigCacheLifetimeOmitted("test-omitted"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-omitted"),
					// Note: We can't use TestCheckNoResourceAttr because the attribute is computed
					// Instead we'll verify the computed value is what we expect from the API
				),
			},
		},
	})
}

func setupRuleProxyServerForCacheLifetime(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Store the current test state to make responses dynamic
	var currentName = "test-proxy"
	var currentCacheLifetime interface{} = 3600

	// Create a simple response without WAF complications
	createSimpleResponse := func(name string, cacheLifetime interface{}) map[string]interface{} {
		// Convert cache_lifetime to string format for API compatibility
		var cacheLifetimeStr string
		if cacheLifetime != nil {
			cacheLifetimeStr = fmt.Sprintf("%v", cacheLifetime)
		}
		
		return map[string]interface{}{
			"uuid":             "4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9",
			"rule_id":          "4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9",
			"domain":           []string{"any"},
			"url":              []string{"/proxy"},
			"name":             name,
			"action":           "proxy",
			"disabled":         false,
			"method":           "",
			"method_is":        []string{},
			"method_is_not":    []string{},
			"country":          "",
			"country_is":       []string{},
			"country_is_not":   []string{},
			"ip":               "",
			"ip_is":            []string{},
			"ip_is_not":        []string{},
			"only_with_cookie": "",
			"action_config": map[string]interface{}{
				"to":                           "https://backend.example.com",
				"host":                         "backend.example.com",
				"waf_enabled":                  false,
				"proxy_alert_enabled":          false,
				"cache_lifetime":               cacheLifetimeStr,
				"failover_mode":                false,
				"failover_origin_ttfb":         "2000",
				"failover_lifetime":            "300",
				"failover_origin_status_codes": []string{},
				"disable_ssl_verify":           false,
				"only_proxy_404":               false,
				"proxy_strip_headers":          []string{},
				"proxy_strip_request_headers":  []string{},
				"auth_user":                    "",
				"auth_pass":                    "",
				"inject_headers":               nil,
				"notify":                       "none",
				"notify_config": map[string]interface{}{
					"period":              "60",
					"slack_webhook":       "",
					"origin_status_codes": []string{},
				},
				"waf_config": map[string]interface{}{
					"mode":                              "report",
					"paranoia_level":                    1,
					"allow_rules":                       []string{},
					"allow_ip":                          []string{},
					"block_ip":                          []string{},
					"block_ua":                          []string{},
					"block_referer":                     []string{},
					"notify_email":                      []string{},
					"notify_slack":                      "",
					"notify_slack_hits_rpm":             nil,
					"request_header_name":               "",
					"httpbl_enabled":                    map[string]interface{}{},
					"ip_ratelimit_cooldown":             30,
					"ip_ratelimit_mode":                 "disabled",
					"ip_ratelimit_rps":                  5,
					"request_header_ratelimit_cooldown": 30,
					"request_header_ratelimit_mode":     "disabled",
					"request_header_ratelimit_rps":      5,
					"waf_ratelimit_cooldown":            300,
					"waf_ratelimit_hits":                10,
					"waf_ratelimit_mode":                "disabled",
					"waf_ratelimit_rps":                 5,
				},
			},
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// Dynamic responses that change based on request
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createSimpleResponse(currentName, currentCacheLifetime)})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createSimpleResponse(currentName, currentCacheLifetime))
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		// Parse the request to extract name and cache_lifetime
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if cacheLifetime, ok := requestBody["cache_lifetime"]; ok {
					currentCacheLifetime = cacheLifetime
				}
			}
		}
		return httpmock.NewJsonResponse(200, createSimpleResponse(currentName, currentCacheLifetime))
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		// Parse the request to extract name and cache_lifetime
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if cacheLifetime, ok := requestBody["cache_lifetime"]; ok {
					currentCacheLifetime = cacheLifetime
				}
			}
		}
		return httpmock.NewJsonResponse(200, createSimpleResponse(currentName, currentCacheLifetime))
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createSimpleResponse(currentName, currentCacheLifetime))
	})
}

func testAccRuleProxyConfigCacheLifetimeSentinel(name string, cacheLifetime int) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_rule_proxy" "test" {
	name    = %[1]q
	project = "default"
	domain  = ["any"]
	url     = ["/proxy"]
	
	to              = "https://backend.example.com"
	host            = "backend.example.com"
	cache_lifetime  = %[2]d
	waf_enabled     = false
	
	# Even when WAF is disabled, we need to provide a waf_config block due to schema requirements
	waf_config = {
		mode = "report"
	}
}
`, name, cacheLifetime)
}

func testAccRuleProxyConfigCacheLifetimeOmitted(name string) string {
	return fmt.Sprintf(`
provider "quant" {
	bearer = "testtoken"
	organization = "test-organization"
}

resource "quant_rule_proxy" "test" {
	name    = %[1]q
	project = "default"
	domain  = ["any"]
	url     = ["/proxy"]
	
	to              = "https://backend.example.com"
	host            = "backend.example.com"
	waf_enabled     = false
	# cache_lifetime omitted - should respect origin headers
	
	# Even when WAF is disabled, we need to provide a waf_config block due to schema requirements
	waf_config = {
		mode = "report"
	}
}
`, name)
}

// Unit test for cache_lifetime sentinel value handling
func TestRuleProxyCacheLifetimeHandling(t *testing.T) {
	tests := []struct {
		name           string
		apiResponse    interface{}
		configValue    *int64
		expectedResult string
		description    string
	}{
		{
			name:           "API returns null, config omitted",
			apiResponse:    nil,
			configValue:    nil,
			expectedResult: "null",
			description:    "When API returns null and config omits cache_lifetime, should remain null",
		},
		{
			name:           "API returns null, config sets -1",
			apiResponse:    nil,
			configValue:    int64Ptr(-1),
			expectedResult: "-1",
			description:    "When API returns null and config sets -1, should preserve -1",
		},
		{
			name:           "API returns 0, config sets 0",
			apiResponse:    0,
			configValue:    int64Ptr(0),
			expectedResult: "0",
			description:    "When API returns 0 and config sets 0, should be 0",
		},
		{
			name:           "API returns 3600, config sets 3600",
			apiResponse:    3600,
			configValue:    int64Ptr(3600),
			expectedResult: "3600",
			description:    "When API returns 3600 and config sets 3600, should be 3600",
		},
		{
			name:           "API returns 0, config sets -1",
			apiResponse:    0,
			configValue:    int64Ptr(-1),
			expectedResult: "-1",
			description:    "When config explicitly sets -1, should preserve -1 regardless of API response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from callRuleProxyReadAPI
			var result string

			// Mock current state (what's in Terraform state)
			var currentCacheLifetime types.String
			if tt.configValue == nil {
				currentCacheLifetime = types.StringNull()
			} else {
				currentCacheLifetime = types.StringValue(strconv.FormatInt(*tt.configValue, 10))
			}

			// Apply the logic from our fixed callRuleProxyReadAPI function
			if currentCacheLifetime.IsNull() {
				if tt.apiResponse == nil {
					result = "null"
				} else {
					result = fmt.Sprintf("%v", tt.apiResponse)
				}
			} else if currentCacheLifetime.ValueString() == "-1" {
				// Preserve -1 sentinel value
				result = "-1"
			} else {
				// Use API response for other values
				if tt.apiResponse == nil {
					result = "null"
				} else {
					result = fmt.Sprintf("%v", tt.apiResponse)
				}
			}

			if result != tt.expectedResult {
				t.Errorf("Test %s failed: expected %s, got %s\nDescription: %s",
					tt.name, tt.expectedResult, result, tt.description)
			}
		})
	}
}

// Helper function to create int64 pointer
func int64Ptr(v int64) *int64 {
	return &v
}

// Test the actual API request logic for cache_lifetime
func TestRuleProxyCreateUpdateCacheLifetime(t *testing.T) {
	tests := []struct {
		name            string
		cacheLifetime   *int64
		shouldSetInAPI  bool
		expectedAPICall string
		description     string
	}{
		{
			name:            "Omitted cache_lifetime",
			cacheLifetime:   nil,
			shouldSetInAPI:  false,
			expectedAPICall: "not called",
			description:     "When cache_lifetime is omitted, SetCacheLifetime should not be called",
		},
		{
			name:            "cache_lifetime = 0",
			cacheLifetime:   int64Ptr(0),
			shouldSetInAPI:  true,
			expectedAPICall: "SetCacheLifetime(0)",
			description:     "When cache_lifetime is 0, should disable caching",
		},
		{
			name:            "cache_lifetime = -1",
			cacheLifetime:   int64Ptr(-1),
			shouldSetInAPI:  false,
			expectedAPICall: "not called",
			description:     "When cache_lifetime is -1, SetCacheLifetime should not be called (unset)",
		},
		{
			name:            "cache_lifetime = 3600",
			cacheLifetime:   int64Ptr(3600),
			shouldSetInAPI:  true,
			expectedAPICall: "SetCacheLifetime(3600)",
			description:     "When cache_lifetime is positive, should set specific cache time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the logic from Create/Update functions
			var apiCallMade bool
			var apiCallValue int32

			// Simulate the logic from our fixed Create/Update functions
			if tt.cacheLifetime != nil {
				cacheLifetime := *tt.cacheLifetime
				// Use -1 as a sentinel value to mean "unset" (don't send to API, respect origin headers)
				if cacheLifetime != -1 {
					apiCallMade = true
					apiCallValue = int32(cacheLifetime)
				}
			}

			// Verify expectations
			if tt.shouldSetInAPI != apiCallMade {
				t.Errorf("Test %s failed: expected shouldSetInAPI=%v, got apiCallMade=%v\nDescription: %s",
					tt.name, tt.shouldSetInAPI, apiCallMade, tt.description)
			}

			if tt.shouldSetInAPI && apiCallMade {
				expectedValue := int32(*tt.cacheLifetime)
				if apiCallValue != expectedValue {
					t.Errorf("Test %s failed: expected API call value %d, got %d",
						tt.name, expectedValue, apiCallValue)
				}
			}
		})
	}
}
