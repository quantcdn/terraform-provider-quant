package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_proxy"
)

// thresholdValue builds a generated ThresholdsValue. Unset attributes are
// null; pass types.Int64Unknown() or types.StringUnknown() to model a value
// the plan leaves to the provider.
func thresholdValue(ctx context.Context, attrs map[string]attr.Value) resource_rule_proxy.ThresholdsValue {
	all := map[string]attr.Value{
		"cooldown": types.Int64Null(), "hits": types.Int64Null(), "minutes": types.Int64Null(),
		"rps": types.Int64Null(), "mode": types.StringNull(), "notify_slack": types.StringNull(),
		"type": types.StringNull(), "value": types.StringNull(),
	}
	for k, v := range attrs {
		all[k] = v
	}
	return resource_rule_proxy.NewThresholdsValueMust(resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx), all)
}

func thresholdList(ctx context.Context, vals ...resource_rule_proxy.ThresholdsValue) types.List {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = v
	}
	return types.ListValueMust(resource_rule_proxy.ThresholdsValue{}.Type(ctx), elems)
}

func ptr[T any](v T) *T { return &v }

// apiDefaultThresholds is what the API stores and returns: all three types.
func apiDefaultThresholds() []quantadmingo.WafConfigThresholdsInner {
	ip := quantadmingo.WafConfigThresholdsInner{Type: ptr("ip"), Rps: ptr(int32(20)), Cooldown: ptr(int32(30)), Mode: ptr("block")}
	ip.SetNotifySlack("")
	header := quantadmingo.WafConfigThresholdsInner{Type: ptr("header"), Rps: ptr(int32(5)), Cooldown: ptr(int32(30)), Mode: ptr("disabled")}
	header.SetValue("")
	header.SetNotifySlack("")
	hit := quantadmingo.WafConfigThresholdsInner{Type: ptr("waf_hit_by_ip"), Hits: ptr(int32(10)), Minutes: ptr(int32(5)), Cooldown: ptr(int32(300)), Mode: ptr("bot_challenge")}
	hit.SetNotifySlack("")
	return []quantadmingo.WafConfigThresholdsInner{ip, header, hit}
}

func thresholdsOf(t *testing.T, ctx context.Context, l types.List) []resource_rule_proxy.ThresholdsValue {
	t.Helper()
	var out []resource_rule_proxy.ThresholdsValue
	if d := l.ElementsAs(ctx, &out, false); d.HasError() {
		t.Fatalf("elements: %v", d)
	}
	return out
}

func TestBuildWafThresholds_SendsOnlyKnownAttributes(t *testing.T) {
	ctx := context.Background()
	list := thresholdList(ctx, thresholdValue(ctx, map[string]attr.Value{
		"type": types.StringValue("ip"), "rps": types.Int64Value(20), "mode": types.StringValue("bot_challenge"),
		"cooldown": types.Int64Unknown(),
	}))

	got, d := buildWafThresholds(ctx, list)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 threshold, got %d", len(got))
	}
	th := got[0]
	if th.GetType() != "ip" || th.GetRps() != 20 || th.GetMode() != "bot_challenge" {
		t.Errorf("threshold not sent: %+v", th)
	}
	for name, sent := range map[string]bool{
		"cooldown": th.HasCooldown(), "hits": th.HasHits(), "minutes": th.HasMinutes(),
		"value": th.HasValue(), "notify_slack": th.HasNotifySlack(),
	} {
		if sent {
			t.Errorf("%s: null or unknown attributes must not be sent", name)
		}
	}
}

func TestBuildWafThresholds_NullOrUnknownSendsNothing(t *testing.T) {
	ctx := context.Background()
	elem := resource_rule_proxy.ThresholdsValue{}.Type(ctx)
	for _, l := range []types.List{types.ListNull(elem), types.ListUnknown(elem)} {
		got, d := buildWafThresholds(ctx, l)
		if d.HasError() || got != nil {
			t.Errorf("want nil, got %v (%v)", got, d)
		}
	}
}

func TestBuildProxyRequest_SendsThresholdsAndHitsRpm(t *testing.T) {
	ctx := context.Background()
	waf := wafConfigWithBlockListsAndHttpbl(ctx)
	waf.Thresholds = thresholdList(ctx, thresholdValue(ctx, map[string]attr.Value{
		"type": types.StringValue("waf_hit_by_ip"), "hits": types.Int64Value(10), "minutes": types.Int64Value(5),
	}))
	waf.NotifySlackHitsRpm = types.Int64Value(60)
	data := &resource_rule_proxy.RuleProxyModel{WafEnabled: types.BoolValue(true), WafConfig: waf}

	req, d := buildProxyRequest(ctx, data)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	wc := req.GetWafConfig()
	if wc.GetNotifySlackHitsRpm() != 60 {
		t.Errorf("notify_slack_hits_rpm: got %d", wc.GetNotifySlackHitsRpm())
	}
	ths := wc.GetThresholds()
	if len(ths) != 1 || ths[0].GetType() != "waf_hit_by_ip" || ths[0].GetHits() != 10 || ths[0].GetMinutes() != 5 {
		t.Errorf("thresholds: %+v", ths)
	}
}

func TestWafThresholdsFromAPI_AbsentReadsNull(t *testing.T) {
	ctx := context.Background()
	got, d := wafThresholdsFromAPI(ctx, nil, types.ListUnknown(resource_rule_proxy.ThresholdsValue{}.Type(ctx)))
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if !got.IsNull() {
		t.Errorf("want null, got %s", got)
	}
}

func TestWafThresholdsFromAPI_UnconfiguredReadsAll(t *testing.T) {
	ctx := context.Background()
	got, d := wafThresholdsFromAPI(ctx, apiDefaultThresholds(), types.ListUnknown(resource_rule_proxy.ThresholdsValue{}.Type(ctx)))
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	ths := thresholdsOf(t, ctx, got)
	if len(ths) != 3 {
		t.Fatalf("want 3 thresholds, got %d", len(ths))
	}
	hit := ths[2]
	if hit.ThresholdsType.ValueString() != "waf_hit_by_ip" || hit.Hits.ValueInt64() != 10 || hit.Mode.ValueString() != "bot_challenge" {
		t.Errorf("waf_hit_by_ip: %s", hit)
	}
	if !hit.Rps.IsNull() || !hit.Value.IsNull() {
		t.Errorf("absent attributes must read null: rps=%s value=%s", hit.Rps, hit.Value)
	}
	if ths[1].Value.IsNull() || ths[1].Value.ValueString() != "" {
		t.Errorf("header value: %s", ths[1].Value)
	}
}

// TestWafThresholdsFromAPI_RoundTripsConfiguredTypes covers the merge by type:
// the API returns all three types, but state keeps only the configured ones,
// filled with the stored values.
func TestWafThresholdsFromAPI_RoundTripsConfiguredTypes(t *testing.T) {
	ctx := context.Background()
	planned := thresholdList(ctx, thresholdValue(ctx, map[string]attr.Value{
		"type": types.StringValue("ip"), "rps": types.Int64Value(20), "mode": types.StringValue("block"),
		"cooldown": types.Int64Unknown(), "hits": types.Int64Unknown(), "minutes": types.Int64Unknown(),
		"notify_slack": types.StringUnknown(), "value": types.StringUnknown(),
	}))

	got, d := wafThresholdsFromAPI(ctx, apiDefaultThresholds(), planned)
	if d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	ths := thresholdsOf(t, ctx, got)
	if len(ths) != 1 {
		t.Fatalf("want 1 threshold, got %d", len(ths))
	}
	ip := ths[0]
	if ip.Rps.ValueInt64() != 20 || ip.Mode.ValueString() != "block" || ip.Cooldown.ValueInt64() != 30 {
		t.Errorf("ip: %s", ip)
	}
	if !ip.Hits.IsNull() || !ip.Value.IsNull() || ip.NotifySlack.ValueString() != "" {
		t.Errorf("ip computed attributes: %s", ip)
	}

	// A second read with the result as prior state is stable.
	again, _ := wafThresholdsFromAPI(ctx, apiDefaultThresholds(), got)
	if !again.Equal(got) {
		t.Errorf("round trip drift:\n%s\n%s", got, again)
	}
}

func TestWafThresholdsFromAPI_MissingTypeNullsUnknowns(t *testing.T) {
	ctx := context.Background()
	planned := thresholdList(ctx, thresholdValue(ctx, map[string]attr.Value{
		"type": types.StringValue("custom"), "rps": types.Int64Value(3), "cooldown": types.Int64Unknown(),
	}))
	got, _ := wafThresholdsFromAPI(ctx, apiDefaultThresholds(), planned)
	ths := thresholdsOf(t, ctx, got)
	if len(ths) != 1 || ths[0].Rps.ValueInt64() != 3 || !ths[0].Cooldown.IsNull() {
		t.Errorf("unmatched entry: %v", ths)
	}
}

func TestNotifySlackHitsRpmFromAPI(t *testing.T) {
	set := quantadmingo.NewWafConfig()
	set.SetNotifySlackHitsRpm(60)
	cases := []struct {
		name  string
		waf   *quantadmingo.WafConfig
		prior types.Int64
		want  types.Int64
	}{
		{"value", set, types.Int64Unknown(), types.Int64Value(60)},
		{"absent", quantadmingo.NewWafConfig(), types.Int64Unknown(), types.Int64Null()},
		{"absent after value", quantadmingo.NewWafConfig(), types.Int64Value(60), types.Int64Null()},
		{"explicit off kept", quantadmingo.NewWafConfig(), types.Int64Value(0), types.Int64Value(0)},
	}
	for _, tc := range cases {
		if got := notifySlackHitsRpmFromAPI(tc.waf, tc.prior); !got.Equal(tc.want) {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestKeepPriorThresholdsIfUnknown(t *testing.T) {
	ctx := context.Background()
	elem := resource_rule_proxy.ThresholdsValue{}.Type(ctx)
	prior := thresholdList(ctx, thresholdValue(ctx, map[string]attr.Value{"type": types.StringValue("ip")}))
	planned := thresholdList(ctx)

	if got := keepPriorThresholdsIfUnknown(types.ListUnknown(elem), prior); !got.Equal(prior) {
		t.Errorf("unknown plan must keep prior, got %s", got)
	}
	if got := keepPriorThresholdsIfUnknown(planned, prior); !got.Equal(planned) {
		t.Errorf("known plan must win, got %s", got)
	}
}

// TestRuleProxySchema_ThresholdsModeHasNoDefault guards against the nested
// default that the framework applies to every stored threshold, which makes
// every plan after an apply show an update.
func TestRuleProxySchema_ThresholdsModeHasNoDefault(t *testing.T) {
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	(&ruleProxyResource{}).Schema(ctx, resource.SchemaRequest{}, resp)

	waf := resp.Schema.Attributes["waf_config"].(schema.SingleNestedAttribute)
	thresholds := waf.Attributes["thresholds"].(schema.ListNestedAttribute)
	mode := thresholds.NestedObject.Attributes["mode"].(schema.StringAttribute)
	if mode.Default != nil {
		t.Error("waf_config.thresholds[*].mode must not have a Default")
	}
	if !mode.Optional || !mode.Computed {
		t.Error("waf_config.thresholds[*].mode must stay Optional+Computed")
	}
}
