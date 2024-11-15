package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPreCheck(t *testing.T) {
	// Add any pre-check logic here if needed
	// For now, we'll leave it empty as a placeholder
}

func TestAccRuleProxyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRuleProxyResourceConfig("test-rule"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-rule"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "to", "https://backend.example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "host", "backend.example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_enabled", "true"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.mode", "detection"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_config.paranoia_level", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "quant_rule_proxy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccRuleProxyResourceConfig("test-rule-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "name", "test-rule-updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRuleProxyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "quant_rule_proxy" "test" {
  name    = %[1]q
  project = "default"
  domain  = ["example.com"]
  to      = "https://backend.example.com"
  host    = "backend.example.com"
  
  waf_enabled = true
  waf_config {
    mode           = "detection"
    paranoia_level = 1
    allow_rules    = []
    block_ip       = []
    block_ua       = []
    block_referer  = []
    notify_email   = []
    httpbl {
      enabled = false
    }
  }
}
`, name)
}
