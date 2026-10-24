package provider

import (
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
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
	return map[string]*tfbridge.ResourceInfo{
		"quant_kv_index":  {Tok: "quant:index:KvIndex", ComputeID: fieldComputeID("name"), Fields: keepName},
		"quant_kv_ledger": {Tok: "quant:index:KvLedger", ComputeID: fieldComputeID("name"), Fields: keepName},
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
