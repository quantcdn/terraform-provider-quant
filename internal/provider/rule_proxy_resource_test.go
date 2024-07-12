package provider_test

import (
	"encoding/json"
	"testing"

	openapi "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

// Static schema from an API request
// curl --request GET \
//  --url http://localhost:8001/api/v2/organizations/quant/projects/api-test/rules/proxy \
//  --header 'Authorization: Bearer $QUANT_BEARER' \
//  --header 'User-Agent: insomnia/8.3.0'
func TestProxySchema(t *testing.T) {
	jsonData := []byte(`[
		{
			"country_is_not": [],
			"country": "country_is",
			"ip_is_not": [],
			"action_config": {
				"disable_ssl_verify": "true",
				"proxy_strip_headers": [],
				"auth_pass": "",
				"origin_timeout": "15000",
				"waf_config": {
					"notify_email": [],
					"block_lists": {
						"user_agent": false,
						"ip": false,
						"referer": false
					},
					"thresholds": [
						{
							"cooldown": 30,
							"type": "ip",
							"rps": 10,
							"notify_slack": "",
							"mode": "disabled"
						},
						{
							"value": "",
							"cooldown": 30,
							"mode": "disabled",
							"rps": 5,
							"notify_slack": "",
							"type": "header"
						},
						{
							"minutes": 5,
							"cooldown": 300,
							"hits": 10,
							"type": "waf_hit_by_ip",
							"notify_slack": "",
							"mode": "disabled"
						}
					],
					"notify_slack_hits_rpm": "",
					"mode": "report",
					"allow_rules": [],
					"httpbl": {
						"block_suspicious": false,
						"block_harvester": false,
						"block_spam": false,
						"block_search_engine": false,
						"httpbl_enabled": false
					},
					"block_ua": [],
					"block_referer": [],
					"block_ip": [],
					"paranoia_level": 1,
					"notify_slack": "",
					"allow_ip": []
				},
				"proxy_alert_enabled": false,
				"waf_enabled": true,
				"cache_lifetime": "",
				"failover_mode": false,
				"failover_lifetime": "",
				"proxy_strip_request_headers": [],
				"failover_origin_ttfb": "",
				"only_proxy_404": false,
				"auth_user": "",
				"inject_headers": null,
				"notify_config": {
					"slack_webhook": "",
					"period": "60",
					"origin_status_codes": [
						"200",
						"404",
						"301",
						"302",
						"304"
					]
				},
				"failover_origin_status_codes": [
					"200",
					"404",
					"301",
					"302",
					"304"
				],
				"to": "https:\/\/sdp3.amazee.io$1",
				"host": "content.premier.vic.gov.au",
				"notify": "none"
			},
			"name": "[drupal] authenticated traffic (WAF to report mode in AU)",
			"domain": "content.premier.vic.gov.au",
			"disabled": false,
			"url": [
				"*"
			],
			"uuid": "2a254c9b-b275-4755-9dc5-30fca699032a",
			"country_is": [
				"AU"
			],
			"method": "any",
			"method_is": [],
			"method_is_not": [],
			"ip_is": [],
			"action": "proxy",
			"only_with_cookie": "SSESS*",
			"ip": "any"
}
]`)

	var rules []openapi.RuleProxy
	err := json.Unmarshal(jsonData, &rules)

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	// Simple assertion, this test aims to ensure that the expected result
	// from the API can be unmarhslled correctlly into openapi.RuleProxy.
	assert.Equal(t, 1, len(rules))
}

// func TestListRules(t *testing.T) {
// 	bearer := os.Getenv("QUANT_BEARER")
// 	cfg := openapi.NewConfiguration()
// 	client := openapi.NewAPIClient(cfg)
// 	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

// 	rules, _, err := client.RulesProxyAPI.RulesProxyList(ctx, "quant", "tf-test-6").Execute()

// 	if err != nil {
// 		t.Fatalf("Error: %v", err)
// 	}

// 	// This is a live API call to the Quant API, it will return the rules for `tf-test-6`
// 	// the rule names may change, but this is to validate that a live API call returns
// 	// values as we're expecting and the SDK is able to unmarshal the response.
// 	for _, rule := range rules {
// 		assert.Equal(t, "proxy", rule.GetAction())
// 	}
// }

// func TestRuleAfterCreate(t *testing.T) {
// 	jsonData := []byte(`[{"uuid":"baf1c2a5-3274-4746-be7b-80487ba62906","domain":"any","country":"country_is","country_is":["AU"],"country_is_not":[],"method":"method_is","method_is":["GET"],"method_is_not":[],"ip":"any","ip_is":[],"ip_is_not":[],"only_with_cookie":"","url":["\/*"],"name":"SDK-TF proxy rule 1719533575","disabled":false,"action":"proxy","action_config":{"to":"http:\/\/origin.com","host":"test.com","auth_user":"test","auth_pass":"test","disable_ssl_verify":true,"cache_lifetime":100,"only_proxy_404":false,"proxy_strip_headers":["x-strip-me"],"waf_enabled":true,"proxy_alert_enabled":true,"origin_timeout":"30000","failover_mode":false,"failover_origin_ttfb":"2000","failover_lifetime":300,"notify":"none","notify_config":{"period":"120","slack_webhook":"https:\/\/slack.com","origin_status_codes":[]},"inject_headers":null,"failover_origin_status_codes":null}}]`)
// 	var rules []openapi.RuleProxy
// 	err := json.Unmarshal(jsonData, &rules)

// 	if err != nil {
// 		t.Fatalf("Error: %v", err)
// 	}

// 	assert.Equal(t, "SDK-TF proxy rule 1719533575", rules[0].GetName())
// }

// func TestCreateProxyRule(t *testing.T) {
// 	bearer := os.Getenv("QUANT_BEARER")
// 	project := "api-test"
// 	ts := time.Now().Unix()

// 	cfg := openapi.NewConfiguration()
// 	client := openapi.NewAPIClient(cfg)
// 	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, bearer)

// 	req := *openapi.NewRuleProxyRequestWithDefaults()

// 	req.SetName(fmt.Sprintf("SDK-TF proxy rule %v", ts))
// 	req.SetDomain("any")

// 	urls := []string{"/*"}
// 	req.SetUrl(urls)

// 	req.SetCountry("country_is")
// 	req.SetCountryIs([]string{"AU"})

// 	// req.SetIp()
// 	// req.SetIpIs()
// 	// req.SetIpIsNot()

// 	req.SetMethod("method_is")
// 	req.SetMethodIs([]string{"GET"})

// 	req.SetTo("http://origin.com")
// 	req.SetHost("test.com")
// 	// req.SetHost() # Override the host header.
// 	req.SetCacheLifetime("100")

// 	req.SetAuthPass("test")
// 	req.SetAuthUser("test")

// 	req.SetDisableSslVerify(true)
// 	req.SetOnlyProxy404(false)
// 	req.SetFailoverMode("false")

// 	// Wrong type; needs to be { key : value }
// 	req.SetProxyStripHeaders([]string{"x-strip-me"})
// 	req.SetProxyStripRequestHeaders([]string{"x-custom-header"})

// 	req.SetWafEnabled(true)

// 	waf := *openapi.NewWAFConfigWithDefaults()
// 	waf.SetMode("block")
// 	waf.SetParanoiaLevel(1)
// 	waf.SetAllowRules([]string{"10001"})
// 	waf.SetAllowIp([]string{"10.0.0.1"})
// 	waf.SetBlockIp([]string{"127.0.0.1"})
// 	waf.SetBlockUa([]string{"python-requests"})
// 	waf.SetBlockReferer([]string{"illegal-referrer.com"})

// 	// Dictionary support.

// 	httpbl := *openapi.NewHttpblWithDefaults()
// 	httpbl.SetHttpblEnabled(false)
// 	httpbl.SetBlockHarvester(true)
// 	httpbl.SetBlockSearchEngine(true)
// 	httpbl.SetBlockSuspicious(true)
// 	httpbl.SetBlockSpam(true)
// 	// Add API key support.

// 	waf.SetHttpbl(httpbl)

// 	notify := *openapi.NewNotifyConfigWithDefaults()
// 	notify.SetPeriod("120")
// 	notify.SetSlackWebhook("https://slack.com")
// 	notify.SetOriginStatusCodes([]string{"200"})

// 	req.SetNotifyConfig(notify)

// 	req.SetWafConfig(waf)

// 	r, _, err := client.RulesProxyAPI.RulesProxyCreate(ctx, "quant", project).RuleProxyRequest(req).Execute()

// 	if err != nil {
// 		t.Logf("Error: %v", err.Error())
// 		t.Fatalf("Unable to add rule")
// 	}

// 	t.Logf("Success %v", r)
// }
