package provider_test

import (
	"fmt"
	"terraform-provider-quant/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"quant": providerserver.NewProtocol6WithError(provider.New()()),
}

func testAccPreCheck(t *testing.T) {
	// You can add any additional setup here
}

var ruleProxyResponse = map[string]interface{}{
    "uuid": "111111-1111-1111-1111-111111111111",
    "rule_id": "111111-1111-1111-1111-111111111111",
    "domain": []string{"example.com"},
	"url": []string{"/api/*"},
    "name": "test-rule",
    "action": "proxy",
	"disabled": false,
	"method": "method_is",
	"method_is": []string{"GET", "POST"},
	"country": "country_is",
	"country_is": []string{"US", "CA"},
	"ip": "ip_is",
	"ip_is": []string{"192.168.1.1"},
    "action_config": map[string]interface{}{
        "to": "https://backend.example.com",
        "host": "backend.example.com",
        "waf_enabled": true,
        "origin_timeout": "30000",
		"cache_lifetime": 3600,
        "failover_mode": false,
        "failover_origin_ttfb": "5000",
        "failover_lifetime": "300",
		"failover_origin_status_codes": []string{"500", "502", "503", "504"},
		"disable_ssl_verify": false,
		"only_proxy_404": false,
		"proxy_strip_headers": []string{"X-Custom-Header"},
		"block_ip": []string{"2.2.2.2"},
		"block_ua": []string{"bad-bot"},
		"block_referer": []string{"spam.com"},
		"request_header_name": "X-WAF-Header",
        "waf_config": map[string]interface{}{
            "mode": "report",
            "paranoia_level": 1,
			"allow_rules": []string{"rule1"},
			"allow_ip": []string{"1.1.1.1"},
			"block_ip": []string{"2.2.2.2"},
			"block_ua": []string{"bad-bot"},
			"block_referer": []string{"spam.com"},
			"notify_email": []string{"admin@example.com"},
			"notify_slack": "slack-webhook",
			"notify_slack_hits_rpm": 100,
			"thresholds": []map[string]interface{}{
                {
                    "type":     "ip",
                    "rps":      5,
                    "cooldown": 30,
                    "mode":     "disabled",
                },
                {
                    "type":     "header",
                    "rps":      5,
                    "cooldown": 30,
                    "mode":     "disabled",
                },
                {
                    "type":     "waf_hit_by_ip",
                    "hits":     10,
                    "minutes":  5,
                    "cooldown": 300,
                    "mode":     "disabled",
                },
            },
        },
		"notify": "none",
        "notify_config": map[string]interface{}{
            "period": "60",
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
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/111111-1111-1111-1111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleProxyResponse)
	})
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleProxyResponse)
	})
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/proxy/111111-1111-1111-1111-111111111111", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, ruleProxyResponse)
	})
}

func TestAccRuleProxyResourceMock(t *testing.T) {
	setupRuleProxyServer(t, "test-organization", "default")
	defer httpmock.DeactivateAndReset()

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccRuleProxyResourceConfigMock("test-rule"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-rule"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "project", "default"),
                    // Domain checks
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.#", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.0", "example.com"),
                    // URL checks
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "url.#", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "url.0", "/api/*"),
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
                    // IP checks
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "ip", "ip_is"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "ip_is.#", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "ip_is.0", "192.168.1.1"),
                    // Method checks
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "method", "method_is"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "method_is.#", "2"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "method_is.0", "GET"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "method_is.1", "POST"),
                    // WAF checks
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_enabled", "true"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.mode", "report"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.paranoia_level", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.allow_rules.#", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.allow_rules.0", "rule1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.allow_ip.#", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.allow_ip.0", "1.1.1.1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.block_ip.#", "1"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.block_ip.0", "2.2.2.2"),
                    resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.notify_slack", "slack-webhook"),
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
	domain  = ["example.com"]
	url     = ["/api/*"]
	disabled = false
	
	to              = "https://backend.example.com"
	host            = "backend.example.com"
	cache_lifetime  = 3600
	disable_ssl_verify = false
	only_proxy_404  = false
	proxy_strip_headers = ["X-Custom-Header"]
	
	country    = "country_is"
	country_is = ["US", "CA"]
	
	ip     = "ip_is"
	ip_is  = ["192.168.1.1"]
	
	method    = "method_is"
	method_is = ["GET", "POST"]
	
	waf_enabled = true
	waf_config = {
		mode           = "report"
		paranoia_level = 1
		allow_rules    = ["rule1"]
		allow_ip       = ["1.1.1.1"]
		block_ip       = ["2.2.2.2"]
		block_ua       = ["bad-bot"]
		block_referer  = ["spam.com"]
		notify_email   = ["admin@example.com"]
		notify_slack   = "slack-webhook"
		notify_slack_hits_rpm = 100
		notify_slack_rpm = 1000
	}

	failover_mode = "false"
	failover_origin_ttfb = "5000"
	failover_origin_status_codes = ["500", "502", "503", "504"]
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
