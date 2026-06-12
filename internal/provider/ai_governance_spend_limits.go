package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_governance"
)

// Conversions for the spend_limits.interface_limits and spend_limits.user_overrides
// map-nested attributes (added in API v4.19.0). interface_limits keys an interface
// label (e.g. "slack", "autonomous") to per-interface daily/monthly cent caps;
// user_overrides keys a user id to a named per-user cap (or unlimited).

// interfaceLimitsToSDK converts the Terraform interface_limits map into the SDK
// map for a governance write. The caller guards against null/unknown.
func interfaceLimitsToSDK(ctx context.Context, m types.Map) (map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	elems := map[string]resource_ai_governance.InterfaceLimitsValue{}
	diags.Append(m.ElementsAs(ctx, &elems, false)...)
	out := make(map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue, len(elems))
	for label, il := range elems {
		v := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue()
		if !il.DailyCents.IsNull() && !il.DailyCents.IsUnknown() {
			v.SetDailyCents(int32(il.DailyCents.ValueInt64()))
		}
		if !il.MonthlyCents.IsNull() && !il.MonthlyCents.IsUnknown() {
			v.SetMonthlyCents(int32(il.MonthlyCents.ValueInt64()))
		}
		out[label] = *v
	}
	return out, diags
}

// userOverridesToSDK converts the Terraform user_overrides map into the SDK map
// for a governance write. The caller guards against null/unknown.
func userOverridesToSDK(ctx context.Context, m types.Map) (map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsUserOverridesValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	elems := map[string]resource_ai_governance.UserOverridesValue{}
	diags.Append(m.ElementsAs(ctx, &elems, false)...)
	out := make(map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsUserOverridesValue, len(elems))
	for label, uo := range elems {
		v := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimitsUserOverridesValue()
		if !uo.DailyCents.IsNull() && !uo.DailyCents.IsUnknown() {
			v.SetDailyCents(int32(uo.DailyCents.ValueInt64()))
		}
		if !uo.MonthlyCents.IsNull() && !uo.MonthlyCents.IsUnknown() {
			v.SetMonthlyCents(int32(uo.MonthlyCents.ValueInt64()))
		}
		if !uo.Unlimited.IsNull() && !uo.Unlimited.IsUnknown() {
			v.SetUnlimited(uo.Unlimited.ValueBool())
		}
		out[label] = *v
	}
	return out, diags
}

// interfaceLimitsToTF builds the Terraform interface_limits map from the typed
// SDK response (Read / ImportState path).
func interfaceLimitsToTF(ctx context.Context, sdkMap map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := resource_ai_governance.InterfaceLimitsValue{}.Type(ctx)
	if len(sdkMap) == 0 {
		return types.MapNull(elemType), diags
	}
	attrTypes := resource_ai_governance.InterfaceLimitsValue{}.AttributeTypes(ctx)
	elems := make(map[string]attr.Value, len(sdkMap))
	for label, il := range sdkMap {
		v, d := resource_ai_governance.NewInterfaceLimitsValue(attrTypes, map[string]attr.Value{
			"daily_cents":   nullableInt32ToInt64(il.DailyCents),
			"monthly_cents": nullableInt32ToInt64(il.MonthlyCents),
		})
		diags.Append(d...)
		elems[label] = v
	}
	out, d := types.MapValue(elemType, elems)
	diags.Append(d...)
	return out, diags
}

// userOverridesToTF builds the Terraform user_overrides map from the typed SDK
// response (Read / ImportState path).
func userOverridesToTF(ctx context.Context, sdkMap map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsUserOverridesValue) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := resource_ai_governance.UserOverridesValue{}.Type(ctx)
	if len(sdkMap) == 0 {
		return types.MapNull(elemType), diags
	}
	attrTypes := resource_ai_governance.UserOverridesValue{}.AttributeTypes(ctx)
	elems := make(map[string]attr.Value, len(sdkMap))
	for label, uo := range sdkMap {
		v, d := resource_ai_governance.NewUserOverridesValue(attrTypes, map[string]attr.Value{
			"daily_cents":   nullableInt32ToInt64(uo.DailyCents),
			"monthly_cents": nullableInt32ToInt64(uo.MonthlyCents),
			"unlimited":     nullableBoolToBool(uo.Unlimited),
		})
		diags.Append(d...)
		elems[label] = v
	}
	out, d := types.MapValue(elemType, elems)
	diags.Append(d...)
	return out, diags
}

// interfaceLimitsFromRawMap builds the Terraform interface_limits map from the
// raw config map returned by the PUT (UpdateGovernanceConfig) response.
func interfaceLimitsFromRawMap(ctx context.Context, raw interface{}) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := resource_ai_governance.InterfaceLimitsValue{}.Type(ctx)
	m, ok := raw.(map[string]interface{})
	if !ok || len(m) == 0 {
		return types.MapNull(elemType), diags
	}
	attrTypes := resource_ai_governance.InterfaceLimitsValue{}.AttributeTypes(ctx)
	elems := make(map[string]attr.Value, len(m))
	for label, rv := range m {
		entry, _ := rv.(map[string]interface{})
		v, d := resource_ai_governance.NewInterfaceLimitsValue(attrTypes, map[string]attr.Value{
			"daily_cents":   optionalInt64FromMap(entry, "dailyCents"),
			"monthly_cents": optionalInt64FromMap(entry, "monthlyCents"),
		})
		diags.Append(d...)
		elems[label] = v
	}
	out, d := types.MapValue(elemType, elems)
	diags.Append(d...)
	return out, diags
}

// userOverridesFromRawMap builds the Terraform user_overrides map from the raw
// config map returned by the PUT (UpdateGovernanceConfig) response.
func userOverridesFromRawMap(ctx context.Context, raw interface{}) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := resource_ai_governance.UserOverridesValue{}.Type(ctx)
	m, ok := raw.(map[string]interface{})
	if !ok || len(m) == 0 {
		return types.MapNull(elemType), diags
	}
	attrTypes := resource_ai_governance.UserOverridesValue{}.AttributeTypes(ctx)
	elems := make(map[string]attr.Value, len(m))
	for label, rv := range m {
		entry, _ := rv.(map[string]interface{})
		v, d := resource_ai_governance.NewUserOverridesValue(attrTypes, map[string]attr.Value{
			"daily_cents":   optionalInt64FromMap(entry, "dailyCents"),
			"monthly_cents": optionalInt64FromMap(entry, "monthlyCents"),
			"unlimited":     optionalBoolFromMap(entry, "unlimited"),
		})
		diags.Append(d...)
		elems[label] = v
	}
	out, d := types.MapValue(elemType, elems)
	diags.Append(d...)
	return out, diags
}

// nullableBoolToBool converts an SDK NullableBool to a Terraform Bool value.
func nullableBoolToBool(n quantadmingo.NullableBool) types.Bool {
	if n.IsSet() && n.Get() != nil {
		return types.BoolValue(*n.Get())
	}
	return types.BoolNull()
}

// optionalBoolFromMap extracts a bool from a map[string]interface{}, returning
// types.BoolNull() if absent, nil, or not a bool.
func optionalBoolFromMap(m map[string]interface{}, key string) types.Bool {
	v, ok := m[key]
	if !ok || v == nil {
		return types.BoolNull()
	}
	if b, ok := v.(bool); ok {
		return types.BoolValue(b)
	}
	return types.BoolNull()
}
