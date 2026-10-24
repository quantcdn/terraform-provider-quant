package provider_test

import (
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fakeKVStore serves one store whose real format is *format.
func fakeKVStore(t *testing.T, format *atomic.Int64) *kvFake {
	f := newKVFake(t)
	store := map[string]any{"id": "store-123", "name": "my-store"}
	f.on("POST ", func(body map[string]any) (int, any) {
		if v, ok := body["format"].(float64); ok {
			format.Store(int64(v))
		}
		return 200, store
	})
	f.json("GET /store-123", 200, store)
	f.json("GET /store-123/items", 200, map[string]any{"data": []any{}, "next_cursor": nil})
	f.on("GET /store-123/upgrade", func(map[string]any) (int, any) {
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
