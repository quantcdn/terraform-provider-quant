package provider_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

// setupRuleErrorResponder configures httpmock to return a specific error
// response for a POST to the given rule type endpoint.
func setupRuleErrorResponder(t *testing.T, org, project, ruleType string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/%s", baseUrl, org, project, ruleType),
		func(req *http.Request) (*http.Response, error) {
			t.Logf("Rule %s create request received — returning %d", ruleType, statusCode)
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

// ---------- Auth rule error tests ----------

func TestRuleAuth_CreateError401(t *testing.T) {
	setupRuleErrorResponder(t, "test-org", "test-project", "auth", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "quant" {
	bearer       = "bad-token"
	organization = "test-org"
}

resource "quant_rule_auth" "test" {
	project   = "test-project"
	name      = "error-test"
	domain    = ["any"]
	url       = ["/*"]
	auth_user = "admin"
	auth_pass = "secret"
}
`,
				ExpectError: regexp.MustCompile(`Failed to create rule`),
			},
		},
	})
}

// ---------- Proxy rule error tests ----------

func TestRuleProxy_CreateError400(t *testing.T) {
	setupRuleErrorResponder(t, "test-org", "test-project", "proxy", 400, map[string]interface{}{
		"error":   true,
		"message": "Invalid proxy configuration",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "quant" {
	bearer       = "testtoken"
	organization = "test-org"
}

resource "quant_rule_proxy" "test" {
	project  = "test-project"
	name     = "error-test"
	domain   = ["any"]
	url      = ["/*"]
	to       = "https://backend.example.com"
	host     = "backend.example.com"
}
`,
				ExpectError: regexp.MustCompile(`Failed to create rule`),
			},
		},
	})
}

// ---------- Content filter rule error tests ----------

func TestRuleContentFilter_CreateError500(t *testing.T) {
	setupRuleErrorResponder(t, "test-org", "test-project", "content-filter", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "quant" {
	bearer       = "testtoken"
	organization = "test-org"
}

resource "quant_rule_content_filter" "test" {
	project  = "test-project"
	name     = "error-test"
	domain   = ["any"]
	url      = ["/*"]
	fn_uuid  = "function-uuid-12345"
}
`,
				ExpectError: regexp.MustCompile(`Failed to create rule`),
			},
		},
	})
}
