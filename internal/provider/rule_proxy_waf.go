package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_proxy"
)

var _ resource.ResourceWithModifyPlan = (*ruleProxyResource)(nil)

// ModifyPlan keeps the prior waf_config.thresholds when the plan leaves them
// unknown. A change to another attribute makes the framework re-plan the
// unconfigured Optional+Computed thresholds as unknown, and the Pulumi bridge
// then reports a permanent update.
func (r *ruleProxyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var waf resource_rule_proxy.WafConfigValue
	if d := req.Plan.GetAttribute(ctx, path.Root("waf_config"), &waf); d.HasError() || waf.IsNull() || waf.IsUnknown() {
		return
	}
	thresholdsPath := path.Root("waf_config").AtName("thresholds")
	var prior types.List
	if d := req.State.GetAttribute(ctx, thresholdsPath, &prior); d.HasError() {
		return
	}
	// Only a known prior value is kept: a null prior would plan null, and the
	// read after apply returns the stored thresholds.
	planned := waf.Thresholds
	if !planned.IsUnknown() || !isKnown(prior) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, thresholdsPath, keepPriorThresholdsIfUnknown(planned, prior))...)
}

// dropThresholdsModeDefault removes the generated Default from
// waf_config.thresholds[*].mode. The framework applies a nested default
// wherever the configuration is null, which includes every threshold that
// the API returns when thresholds are not configured. The planned mode then
// becomes "disabled" while the state holds the stored mode, so every plan
// shows an update. The API applies the same default itself.
func dropThresholdsModeDefault(attrs map[string]schema.Attribute) {
	waf, ok := attrs["waf_config"].(schema.SingleNestedAttribute)
	if !ok {
		return
	}
	thresholds, ok := waf.Attributes["thresholds"].(schema.ListNestedAttribute)
	if !ok {
		return
	}
	mode, ok := thresholds.NestedObject.Attributes["mode"].(schema.StringAttribute)
	if !ok {
		return
	}
	mode.Default = nil
	thresholds.NestedObject.Attributes["mode"] = mode
	waf.Attributes["thresholds"] = thresholds
	attrs["waf_config"] = waf
}

// keepPriorThresholdsIfUnknown returns the prior thresholds when the planned
// value is unknown, mirroring UseStateForUnknown.
func keepPriorThresholdsIfUnknown(planned, prior types.List) types.List {
	if planned.IsUnknown() {
		return prior
	}
	return planned
}

// buildWafConfig maps the waf_config block to the SDK request value.
func buildWafConfig(ctx context.Context, w resource_rule_proxy.WafConfigValue) (*quantadmingo.WafConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	wafConfig := quantadmingo.NewWafConfig()
	wafConfig.SetMode(w.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(w.ParanoiaLevel.ValueInt64()))

	setWafStringList(ctx, w.AllowRules, wafConfig.SetAllowRules, &diags)
	setWafStringList(ctx, w.AllowIp, wafConfig.SetAllowIp, &diags)
	setWafStringList(ctx, w.BlockIp, wafConfig.SetBlockIp, &diags)
	setWafStringList(ctx, w.BlockUa, wafConfig.SetBlockUa, &diags)
	setWafStringList(ctx, w.BlockReferer, wafConfig.SetBlockReferer, &diags)
	setWafStringList(ctx, w.NotifyEmail, wafConfig.SetNotifyEmail, &diags)
	setWafStringList(ctx, w.BlockAsn, wafConfig.SetBlockAsn, &diags)
	if diags.HasError() {
		return nil, diags
	}

	if isKnown(w.NotifySlack) {
		wafConfig.SetNotifySlack(w.NotifySlack.ValueString())
	}
	if isKnown(w.NotifySlackHitsRpm) {
		wafConfig.SetNotifySlackHitsRpm(int32(w.NotifySlackHitsRpm.ValueInt64()))
	}

	diags.Append(buildWafBlockLists(ctx, w.BlockLists, wafConfig)...)
	thresholds, d := buildWafThresholds(ctx, w.Thresholds)
	diags.Append(d...)
	if thresholds != nil {
		wafConfig.SetThresholds(thresholds)
	}
	diags.Append(buildWafHttpbl(ctx, w.Httpbl, wafConfig)...)
	return wafConfig, diags
}

// buildWafBlockLists maps waf_config.block_lists. WafConfigValue stores
// block_lists as a plain basetypes.ObjectValue, so ObjectValue.As into the
// generated BlockListsValue raises a framework Value Conversion Error. Build
// the value directly.
func buildWafBlockLists(ctx context.Context, obj basetypes.ObjectValue, wafConfig *quantadmingo.WafConfig) diag.Diagnostics {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	v, diags := resource_rule_proxy.NewBlockListsValue(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx), obj.Attributes())
	if diags.HasError() {
		return diags
	}
	blockLists := quantadmingo.NewWafConfigBlockLists()
	blockLists.SetAi(v.Ai.ValueBool())
	blockLists.SetIp(v.Ip.ValueBool())
	blockLists.SetReferer(v.Referer.ValueBool())
	blockLists.SetUserAgent(v.UserAgent.ValueBool())
	wafConfig.SetBlockLists(*blockLists)
	return diags
}

// buildWafHttpbl maps waf_config.httpbl, a plain basetypes.ObjectValue like
// block_lists.
func buildWafHttpbl(ctx context.Context, obj basetypes.ObjectValue, wafConfig *quantadmingo.WafConfig) diag.Diagnostics {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	v, diags := resource_rule_proxy.NewHttpblValue(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx), obj.Attributes())
	if diags.HasError() {
		return diags
	}
	httpbl := quantadmingo.NewWafConfigHttpbl()
	httpbl.SetHttpblEnabled(v.HttpblEnabled.ValueBool())
	httpbl.SetHttpblKey(v.HttpblKey.ValueString())
	httpbl.SetBlockHarvester(v.BlockHarvester.ValueBool())
	httpbl.SetBlockSearchEngine(v.BlockSearchEngine.ValueBool())
	httpbl.SetBlockSpam(v.BlockSpam.ValueBool())
	httpbl.SetBlockSuspicious(v.BlockSuspicious.ValueBool())
	wafConfig.SetHttpbl(*httpbl)
	return diags
}

// buildWafThresholds maps waf_config.thresholds. It returns nil for a null
// or unknown list, so the request leaves the stored thresholds alone.
func buildWafThresholds(ctx context.Context, list types.List) ([]quantadmingo.WafConfigThresholdsInner, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var objs []resource_rule_proxy.ThresholdsValue
	diags := list.ElementsAs(ctx, &objs, false)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]quantadmingo.WafConfigThresholdsInner, len(objs))
	for i, t := range objs {
		out[i] = thresholdToSDK(t)
	}
	return out, diags
}

// thresholdToSDK sends only the known attributes of one threshold.
func thresholdToSDK(t resource_rule_proxy.ThresholdsValue) quantadmingo.WafConfigThresholdsInner {
	th := quantadmingo.NewWafConfigThresholdsInner()
	strs := []struct {
		v   types.String
		set func(string)
	}{
		{t.ThresholdsType, th.SetType}, {t.Mode, th.SetMode},
		{t.Value, th.SetValue}, {t.NotifySlack, th.SetNotifySlack},
	}
	for _, s := range strs {
		if isKnown(s.v) {
			s.set(s.v.ValueString())
		}
	}
	ints := []struct {
		v   types.Int64
		set func(int32)
	}{
		{t.Rps, th.SetRps}, {t.Cooldown, th.SetCooldown},
		{t.Hits, th.SetHits}, {t.Minutes, th.SetMinutes},
	}
	for _, n := range ints {
		if isKnown(n.v) {
			n.set(int32(n.v.ValueInt64()))
		}
	}
	return *th
}

// wafThresholdsFromAPI reads waf_config.thresholds back from the API.
//
// The API merges thresholds by type and always stores all three types. When
// the prior value is known (configured, or already in state) the result keeps
// the prior types, in the prior order, filled with the stored values. When the
// prior value is null or unknown the result is every stored threshold. No
// stored thresholds read back as null.
func wafThresholdsFromAPI(ctx context.Context, api []quantadmingo.WafConfigThresholdsInner, prior types.List) (types.List, diag.Diagnostics) {
	elemType := resource_rule_proxy.ThresholdsValue{}.Type(ctx)
	if len(api) == 0 {
		return types.ListNull(elemType), nil
	}
	if !isKnown(prior) {
		elems := make([]attr.Value, len(api))
		for i, th := range api {
			elems[i] = thresholdFromAPI(ctx, th)
		}
		return types.ListValue(elemType, elems)
	}
	return projectThresholds(ctx, api, prior)
}

// projectThresholds returns the prior threshold entries with the values the
// API stored for the same type. An entry with no stored type keeps its own
// values, with unknown attributes set to null.
func projectThresholds(ctx context.Context, api []quantadmingo.WafConfigThresholdsInner, prior types.List) (types.List, diag.Diagnostics) {
	elemType := resource_rule_proxy.ThresholdsValue{}.Type(ctx)
	var priorObjs []resource_rule_proxy.ThresholdsValue
	diags := prior.ElementsAs(ctx, &priorObjs, false)
	if diags.HasError() {
		return prior, diags
	}
	byType := make(map[string]quantadmingo.WafConfigThresholdsInner, len(api))
	for _, th := range api {
		byType[th.GetType()] = th
	}
	elems := make([]attr.Value, len(priorObjs))
	for i, p := range priorObjs {
		if th, ok := byType[p.ThresholdsType.ValueString()]; ok && isKnown(p.ThresholdsType) {
			elems[i] = thresholdFromAPI(ctx, th)
			continue
		}
		elems[i] = nullUnknownThreshold(p)
	}
	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}

// thresholdFromAPI builds one threshold value. Absent attributes are null.
func thresholdFromAPI(ctx context.Context, th quantadmingo.WafConfigThresholdsInner) resource_rule_proxy.ThresholdsValue {
	value, _ := th.GetValueOk()
	notify, _ := th.GetNotifySlackOk()
	return resource_rule_proxy.NewThresholdsValueMust(resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx), map[string]attr.Value{
		"type":         types.StringPointerValue(th.Type),
		"mode":         types.StringPointerValue(th.Mode),
		"value":        types.StringPointerValue(value),
		"notify_slack": types.StringPointerValue(notify),
		"rps":          int64FromInt32Ptr(th.Rps),
		"cooldown":     int64FromInt32Ptr(th.Cooldown),
		"hits":         int64FromInt32Ptr(th.Hits),
		"minutes":      int64FromInt32Ptr(th.Minutes),
	})
}

func int64FromInt32Ptr(p *int32) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// nullUnknownThreshold sets the unknown attributes of a threshold to null.
func nullUnknownThreshold(t resource_rule_proxy.ThresholdsValue) resource_rule_proxy.ThresholdsValue {
	for _, s := range []*types.String{&t.ThresholdsType, &t.Mode, &t.Value, &t.NotifySlack} {
		if s.IsUnknown() {
			*s = types.StringNull()
		}
	}
	for _, n := range []*types.Int64{&t.Rps, &t.Cooldown, &t.Hits, &t.Minutes} {
		if n.IsUnknown() {
			*n = types.Int64Null()
		}
	}
	return t
}

// notifySlackHitsRpmFromAPI reads waf_config.notify_slack_hits_rpm. The API
// stores 0 as "off" and returns it as null. An explicit 0 in the prior value
// is kept so that the apply result matches the plan.
func notifySlackHitsRpmFromAPI(w *quantadmingo.WafConfig, prior types.Int64) types.Int64 {
	if v, ok := w.GetNotifySlackHitsRpmOk(); ok && *v != 0 {
		return types.Int64Value(int64(*v))
	}
	if isKnown(prior) && prior.ValueInt64() == 0 {
		return prior
	}
	return types.Int64Null()
}

// setWafConfigFromAPI reads an enabled WAF configuration back into the model.
func setWafConfigFromAPI(ctx context.Context, wafConfig quantadmingo.WafConfig, data *resource_rule_proxy.RuleProxyModel) diag.Diagnostics {
	var diags diag.Diagnostics
	w := &data.WafConfig
	w.Mode = types.StringValue(wafConfig.GetMode())
	w.ParanoiaLevel = types.Int64Value(int64(wafConfig.GetParanoiaLevel()))

	diags.Append(setStringListFromAPI(ctx, wafConfig.GetAllowRules(), &w.AllowRules)...)
	diags.Append(setStringListFromAPI(ctx, wafConfig.GetAllowIp(), &w.AllowIp)...)
	diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockIp(), &w.BlockIp)...)
	diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockUa(), &w.BlockUa)...)
	diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockReferer(), &w.BlockReferer)...)
	diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockAsn(), &w.BlockAsn)...)
	diags.Append(setStringListFromAPI(ctx, wafConfig.GetNotifyEmail(), &w.NotifyEmail)...)

	w.NotifySlack = types.StringValue(wafConfig.GetNotifySlack())
	w.NotifySlackHitsRpm = notifySlackHitsRpmFromAPI(&wafConfig, w.NotifySlackHitsRpm)

	// Block lists and httpbl: preserve from prior state to avoid custom type
	// conversion issues. User-configured values round-trip via the plan.
	if w.BlockLists.IsUnknown() {
		w.BlockLists = types.ObjectNull(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx))
	}
	if w.Httpbl.IsUnknown() {
		w.Httpbl = types.ObjectNull(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx))
	}

	thresholds, d := wafThresholdsFromAPI(ctx, wafConfig.GetThresholds(), w.Thresholds)
	diags.Append(d...)
	w.Thresholds = thresholds
	return diags
}

// setWafDisabledDefaults fills a configured waf_config block when the WAF is
// off, so that every Optional+Computed attribute is known after apply.
func setWafDisabledDefaults(ctx context.Context, data *resource_rule_proxy.RuleProxyModel) {
	w := &data.WafConfig
	if w.Mode.IsNull() || w.Mode.IsUnknown() {
		return
	}
	emptyStringList, _ := types.ListValue(types.StringType, []attr.Value{})
	w.AllowRules = emptyStringList
	w.AllowIp = emptyStringList
	w.BlockIp = emptyStringList
	w.BlockUa = emptyStringList
	w.BlockReferer = emptyStringList
	w.NotifyEmail = emptyStringList
	w.NotifySlack = types.StringValue("")
	w.NotifySlackHitsRpm = types.Int64Null()
	w.BlockAsn = emptyStringList
	w.Httpbl = types.ObjectNull(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx))
	w.BlockLists = types.ObjectNull(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx))
	thObjType := types.ObjectType{AttrTypes: resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx)}
	w.Thresholds, _ = types.ListValue(thObjType, []attr.Value{})
	if w.ParanoiaLevel.IsNull() || w.ParanoiaLevel.IsUnknown() {
		w.ParanoiaLevel = types.Int64Value(1)
	}
}
