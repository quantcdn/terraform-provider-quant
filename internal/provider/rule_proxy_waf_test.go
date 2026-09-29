package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_proxy"
)

// wafConfigWithBlockListsAndHttpbl builds the waf_config the framework hands
// the resource. WafConfigValue declares block_lists and httpbl as plain
// basetypes.ObjectValue, so the sub-objects are plain object values.
func wafConfigWithBlockListsAndHttpbl(ctx context.Context) resource_rule_proxy.WafConfigValue {
	blockListsAttrTypes := resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx)
	httpblAttrTypes := resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx)

	wafAttrTypes := resource_rule_proxy.WafConfigValue{}.AttributeTypes(ctx)

	attrs := map[string]attr.Value{}
	for name, attrType := range wafAttrTypes {
		attrs[name] = nullValueFor(ctx, attrType)
	}

	attrs["block_lists"] = types.ObjectValueMust(blockListsAttrTypes, map[string]attr.Value{
		"ai":         types.BoolValue(true),
		"ip":         types.BoolValue(true),
		"referer":    types.BoolValue(false),
		"user_agent": types.BoolValue(true),
	})
	attrs["httpbl"] = types.ObjectValueMust(httpblAttrTypes, map[string]attr.Value{
		"httpbl_enabled":      types.BoolValue(true),
		"httpbl_key":          types.StringValue("test-key"),
		"block_harvester":     types.BoolValue(true),
		"block_search_engine": types.BoolValue(false),
		"block_spam":          types.BoolValue(true),
		"block_suspicious":    types.BoolValue(false),
	})
	attrs["mode"] = types.StringValue("block")

	return resource_rule_proxy.NewWafConfigValueMust(wafAttrTypes, attrs)
}

// nullValueFor returns the null value of an attribute type, so that a test can
// fill every attribute a generated value constructor insists on.
func nullValueFor(_ context.Context, attrType attr.Type) attr.Value {
	switch t := attrType.(type) {
	case basetypes.ListType:
		return types.ListNull(t.ElemType)
	case basetypes.ObjectType:
		return types.ObjectNull(t.AttrTypes)
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	case basetypes.BoolType:
		return types.BoolNull()
	default:
		return nil
	}
}

// TestBuildProxyRequest_WafBlockListsAndHttpbl reproduces the same defect the
// crawler assets block had: reading a plain basetypes.ObjectValue into a
// generated BlockListsValue or HttpblValue raises a framework Value
// Conversion Error.
func TestBuildProxyRequest_WafBlockListsAndHttpbl(t *testing.T) {
	ctx := context.Background()

	data := &resource_rule_proxy.RuleProxyModel{
		Name:       types.StringValue("test"),
		Project:    types.StringValue("default"),
		WafEnabled: types.BoolValue(true),
		WafConfig:  wafConfigWithBlockListsAndHttpbl(ctx),
	}

	req, diags := buildProxyRequest(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}

	waf, ok := req.GetWafConfigOk()
	if !ok || waf == nil {
		t.Fatal("waf_config was not set on the request")
	}

	bl, ok := waf.GetBlockListsOk()
	if !ok || bl == nil {
		t.Fatal("block_lists was not set on the request")
	}
	if !bl.GetAi() || !bl.GetIp() || bl.GetReferer() || !bl.GetUserAgent() {
		t.Errorf("block_lists = ai:%v ip:%v referer:%v user_agent:%v, want true true false true",
			bl.GetAi(), bl.GetIp(), bl.GetReferer(), bl.GetUserAgent())
	}

	hb, ok := waf.GetHttpblOk()
	if !ok || hb == nil {
		t.Fatal("httpbl was not set on the request")
	}
	if !hb.GetHttpblEnabled() {
		t.Error("httpbl.httpbl_enabled = false, want true")
	}
	if got := hb.GetHttpblKey(); got != "test-key" {
		t.Errorf("httpbl.httpbl_key = %q, want %q", got, "test-key")
	}
	if !hb.GetBlockHarvester() || hb.GetBlockSearchEngine() || !hb.GetBlockSpam() || hb.GetBlockSuspicious() {
		t.Error("httpbl block flags did not round-trip")
	}
}

// TestBuildProxyRequest_NoWafConfig guards the common case: a proxy rule with
// no waf_config must be unaffected.
func TestBuildProxyRequest_NoWafConfig(t *testing.T) {
	ctx := context.Background()

	data := &resource_rule_proxy.RuleProxyModel{
		Name:       types.StringValue("test"),
		Project:    types.StringValue("default"),
		WafEnabled: types.BoolValue(false),
		WafConfig:  resource_rule_proxy.NewWafConfigValueNull(),
	}

	req, diags := buildProxyRequest(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}
	if _, ok := req.GetWafConfigOk(); ok {
		t.Error("waf_config must stay unset when the rule configures none")
	}
	if got := req.GetName(); got != "test" {
		t.Errorf("name = %q, want %q", got, "test")
	}
}
