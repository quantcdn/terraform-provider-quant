package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testPEM = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE\n-----END PUBLIC KEY-----\n"

func TestKVLedgerDataSources(t *testing.T) {
	f := newKVFake(t)
	f.json("GET /store-123/ledgers/public-key", 200, map[string]any{
		"keyId": "arn:aws:kms:ap-southeast-2:111122223333:key/k2", "algorithm": "ECDSA_SHA_256", "curve": "P-256", "publicKeyPem": testPEM,
		"keys": []any{
			map[string]any{"keyId": "arn:aws:kms:ap-southeast-2:111122223333:key/k2", "publicKeyPem": testPEM, "current": true},
			map[string]any{"keyId": "arn:aws:kms:ap-southeast-2:111122223333:key/k1", "publicKeyPem": testPEM, "current": false},
		}})
	f.json("GET /store-123/ledgers/aiUse", 200, map[string]any{
		"name": "aiUse", "collection": "ai-use:*", "chains": 4, "retentionDays": 2555, "lockMode": "governance",
		"personalFields": []string{"requesterName"}, "subjectField": "requesterId",
		"anchor": map[string]any{"type": "rfc3161", "tsaUrl": "http://timestamp.digicert.com"},
		"state":  "ACTIVE", "createdAt": "2026-10-16T00:00:00.000Z", "activeAfter": "2026-10-16T00:01:15.000Z",
		"definitionHash": "c4a5", "retainUntil": "2033-10-14T00:00:00.000Z", "lastDigestAt": "2026-10-23T00:05:00.000Z",
		"lastVerify": map[string]any{"at": "2026-10-23T01:00:00.000Z", "startedAt": "2026-10-23T00:59:58.000Z", "ok": true,
			"entries": 120, "shardsFailed": 0, "complete": true, "trigger": "schedule",
			"scope": map[string]any{"collection": "ai-use:a"}},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: f.providerBlock() + `
data "quant_kv_ledger_public_key" "k" {
  project  = "test-project"
  store_id = "store-123"
}

data "quant_kv_ledger" "l" {
  project     = "test-project"
  store_id    = "store-123"
  ledger_name = "aiUse"
}
`,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("data.quant_kv_ledger_public_key.k", "organization", "test-org"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger_public_key.k", "curve", "P-256"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger_public_key.k", "public_key_pem", testPEM),
				resource.TestCheckResourceAttr("data.quant_kv_ledger_public_key.k", "keys.#", "2"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger_public_key.k", "keys.1.current", "false"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "name", "aiUse"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "retain_until", "2033-10-14T00:00:00Z"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "anchor.type", "rfc3161"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "anchor.tsa_url", "http://timestamp.digicert.com"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "personal_fields.0", "requesterName"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "last_verify.ok", "true"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "last_verify.at", "2026-10-23T01:00:00Z"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "last_verify.scope.collection", "ai-use:a"),
				resource.TestCheckResourceAttr("data.quant_kv_ledger.l", "last_digest_at", "2026-10-23T00:05:00Z"),
			),
		}},
	})
}

func TestKVLedgerDataSource_NotFoundIsAnError(t *testing.T) {
	f := newKVFake(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: f.providerBlock() + `
data "quant_kv_ledger" "l" {
  project     = "test-project"
  store_id    = "store-123"
  ledger_name = "missing"
}
`,
			ExpectError: regexp.MustCompile(`Unable to read the KV ledger`),
		}},
	})
}
