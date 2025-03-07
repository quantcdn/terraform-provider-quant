package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"net/http"
)

func testAccRuleProxyPreCheck(t *testing.T) {
	// You can add any additional setup here
}

var ruleProxyResponse = map[string]interface{}{
    "uuid": "4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9",
    "rule_id": "4bf0b98f-d2f6-49dd-b5f6-5908623a9bc9",
    "domain": []string{"any"},
    "url": []string{"/proxy"},
    "name": "test-proxy",
    "action": "proxy",
    "disabled": false,
    "method": "",
    "method_is": []string{},
    "method_is_not": []string{},
    "country": "country_is",
    "country_is": []string{"US", "CA"},
    "country_is_not": []string{},
    "ip": "",
    "ip_is": []string{},
    "ip_is_not": []string{},
    "only_with_cookie": "",
    "action_config": map[string]interface{}{
        "to": "https://backend.example.com",
        "host": "backend.example.com",
        "waf_enabled": true,
        "origin_timeout": "30000",
        "cache_lifetime": 3600,
        "failover_mode": false,
        "failover_origin_ttfb": "5000",
        "failover_lifetime": "300",
        "failover_origin_status_codes": []string{},
        "disable_ssl_verify": false,
        "only_proxy_404": false,
        "proxy_strip_headers": []string{"X-Custom-Header"},
        "proxy_alert_enabled": true,
        "proxy_inline_fn_enabled": false,
        "auth_user": "",
        "auth_pass": "",
        "inject_headers": nil,
        "waf_config": map[string]interface{}{
            "mode": "report",
            "paranoia_level": 1,
            "allow_rules": []string{},
            "allow_ip": []string{},
            "block_ip": []string{},
            "block_ua": []string{},
            "block_referer": []string{},
            "notify_email": []string{},
            "notify_slack": "",
            "notify_slack_hits_rpm": nil,
            "block_lists": map[string]interface{}{
                "referer": false,
                "user_agent": false,
                "ai": false,
                "ip": false,
            },
            "httpbl": map[string]interface{}{
                "httpbl_enabled": false,
                "block_suspicious": false,
                "block_harvester": false,
                "api_key": "",
                "block_search_engine": false,
                "block_spam": false,
            },
            "thresholds": []map[string]interface{}{
                {
                    "type":     "ip",
                    "rps":      5,
                    "cooldown": 30,
                    "mode":     "disabled",
                    "notify_slack": "",
                },
                {
                    "type":     "header",
                    "rps":      5,
                    "cooldown": 30,
                    "mode":     "disabled",
                    "value":    "",
                    "notify_slack": "",
                },
                {
                    "type":     "waf_hit_by_ip",
                    "hits":     10,
                    "minutes":  5,
                    "cooldown": 300,
                    "mode":     "disabled",
                    "notify_slack": "",
                },
            },
        },
        "notify": "none",
        "notify_config": map[string]interface{}{
            "period": "60",
            "slack_webhook": "",
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
