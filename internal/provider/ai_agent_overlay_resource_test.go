package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

// parseAgentOverlayJSONBody decodes the JSON body of a mocked request into dst.
// It's a tiny helper local to this test file to avoid re-reading the body as
// bytes everywhere.
func parseAgentOverlayJSONBody(req *http.Request, dst interface{}) error {
	defer func() { _ = req.Body.Close() }()
	return json.NewDecoder(req.Body).Decode(dst)
}

// mockAgentOverlayServer sets up GET/PUT/DELETE responders for a given org and
// agent, with shared mutable state so Create -> Read -> Update -> Read flows
// behave like a real server (PUT stores fields, GET returns what was last PUT,
// PUT bumps the version).
func mockAgentOverlayServer(t *testing.T, org, agentId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io"

	overlayURL := fmt.Sprintf("%s/api/v3/organizations/%s/ai/agents/%s/overlay", baseUrl, org, agentId)

	// Shared state — populated by PUT, served by GET.
	state := map[string]interface{}{}
	var version int32 = 0
	var deleted bool

	// Fixed base metadata for the agent.
	base := map[string]interface{}{
		"agentId":          agentId,
		"name":             "Code Agent",
		"modelId":          "anthropic.claude-sonnet-4-6",
		"allowedTools":     []string{},
		"assignedSkillIds": []string{},
	}

	buildGetResponse := func() map[string]interface{} {
		resp := map[string]interface{}{
			"base": base,
		}
		if deleted || len(state) == 0 {
			resp["overlay"] = nil
			return resp
		}
		overlay := map[string]interface{}{}
		for k, v := range state {
			overlay[k] = v
		}
		overlay["version"] = version
		resp["overlay"] = overlay
		return resp
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", overlayURL,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, buildGetResponse())
		})

	httpmock.RegisterResponder("PUT", overlayURL,
		func(req *http.Request) (*http.Response, error) {
			var body map[string]interface{}
			if err := parseAgentOverlayJSONBody(req, &body); err != nil {
				return httpmock.NewStringResponse(400, "bad json"), nil
			}
			// Replace all tracked overlay fields from the request body.
			// Fields absent from the body are removed so subsequent GETs match.
			newState := map[string]interface{}{}
			for _, k := range []string{
				"modelId", "temperature", "maxTokens",
				"disabledSkills", "additionalSkills",
				"additionalTools", "disabledTools",
				"systemPromptAppend", "allowedCollections",
				"guardrailPreset",
			} {
				if v, ok := body[k]; ok {
					newState[k] = v
				}
			}
			state = newState
			deleted = false
			version++

			respOverlay := map[string]interface{}{}
			for k, v := range state {
				respOverlay[k] = v
			}
			respOverlay["version"] = version
			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"overlay": respOverlay,
			})
		})

	httpmock.RegisterResponder("DELETE", overlayURL,
		func(req *http.Request) (*http.Response, error) {
			deleted = true
			state = map[string]interface{}{}
			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"deleted": true,
			})
		})
}

func TestAccAiAgentOverlayResource_basic(t *testing.T) {
	org := "test-org"
	agentId := "code"
	mockAgentOverlayServer(t, org, agentId)
	defer httpmock.DeactivateAndReset()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAiAgentOverlayBasicConfig(org),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "agent_id", "code"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "organisation", org),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "model_id", "anthropic.claude-sonnet-4-6"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "temperature", "0.5"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "max_tokens", "4096"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "system_prompt_append", "You work at Salsa Digital."),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "guardrail_preset", "official"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "version", "1"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "base.agent_id", "code"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "base.name", "Code Agent"),
				),
			},
		},
	})
}

func TestAccAiAgentOverlayResource_update(t *testing.T) {
	org := "test-org"
	agentId := "plan"
	mockAgentOverlayServer(t, org, agentId)
	defer httpmock.DeactivateAndReset()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAiAgentOverlayTunableConfig(org, agentId, "0.25", 2048),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "agent_id", agentId),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "temperature", "0.25"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "max_tokens", "2048"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "version", "1"),
				),
			},
			{
				Config: testAccAiAgentOverlayTunableConfig(org, agentId, "0.75", 8192),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "temperature", "0.75"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "max_tokens", "8192"),
					resource.TestCheckResourceAttr("quant_ai_agent_overlay.test", "version", "2"),
				),
			},
		},
	})
}

func TestAccAiAgentOverlayResource_versionConflict(t *testing.T) {
	org := "test-org"
	agentId := "review"

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseUrl := "https://dashboard.quantcdn.io"
	overlayURL := fmt.Sprintf("%s/api/v3/organizations/%s/ai/agents/%s/overlay", baseUrl, org, agentId)

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// GET returns base + null overlay (no prior overlay).
	httpmock.RegisterResponder("GET", overlayURL,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"base": map[string]interface{}{
					"agentId":          agentId,
					"name":             "Review Agent",
					"modelId":          "anthropic.claude-sonnet-4-6",
					"allowedTools":     []string{},
					"assignedSkillIds": []string{},
				},
				"overlay": nil,
			})
		})

	// PUT returns 409 to trigger the version conflict path.
	httpmock.RegisterResponder("PUT", overlayURL,
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(409, map[string]interface{}{
				"message": "version conflict",
			})
		})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAiAgentOverlayStandardConfig(org, agentId),
				ExpectError: regexp.MustCompile("Overlay version conflict"),
			},
		},
	})
}

func TestAccAiAgentOverlayResource_import(t *testing.T) {
	org := "test-org"
	agentId := "code"
	mockAgentOverlayServer(t, org, agentId)
	defer httpmock.DeactivateAndReset()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Seed mock state via apply.
			{
				Config: testAccAiAgentOverlayBasicConfig(org),
			},
			// Import and verify.
			{
				ResourceName:                         "quant_ai_agent_overlay.test",
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s/%s", org, agentId),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "agent_id",
			},
		},
	})
}

// -----------------------------------------------------------------------------
// Config builders
// -----------------------------------------------------------------------------

func testAccAiAgentOverlayBasicConfig(org string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "test"
}

resource "quant_ai_agent_overlay" "test" {
  agent_id             = "code"
  model_id             = "anthropic.claude-sonnet-4-6"
  temperature          = 0.5
  max_tokens           = 4096
  system_prompt_append = "You work at Salsa Digital."
  disabled_tools       = ["code_execution"]
  additional_skills    = ["drupal-expertise"]
  guardrail_preset     = "official"
}
`, org)
}

func testAccAiAgentOverlayTunableConfig(org, agentId, temperature string, maxTokens int) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "test"
}

resource "quant_ai_agent_overlay" "test" {
  agent_id    = %[2]q
  model_id    = "anthropic.claude-sonnet-4-6"
  temperature = %[3]s
  max_tokens  = %[4]d
}
`, org, agentId, temperature, maxTokens)
}

func testAccAiAgentOverlayStandardConfig(org, agentId string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "test"
}

resource "quant_ai_agent_overlay" "test" {
  agent_id             = %[2]q
  model_id             = "anthropic.claude-sonnet-4-6"
  temperature          = 0.5
  max_tokens           = 4096
  system_prompt_append = "You work at Salsa Digital."
  guardrail_preset     = "official"
}
`, org, agentId)
}
