package provider_test

import (
	"encoding/json"
	"testing"

	openapi "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

// Static schema from an API request
//
//	curl --request GET \
//	 --url http://localhost:8001/api/v2/organizations/quant/projects/api-test/rules/proxy \
//	 --header 'Authorization: Bearer $QUANT_BEARER' \
//	 --header 'User-Agent: insomnia/8.3.0'
func TestProxySchema(t *testing.T) {
	jsonData := []byte(`[
		{
			"action_config": {
				"notify_config": {
					"period": 60,
					"slack_webhook": 60,
					"origin_status_codes": []
				},
				"origin_timeout": "30000",
				"proxy_strip_headers": [],
				"proxy_alert_enabled": true,
				"failover_mode": false,
				"failover_lifetime": "300",
				"inject_headers": null,
				"notify": "none",
				"host": "www.example.com",
				"cache_lifetime": null,
				"waf_config": {
					"block_lists": {
						"user_agent": false,
						"ip": false,
						"referer": false,
						"ai": false
					},
					"notify_email": [],
					"block_referer": [],
					"notify_slack_hits_rpm": null,
					"thresholds": [
						{
							"mode": "disabled",
							"rps": 5,
							"cooldown": 30,
							"notify_slack": "",
							"type": "ip"
						}
					],
					"mode": "report",
					"httpbl": {
						"api_key": "",
						"block_suspicious": false,
						"block_harvester": false,
						"block_spam": false,
						"block_search_engine": false,
						"httpbl_enabled": false
					},
					"block_ua": [
						"Mozilla\/5.0 (compatible; Googlebot\/2.1; +http:\/\/www.google.com\/bot.html)"
					],
					"block_ip": [],
					"allow_rules": [],
					"paranoia_level": 0,
					"notify_slack": "",
					"allow_ip": [
						"10.0.0.1"
					]
				},
				"waf_enabled": true,
				"to": "http:\/\/origin.example.com",
				"auth_user": "",
				"auth_pass": "",
				"only_proxy_404": false,
				"failover_origin_ttfb": "2000",
				"disable_ssl_verify": true,
				"failover_origin_status_codes": []
			},
			"country_is_not": [],
			"domain": [
				"*"
			],
			"ip": "",
			"action": "proxy",
			"name": "proxyrule",
			"uuid": "5da31117-7fbd-48bd-8fdf-08ab8218ae87",
			"url": [
				"\/test-proxy"
			],
			"disabled": false,
			"method_is_not": [],
			"ip_is": [],
			"ip_is_not": [],
			"country_is": [
				"AU"
			],
			"method": "",
			"country": "country_is",
			"method_is": [],
			"only_with_cookie": ""
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
