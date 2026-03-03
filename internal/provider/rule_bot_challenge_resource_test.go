package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
	"net/http"
)

var botChallengeRuleResponse = map[string]interface{}{
	"uuid":    "444444-4444-4444-4444-444444444444",
	"rule_id": "444444-4444-4444-4444-444444444444",
	"name":    "test-bot-challenge",
	"domain":  []string{"example.com"},
	"url":     []string{"/login"},
	"action":  "bot_challenge",
	"action_config": map[string]interface{}{
		"robot_challenge_type":             "captcha",
		"robot_challenge_verification_ttl": 10800,
		"robot_challenge_challenge_ttl":    30,
	},
	"method_is":      []string{},
	"method_is_not":  []string{},
	"ip_is":          []string{},
	"ip_is_not":      []string{},
	"country_is":     []string{},
	"country_is_not": []string{},

	"disabled":         false,
	"ip":               "any",
	"method":           "any",
	"country":          "any",
	"weight":           0,
	"only_with_cookie": "",
}

func testBotChallengePreCheck(t *testing.T) {
	// You can add any additional setup here
}

func setupBotChallengeRuleServer(t *testing.T, organizationID string, projectID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, []map[string]interface{}{botChallengeRuleResponse})
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge/444444-4444-4444-4444-444444444444", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, botChallengeRuleResponse)
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, botChallengeRuleResponse)
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/rules/bot-challenge/444444-4444-4444-4444-444444444444", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, botChallengeRuleResponse)
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
