package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

// handRegisteredResources maps the resources that generator_config.yml lists
// with `skip: true` (registered by hand in internal/provider too). When the
// portal removes the skip, resources_gen.go maps them and this map must lose
// the entries.
//
// The explicit "name" field opts these resources out of autonaming: a KV
// index or ledger name allows no "-", and the name is part of the definition.
func handRegisteredResources() map[string]*tfbridge.ResourceInfo {
	keepName := map[string]*tfbridge.SchemaInfo{"name": {Name: "name"}}
	importID := compositeComputeID("project", "storeId", "name")
	return map[string]*tfbridge.ResourceInfo{
		"quant_kv_index":  {Tok: "quant:index:KvIndex", ComputeID: importID, Fields: keepName},
		"quant_kv_ledger": {Tok: "quant:index:KvLedger", ComputeID: importID, Fields: keepName},
	}
}

// compositeComputeID joins the named string properties with "/". For a KV
// index or ledger it is the import ID, project/store_id/name: a name alone
// is unique only within one store.
func compositeComputeID(keys ...string) tfbridge.ComputeID {
	return func(_ context.Context, state resource.PropertyMap) (resource.ID, error) {
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			prop, ok := state[resource.PropertyKey(k)]
			if !ok || !prop.IsString() || prop.StringValue() == "" {
				return "", fmt.Errorf("missing %s property in state", k)
			}
			parts = append(parts, prop.StringValue())
		}
		return resource.ID(strings.Join(parts, "/")), nil
	}
}

// resourceMap is the generated map plus the hand-registered resources.
func resourceMap() map[string]*tfbridge.ResourceInfo {
	m := GeneratedResourceMap()
	for k, v := range handRegisteredResources() {
		m[k] = v
	}
	return m
}
