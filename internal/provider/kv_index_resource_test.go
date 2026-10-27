package provider_test

import (
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

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
// `polls` reads, then `final` (reset to the constructor value at each
// declare). A drop answers 202 DROPPING; the index is then 404. While
// droppingReads is above 0, a read answers DROPPING and counts down; at 0 the
// index is gone. deleteStatus overrides the drop answer.
type kvIndexFake struct {
	*kvFake
	reads, dropped, droppingReads atomic.Int64
	deleteStatus                  int
	fields, final                 atomic.Value
}

var indexNotFound = map[string]any{"error": true, "code": "INDEX_NOT_FOUND", "message": "No index", "retryable": false}

func fakeKVIndex(t *testing.T, polls int64, final string) *kvIndexFake {
	f := &kvIndexFake{kvFake: newKVFake(t), deleteStatus: 202}
	f.final.Store(final)
	f.on("POST /store-123/indexes", func(body map[string]any) (int, any) {
		f.dropped.Store(0)
		f.reads.Store(0)
		f.final.Store(final)
		f.fields.Store(body["fields"])
		return 202, kvIndexBody("BUILDING", f.fields.Load())
	})
	f.on("GET /store-123/indexes/byCustomer", func(map[string]any) (int, any) { return f.show(polls) })
	f.on("DELETE /store-123/indexes/byCustomer", func(map[string]any) (int, any) {
		f.dropped.Store(1)
		if f.deleteStatus == 404 {
			return 404, indexNotFound
		}
		return f.deleteStatus, kvIndexBody("DROPPING", f.fields.Load())
	})
	return f
}

func (f *kvIndexFake) show(polls int64) (int, any) {
	if f.droppingReads.Load() > 0 {
		if f.droppingReads.Add(-1) == 0 {
			f.dropped.Store(1)
		}
		return 200, kvIndexBody("DROPPING", f.fields.Load())
	}
	if f.dropped.Load() == 1 {
		return 404, indexNotFound
	}
	if f.reads.Add(1) < polls {
		return 200, kvIndexBody("BUILDING", f.fields.Load())
	}
	return 200, kvIndexBody(f.final.Load().(string), f.fields.Load())
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

// The KV API allows 10 index changes per minute per project. A 429 on the
// declare and on the drop waits the full Retry-After and sends again.
func TestKVIndex_RateLimitedDeclareAndDropRetry(t *testing.T) {
	shortKVIndexWait(t)
	provider.KVIndexWaitTimeout = 10 * time.Second
	f := fakeKVIndex(t, 1, "ACTIVE")
	f.on("POST /store-123/indexes", limitedTwice(f.routes["POST /store-123/indexes"]))
	f.on("DELETE /store-123/indexes/byCustomer", limitedTwice(f.routes["DELETE /store-123/indexes/byCustomer"]))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{Config: kvIndexConfig(f, `"customerId"`),
			Check: resource.TestCheckResourceAttr("quant_kv_index.test", "state", "ACTIVE")}},
	})
	if n := f.count("POST /store-123/indexes"); n != 3 {
		t.Errorf("declare sent %d times, want 3 (two 429)", n)
	}
	if n := f.count("DELETE /store-123/indexes/byCustomer"); n != 3 {
		t.Errorf("drop sent %d times, want 3 (two 429)", n)
	}
}

// An index that is DROPPING stays in state. The next apply waits until it is
// gone (without a second drop) and then declares it again, so the declare
// does not race the drop.
func TestKVIndex_DroppingIndexStaysInStateUntilGone(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "ACTIVE")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvIndexConfig(f, `"customerId"`)},
			{PreConfig: func() { f.droppingReads.Store(1000) }, RefreshState: true, ExpectNonEmptyPlan: true,
				Check: resource.TestCheckResourceAttr("quant_kv_index.test", "state", "DROPPING")},
			{PreConfig: func() { f.droppingReads.Store(4) }, Config: kvIndexConfig(f, `"customerId"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("quant_kv_index.test", plancheck.ResourceActionDestroyBeforeCreate)}},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("quant_kv_index.test", "state", "ACTIVE"),
					func(*terraform.State) error {
						if n := f.count("DELETE /store-123/indexes/byCustomer"); n != 0 {
							return fmt.Errorf("a DROPPING index was dropped again: %d DELETE", n)
						}
						return nil
					})},
		},
	})
}

// A FAILED index, in state or imported, is replaced by the next apply.
func TestKVIndex_FailedIndexIsReplaced(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "ACTIVE")
	replace := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction("quant_kv_index.test", plancheck.ResourceActionDestroyBeforeCreate)}}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvIndexConfig(f, `"customerId"`)},
			{PreConfig: func() { f.final.Store("FAILED") }, Config: kvIndexConfig(f, `"customerId"`), ConfigPlanChecks: replace,
				Check: resource.TestCheckResourceAttr("quant_kv_index.test", "state", "ACTIVE")},
		},
	})
}

func TestKVIndex_ImportedFailedIndexIsReplaced(t *testing.T) {
	shortKVIndexWait(t)
	f := fakeKVIndex(t, 1, "ACTIVE")
	f.fields.Store([]any{"customerId"})
	f.final.Store("FAILED")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvIndexConfig(f, `"customerId"`), ResourceName: "quant_kv_index.test", ImportState: true,
				ImportStateId: "test-project/store-123/byCustomer", ImportStatePersist: true},
			{Config: kvIndexConfig(f, `"customerId"`), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("quant_kv_index.test", plancheck.ResourceActionDestroyBeforeCreate)}},
				Check: resource.TestCheckResourceAttr("quant_kv_index.test", "state", "ACTIVE")},
		},
	})
}
