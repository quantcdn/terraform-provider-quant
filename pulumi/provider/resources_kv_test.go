package provider

import (
	"context"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

// The ID of a KV index or ledger is its import ID, project/store_id/name, so
// `pulumi import` and the stored ID agree.
func TestKVComputeIDMatchesTheImportID(t *testing.T) {
	state := resource.PropertyMap{
		"project": resource.NewStringProperty("my-project"),
		"storeId": resource.NewStringProperty("store-123"),
		"name":    resource.NewStringProperty("aiUse"),
	}
	for _, tfName := range []string{"quant_kv_index", "quant_kv_ledger"} {
		id, err := handRegisteredResources()[tfName].ComputeID(context.Background(), state)
		if err != nil || id != "my-project/store-123/aiUse" {
			t.Errorf("%s: ID %q, %v; want my-project/store-123/aiUse", tfName, id, err)
		}
	}
	delete(state, "storeId")
	if _, err := handRegisteredResources()["quant_kv_index"].ComputeID(context.Background(), state); err == nil {
		t.Error("a missing storeId must be an error")
	}
}
