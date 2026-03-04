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

func testBotChallengePreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupBotChallengeRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	// Dynamic closure state
	var currentName = "test-bot-challenge"
	var currentRobotChallengeType = "captcha"
	var currentVerificationTtl = 10800
	var currentChallengeTtl = 30
	var currentDisabled = false

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"uuid":    "44444444-4444-4444-a444-444444444444",
			"rule_id": "44444444-4444-4444-a444-444444444444",
			"name":    currentName,
			"domain":  []string{"example.com"},
			"url":     []string{"/login"},
			"action":  "bot_challenge",
			"action_config": map[string]interface{}{
				"robot_challenge_type":             currentRobotChallengeType,
				"robot_challenge_verification_ttl": currentVerificationTtl,
				"robot_challenge_challenge_ttl":    currentChallengeTtl,
			},
			"method_is":      []string{},
			"method_is_not":  []string{},
			"ip_is":          []string{},
			"ip_is_not":      []string{},
			"country_is":     []string{},
			"country_is_not": []string{},

			"disabled":         currentDisabled,
			"ip":               "any",
			"method":           "any",
			"country":          "any",
			"weight":           0,
			"only_with_cookie": "",
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{createResponse()})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge/44444444-4444-4444-a444-444444444444", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if rct, ok := requestBody["robot_challenge_type"].(string); ok {
					currentRobotChallengeType = rct
				}
				if vttl, ok := requestBody["robot_challenge_verification_ttl"].(float64); ok {
					currentVerificationTtl = int(vttl)
				}
				if cttl, ok := requestBody["robot_challenge_challenge_ttl"].(float64); ok {
					currentChallengeTtl = int(cttl)
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge/44444444-4444-4444-a444-444444444444", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		var requestBody map[string]interface{}
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(body, &requestBody); err == nil {
				if name, ok := requestBody["name"].(string); ok {
					currentName = name
				}
				if rct, ok := requestBody["robot_challenge_type"].(string); ok {
					currentRobotChallengeType = rct
				}
				if vttl, ok := requestBody["robot_challenge_verification_ttl"].(float64); ok {
					currentVerificationTtl = int(vttl)
				}
				if cttl, ok := requestBody["robot_challenge_challenge_ttl"].(float64); ok {
					currentChallengeTtl = int(cttl)
				}
				if disabled, ok := requestBody["disabled"].(bool); ok {
					currentDisabled = disabled
				}
			}
		}
		return httpmock.NewJsonResponse(200, createResponse())
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge/44444444-4444-4444-a444-444444444444", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewStringResponse(204, ""), nil
	})
}

func TestAccRuleBotChallengeResourceMock(t *testing.T) {
	organizationID := "test-organization"
	projectID := "default"
	setupBotChallengeRuleServer(t, organizationID, projectID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testBotChallengePreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccRuleBotChallengeResourceConfigMock(organizationID, projectID, "test-bot-challenge"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "name", "test-bot-challenge"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "project", "default"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "robot_challenge_type", "captcha"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "domain.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "domain.0", "example.com"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "url.#", "1"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "url.0", "/login"),
				),
			},
			// Update
			{
				Config: testAccRuleBotChallengeResourceConfigUpdateMock(organizationID, projectID, "test-bot-challenge-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "name", "test-bot-challenge-updated"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "robot_challenge_type", "invisible"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "robot_challenge_verification_ttl", "7200"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "robot_challenge_challenge_ttl", "60"),
					resource.TestCheckResourceAttr("quant_rule_bot_challenge.test", "disabled", "true"),
				),
			},
			// Import
			{
				ResourceName:                         "quant_rule_bot_challenge.test",
				ImportState:                          true,
				ImportStateId:                        "default/44444444-4444-4444-a444-444444444444",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
		},
	})
}

func testAccRuleBotChallengeResourceConfigMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`

provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_bot_challenge" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/login"]
	robot_challenge_type = "captcha"
}
`, organizationID, projectID, name)
}

func testAccRuleBotChallengeResourceConfigUpdateMock(organizationID string, projectID string, name string) string {
	return fmt.Sprintf(`

provider "quant" {
	bearer = "testtoken"
	organization = "%s"
}

resource "quant_rule_bot_challenge" "test" {
	project = "%s"
	name = "%s"
	domain = ["example.com"]
	url = ["/login"]
	robot_challenge_type = "invisible"
	robot_challenge_verification_ttl = 7200
	robot_challenge_challenge_ttl = 60
	disabled = true
}
`, organizationID, projectID, name)
}
