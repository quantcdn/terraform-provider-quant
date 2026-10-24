package provider_test

import (
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/provider"
)

func shortKVIndexWait(t *testing.T) {
	t.Helper()
	pi, pt := provider.KVIndexWaitInterval, provider.KVIndexWaitTimeout
	provider.KVIndexWaitInterval, provider.KVIndexWaitTimeout = 5*time.Millisecond, 2*time.Second
	t.Cleanup(func() { provider.KVIndexWaitInterval, provider.KVIndexWaitTimeout = pi, pt })
}

func kvIndexBody(state string, fields any) map[string]any {
	if fields == nil {
		fields = []string{"customerId"}
	}
	b := map[string]any{"name": "byCustomer", "collection": "order", "fields": fields, "unique": false, "count": false,
		"slot": 1, "state": state, "createdAt": "2026-10-16T00:00:00.000Z"}
	if state == "FAILED" {
		b["error"] = "UNIQUE_VIOLATION"
		b["conflicts"] = []string{"order:1", "order:2"}
	}
	return b
}

// fakeKVIndex answers the declare with 202 BUILDING, then BUILDING until
// `polls` reads, then `final`. A drop answers 202 DROPPING; the index is then
// 404. deleteStatus overrides the drop answer.
type kvIndexFake struct {
	*kvFake
	reads, dropped atomic.Int64
	deleteStatus   int
	fields         atomic.Value
}

func fakeKVIndex(t *testing.T, polls int64, final string) *kvIndexFake {
	f := &kvIndexFake{kvFake: newKVFake(t), deleteStatus: 202}
	f.on("POST /store-123/indexes", func(body map[string]any) (int, any) {
		f.dropped.Store(0)
		f.reads.Store(0)
		f.fields.Store(body["fields"])
		return 202, kvIndexBody("BUILDING", f.fields.Load())
	})
	f.on("GET /store-123/indexes/byCustomer", func(map[string]any) (int, any) {
		if f.dropped.Load() == 1 {
			return 404, map[string]any{"error": true, "code": "INDEX_NOT_FOUND", "message": "No index", "retryable": false}
		}
		if f.reads.Add(1) < polls {
			return 200, kvIndexBody("BUILDING", f.fields.Load())
		}
		return 200, kvIndexBody(final, f.fields.Load())
	})
	f.on("DELETE /store-123/indexes/byCustomer", func(map[string]any) (int, any) {
		f.dropped.Store(1)
		if f.deleteStatus == 404 {
			return 404, map[string]any{"error": true, "code": "INDEX_NOT_FOUND", "message": "No index", "retryable": false}
		}
		return f.deleteStatus, kvIndexBody("DROPPING", f.fields.Load())
	})
	return f
}

func kvIndexConfig(f *kvIndexFake, fields string) string {
	return f.providerBlock() + `
resource "quant_kv_index" "test" {
  project    = "test-project"
  store_id   = "store-123"
  name       = "byCustomer"
  collection = "order"
  fields     = [` + fields + `]
}
`
}

func TestKVIndex_CreateWaitsForActiveNoDriftAndImport(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 3, "ACTIVE")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvIndexConfig(f, `"customerId"`), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("quant_kv_index.test", "state", "ACTIVE"),
				resource.TestCheckResourceAttr("quant_kv_index.test", "organization", "test-org"),
				resource.TestCheckResourceAttr("quant_kv_index.test", "unique", "false"),
				resource.TestCheckResourceAttr("quant_kv_index.test", "slot", "1"))},
			{Config: kvIndexConfig(f, `"customerId"`), PlanOnly: true},
			{ResourceName: "quant_kv_index.test", ImportState: true, ImportStateId: "test-project/store-123/byCustomer", ImportStateVerify: true,
				ImportStateVerifyIdentifierAttribute: "name"},
		},
	})
	if f.reads.Load() < 3 {
		t.Fatalf("create did not wait for ACTIVE: %d reads", f.reads.Load())
	}
	body := f.bodies("POST /store-123/indexes")[0]
	if body["name"] != "byCustomer" || body["collection"] != "order" || body["unique"] != false || body["count"] != false {
		t.Fatalf("declare body: %v", body)
	}
}

func TestKVIndex_FailedBuildIsAnErrorWithTheReason(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "FAILED")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{Config: kvIndexConfig(f, `"customerId"`),
			ExpectError: regexp.MustCompile(`(?s)KV index build failed.*UNIQUE_VIOLATION.*order:1`)}},
	})
}

func TestKVIndex_ChangedFieldsForceReplace(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "ACTIVE")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvIndexConfig(f, `"customerId"`)},
			{Config: kvIndexConfig(f, `"customerId", "status"`), PlanOnly: true, ExpectNonEmptyPlan: true},
			{Config: kvIndexConfig(f, `"customerId", "status"`), ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("quant_kv_index.test", plancheck.ResourceActionDestroyBeforeCreate)}}},
		},
	})
}

func TestKVIndex_DeleteOf404IsGone(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "ACTIVE")
	f.deleteStatus = 404

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: kvIndexConfig(f, `"customerId"`)}},
	})
	if f.count("DELETE /store-123/indexes/byCustomer") != 1 {
		t.Fatal("destroy did not drop the index")
	}
}

func TestKVIndex_BadImportID(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "ACTIVE")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvIndexConfig(f, `"customerId"`)},
			{ResourceName: "quant_kv_index.test", ImportState: true, ImportStateId: "store-123/byCustomer",
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("project/store_id/name"))},
		},
	})
}
