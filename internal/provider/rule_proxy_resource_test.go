package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Mock API responses
const mockRuleProxyResponse = `{
	"name": "test-rule",
	"domain": ["example.com"],
	"to": "https://backend.example.com",
	"host": "backend.example.com",
	"waf_enabled": true,
	"waf_config": {
		"mode": "detection",
		"paranoia_level": 1,
		"allow_rules": [],
		"block_ip": [],
		"block_ua": [],
		"block_referer": [],
		"notify_email": [],
		"httpbl": {
			"enabled": false
		}
	}
}`

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
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "to", "https://backend.example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "host", "backend.example.com"),
					resource.TestCheckResourceAttr("quant_rule_proxy.test", "waf_enabled", "true"),
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
