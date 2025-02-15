package provider_test

import (
	"fmt"
	"terraform-provider-quant/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Mock API responses
const mockRuleProxyResponse = `{
    "uuid": "test-uuid",
    "rule_id": "test-rule-id",
    "name": "test-rule",
    "domain": ["example.com"],
    "url": ["/api/*"],
    "action": "proxy",
    "action_config": {
        "proxy": {
            "to": "https://backend.example.com",
            "host": "backend.example.com",
            "cache_lifetime": 3600,
            "disable_ssl_verify": false,
            "only_proxy_404": false,
            "proxy_strip_headers": ["X-Custom-Header"]
        },
        "failover": {
            "failover_mode": "false",
            "failover_origin_ttfb": "5s",
            "failover_origin_status_codes": ["500", "502", "503", "504"]
        },
        "waf_config": {
            "mode": "report",
            "paranoia_level": 1,
            "allow_rules": ["rule1"],
            "allow_ip": ["1.1.1.1"],
            "block_ip": ["2.2.2.2"],
            "block_ua": ["bad-bot"],
            "block_referer": ["spam.com"],
            "notify_email": ["admin@example.com"],
            "notify_slack": "slack-webhook",
            "notify_slack_hits_rpm": 100,
            "notify_slack_rpm": 1000,
            "request_header_name": "X-WAF-Header"
        }
    },
    "country": "country_is",
    "country_is": ["US", "CA"],
    "ip": "ip_is",
    "ip_is": ["192.168.1.1"],
    "method": "method_is",
    "method_is": ["GET", "POST"],
    "waf_enabled": true
}`

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"quant": providerserver.NewProtocol6WithError(provider.New()()),
}

func testAccPreCheck(t *testing.T) {
	// You can add any additional setup here
}

func TestAccRuleProxyResourceMock(t *testing.T) {
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
resource "quant_rule_proxy" "test" {
    name    = %[1]q
    project = "default"
    domain  = ["example.com"]
    url     = ["/api/*"]
    
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
    waf_config {
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
        request_header_name = "X-WAF-Header"
    }

    failover {
        failover_mode = "false"
        failover_origin_ttfb = "5s"
        failover_origin_status_codes = ["500", "502", "503", "504"]
    }
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
