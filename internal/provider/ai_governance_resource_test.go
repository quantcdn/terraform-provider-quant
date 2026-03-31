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

func mockAiGovernanceServer(t *testing.T, org string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io"

	// Server-side state
	currentConfig := map[string]interface{}{
		"aiEnabled":   true,
		"modelPolicy": "unrestricted",
		"version":     float64(1),
	}

	governanceURL := fmt.Sprintf("%s/api/v3/organisations/%s/ai/governance", baseUrl, org)

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// GET governance config
	httpmock.RegisterResponder("GET", governanceURL,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"config": currentConfig,
			})
		})

	// PUT governance config
	httpmock.RegisterResponder("PUT", governanceURL,
		func(req *http.Request) (*http.Response, error) {
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				var requestBody map[string]interface{}
				if err := json.Unmarshal(body, &requestBody); err == nil {
					// Update server-side state from request
					if v, ok := requestBody["aiEnabled"]; ok {
						currentConfig["aiEnabled"] = v
					}
					if v, ok := requestBody["modelPolicy"]; ok {
						currentConfig["modelPolicy"] = v
					}
					if v, ok := requestBody["modelList"]; ok {
						currentConfig["modelList"] = v
					} else {
						delete(currentConfig, "modelList")
					}
					if v, ok := requestBody["mandatoryGuardrailPreset"]; ok {
						currentConfig["mandatoryGuardrailPreset"] = v
					} else {
						delete(currentConfig, "mandatoryGuardrailPreset")
					}
					if v, ok := requestBody["mandatoryFilterPolicies"]; ok {
						currentConfig["mandatoryFilterPolicies"] = v
					} else {
						delete(currentConfig, "mandatoryFilterPolicies")
					}
					if v, ok := requestBody["spendLimits"]; ok {
						currentConfig["spendLimits"] = v
					} else {
						delete(currentConfig, "spendLimits")
					}
					// Bump version
					if ver, ok := currentConfig["version"].(float64); ok {
						currentConfig["version"] = ver + 1
					} else {
						currentConfig["version"] = float64(2)
					}
				}
			}
			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"config": currentConfig,
			})
		})
}

func TestAccAiGovernanceResource(t *testing.T) {
	org := "test-org"
	mockAiGovernanceServer(t, org)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create with basic config
			{
				Config: testAccAiGovernanceConfig(org, true, "allowlist"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_governance.test", "ai_enabled", "true"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "model_policy", "allowlist"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "model_list.0", "gpt-4"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "organization", org),
					resource.TestCheckResourceAttrSet("quant_ai_governance.test", "version"),
				),
			},
			// Step 2: Update — change policy and disable AI
			{
				Config: testAccAiGovernanceConfigUpdated(org),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_governance.test", "ai_enabled", "false"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "model_policy", "blocklist"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "model_list.0", "gpt-3.5-turbo"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "spend_limits.monthly_budget_cents", "10000"),
					resource.TestCheckResourceAttr("quant_ai_governance.test", "spend_limits.warning_threshold_percent", "80"),
				),
			},
			// Step 3: Destroy — verify reset (implicitly tested by framework)
		},
	})
}

func TestAccAiGovernanceResource_Import(t *testing.T) {
	org := "test-org"
	mockAiGovernanceServer(t, org)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create first
			{
				Config: testAccAiGovernanceConfig(org, true, "allowlist"),
			},
			// Import
			{
				ResourceName:  "quant_ai_governance.test",
				ImportState:   true,
				ImportStateId: org,
				ImportStateVerifyIgnore: []string{
					"organization",
				},
			},
		},
	})
}

func testAccAiGovernanceConfig(org string, aiEnabled bool, modelPolicy string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_governance" "test" {
  organization = %[1]q
  ai_enabled   = %[2]t
  model_policy = %[3]q
  model_list   = ["gpt-4"]
}
`, org, aiEnabled, modelPolicy)
}

func testAccAiGovernanceConfigUpdated(org string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_governance" "test" {
  organization = %[1]q
  ai_enabled   = false
  model_policy = "blocklist"
  model_list   = ["gpt-3.5-turbo"]

  spend_limits = {
    monthly_budget_cents      = 10000
    warning_threshold_percent = 80
  }
}
`, org)
}
