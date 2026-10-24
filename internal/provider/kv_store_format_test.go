package provider_test

import (
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fakeKVStore serves one store whose real format is *format. While
// upgradeStatus is not 0, the upgrade endpoint answers that status.
var upgradeStatus atomic.Int64

func fakeKVStore(t *testing.T, format *atomic.Int64) *kvFake {
	upgradeStatus.Store(0)
	t.Cleanup(func() { upgradeStatus.Store(0) })
	f := newKVFake(t)
	store := map[string]any{"id": "store-123", "name": "my-store"}
	f.on("POST ", func(body map[string]any) (int, any) {
		if v, ok := body["format"].(float64); ok {
			format.Store(int64(v))
		}
		return 200, store
	})
	// KV_show reports the readiness status; KV_create may leave it out.
	f.json("GET /store-123", 200, map[string]any{"id": "store-123", "name": "my-store", "status": "ready"})
	f.json("GET /store-123/items", 200, map[string]any{"data": []any{}, "next_cursor": nil})
	f.on("GET /store-123/upgrade", func(map[string]any) (int, any) {
		if st := upgradeStatus.Load(); st != 0 {
			return int(st), map[string]any{"error": true, "code": "UPSTREAM", "message": "upgrade state unavailable", "retryable": true}
		}
		return 200, map[string]any{"format": format.Load(), "state": "active", "table": "t", "upgrade": nil}
	})
	f.json("DELETE /store-123", 200, map[string]any{"success": true})
	return f
}

func kvStoreConfig(f *kvFake, formatLine string) string {
	return f.providerBlock() + fmt.Sprintf(`
resource "quant_kv_store" "test" {
  project = "test-project"
  name    = "my-store"
  %s
}
`, formatLine)
}

func TestKVStoreFormat_SentOnCreateReadBackAndChangeIsAPlanError(t *testing.T) {
	var format atomic.Int64
	f := fakeKVStore(t, &format)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvStoreConfig(f, "format = 2"), Check: resource.TestCheckResourceAttr("quant_kv_store.test", "format", "2")},
			{Config: kvStoreConfig(f, "format = 2"), PlanOnly: true},
			{Config: kvStoreConfig(f, "format = 1"), PlanOnly: true, ExpectError: regexp.MustCompile(`(?s)format cannot change.*upgrade API`)},
			{ResourceName: "quant_kv_store.test", ImportState: true, ImportStateId: "test-project/store-123", ImportStateVerify: true},
		},
	})
	if got := f.bodies("POST ")[0]["format"]; got != float64(2) {
		t.Fatalf("create sent format %v, want 2", got)
	}
}

func TestKVStoreFormat_OmittedIsFormat1AndNotSent(t *testing.T) {
	var format atomic.Int64
	format.Store(1)
	f := fakeKVStore(t, &format)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvStoreConfig(f, ""), Check: resource.TestCheckResourceAttr("quant_kv_store.test", "format", "1")},
			{Config: kvStoreConfig(f, ""), PlanOnly: true},
		},
	})
}

// A store upgraded through the API reads back as format 2: the old config is
// a plan error that names the change, and format = 2 plans no change.
func TestKVStoreFormat_UpgradeOutsideTerraformIsDetected(t *testing.T) {
	var format atomic.Int64
	f := fakeKVStore(t, &format)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvStoreConfig(f, "format = 1")},
			{PreConfig: func() { format.Store(2) }, Config: kvStoreConfig(f, "format = 1"), PlanOnly: true,
				ExpectError: regexp.MustCompile(`The store is format 2 and the configuration asks for format 1`)},
			{Config: kvStoreConfig(f, "format = 2"), Check: resource.TestCheckResourceAttr("quant_kv_store.test", "format", "2")},
		},
	})
}

// A store_id in the configuration that differs from state is a plan error,
// not a replacement and not an update that points at another store.
func TestKVStore_StoreIDChangeIsAPlanError(t *testing.T) {
	var format atomic.Int64
	f := fakeKVStore(t, &format)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvStoreConfig(f, "")},
			{Config: kvStoreConfig(f, `store_id = "store-123"`), PlanOnly: true},
			{Config: kvStoreConfig(f, `store_id = "store-999"`), PlanOnly: true,
				ExpectError: regexp.MustCompile(`(?s)store_id cannot change.*"store-123".*"store-999"`)},
		},
	})
}

// A failed upgrade endpoint (403, 5xx) keeps the prior format with a
// warning: the read does not fail and the plan does not change. A 404 means
// format 1.
func TestKVStoreFormat_UpgradeEndpointErrorKeepsThePriorFormat(t *testing.T) {
	var format atomic.Int64
	f := fakeKVStore(t, &format)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: kvStoreConfig(f, "format = 2")},
			{PreConfig: func() { upgradeStatus.Store(403) }, Config: kvStoreConfig(f, "format = 2"), PlanOnly: true},
			{PreConfig: func() { upgradeStatus.Store(503) }, Config: kvStoreConfig(f, "format = 2"),
				Check: resource.TestCheckResourceAttr("quant_kv_store.test", "format", "2")},
			{PreConfig: func() { upgradeStatus.Store(404) }, Config: kvStoreConfig(f, "format = 2"), PlanOnly: true,
				ExpectError: regexp.MustCompile(`The store is format 1 and the configuration asks for format 2`)},
			{PreConfig: func() { upgradeStatus.Store(0) }, Config: kvStoreConfig(f, "format = 2"), PlanOnly: true},
		},
	})
}
