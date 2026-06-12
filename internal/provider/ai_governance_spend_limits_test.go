package provider

import (
	"context"
	"testing"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

// TestInterfaceLimitsRoundTrip exercises SDK -> Terraform -> SDK for the
// spend_limits.interface_limits map, including ElementsAs into the generated
// InterfaceLimitsValue and null handling for an unset field.
func TestInterfaceLimitsRoundTrip(t *testing.T) {
	ctx := context.Background()

	sdkIn := map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue{}
	slack := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue()
	slack.SetDailyCents(5000)
	slack.SetMonthlyCents(100000)
	sdkIn["slack"] = *slack
	auto := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimitsInterfaceLimitsValue()
	auto.SetMonthlyCents(250000) // daily intentionally unset
	sdkIn["autonomous"] = *auto

	tfMap, diags := interfaceLimitsToTF(ctx, sdkIn)
	if diags.HasError() {
		t.Fatalf("interfaceLimitsToTF: %v", diags)
	}
	if tfMap.IsNull() || len(tfMap.Elements()) != 2 {
		t.Fatalf("expected 2 elements, got null=%v len=%d", tfMap.IsNull(), len(tfMap.Elements()))
	}

	sdkOut, diags := interfaceLimitsToSDK(ctx, tfMap)
	if diags.HasError() {
		t.Fatalf("interfaceLimitsToSDK: %v", diags)
	}
	if len(sdkOut) != 2 {
		t.Fatalf("expected 2 SDK entries, got %d", len(sdkOut))
	}
	slackOut := sdkOut["slack"]
	if got := slackOut.GetDailyCents(); got != 5000 {
		t.Errorf("slack daily_cents: want 5000, got %d", got)
	}
	if got := slackOut.GetMonthlyCents(); got != 100000 {
		t.Errorf("slack monthly_cents: want 100000, got %d", got)
	}
	autoOut := sdkOut["autonomous"]
	if got := autoOut.GetMonthlyCents(); got != 250000 {
		t.Errorf("autonomous monthly_cents: want 250000, got %d", got)
	}
	if _, ok := autoOut.GetDailyCentsOk(); ok {
		t.Error("autonomous daily_cents should remain unset")
	}
}

// TestUserOverridesRoundTrip exercises SDK -> Terraform -> SDK for the
// spend_limits.user_overrides map, including the unlimited bool.
func TestUserOverridesRoundTrip(t *testing.T) {
	ctx := context.Background()

	sdkIn := map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsUserOverridesValue{}
	capped := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimitsUserOverridesValue()
	capped.SetDailyCents(20000)
	capped.SetMonthlyCents(400000)
	sdkIn["1234"] = *capped
	unlimited := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimitsUserOverridesValue()
	unlimited.SetUnlimited(true)
	sdkIn["5678"] = *unlimited

	tfMap, diags := userOverridesToTF(ctx, sdkIn)
	if diags.HasError() {
		t.Fatalf("userOverridesToTF: %v", diags)
	}
	if len(tfMap.Elements()) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(tfMap.Elements()))
	}

	sdkOut, diags := userOverridesToSDK(ctx, tfMap)
	if diags.HasError() {
		t.Fatalf("userOverridesToSDK: %v", diags)
	}
	cappedOut := sdkOut["1234"]
	if got := cappedOut.GetDailyCents(); got != 20000 {
		t.Errorf("user 1234 daily_cents: want 20000, got %d", got)
	}
	if got := cappedOut.GetMonthlyCents(); got != 400000 {
		t.Errorf("user 1234 monthly_cents: want 400000, got %d", got)
	}
	unlimitedOut := sdkOut["5678"]
	if !unlimitedOut.GetUnlimited() {
		t.Error("user 5678 unlimited: want true")
	}
	if _, ok := unlimitedOut.GetDailyCentsOk(); ok {
		t.Error("user 5678 daily_cents should remain unset")
	}
}

// TestInterfaceLimitsFromRawMap covers the PUT-response path, where spend limits
// arrive as a raw map[string]interface{} with JSON-number (float64) cents.
func TestInterfaceLimitsFromRawMap(t *testing.T) {
	ctx := context.Background()
	raw := map[string]interface{}{
		"slack": map[string]interface{}{
			"dailyCents":   float64(5000),
			"monthlyCents": float64(100000),
		},
	}
	tfMap, diags := interfaceLimitsFromRawMap(ctx, raw)
	if diags.HasError() {
		t.Fatalf("interfaceLimitsFromRawMap: %v", diags)
	}
	if len(tfMap.Elements()) != 1 {
		t.Fatalf("expected 1 element, got %d", len(tfMap.Elements()))
	}
	sdkOut, diags := interfaceLimitsToSDK(ctx, tfMap)
	if diags.HasError() {
		t.Fatalf("interfaceLimitsToSDK: %v", diags)
	}
	slackOut := sdkOut["slack"]
	if got := slackOut.GetDailyCents(); got != 5000 {
		t.Errorf("slack daily_cents: want 5000, got %d", got)
	}
	if got := slackOut.GetMonthlyCents(); got != 100000 {
		t.Errorf("slack monthly_cents: want 100000, got %d", got)
	}
}

// TestUserOverridesFromRawMap covers the PUT-response path for user overrides,
// including the unlimited bool.
func TestUserOverridesFromRawMap(t *testing.T) {
	ctx := context.Background()
	raw := map[string]interface{}{
		"5678": map[string]interface{}{
			"unlimited": true,
		},
	}
	tfMap, diags := userOverridesFromRawMap(ctx, raw)
	if diags.HasError() {
		t.Fatalf("userOverridesFromRawMap: %v", diags)
	}
	sdkOut, diags := userOverridesToSDK(ctx, tfMap)
	if diags.HasError() {
		t.Fatalf("userOverridesToSDK: %v", diags)
	}
	unlimitedOut := sdkOut["5678"]
	if !unlimitedOut.GetUnlimited() {
		t.Error("user 5678 unlimited: want true")
	}
}

// TestSpendLimitsMapsNullWhenEmpty ensures empty/absent maps map to a null
// Terraform map (not an empty one), so plans stay stable.
func TestSpendLimitsMapsNullWhenEmpty(t *testing.T) {
	ctx := context.Background()
	il, diags := interfaceLimitsToTF(ctx, nil)
	if diags.HasError() {
		t.Fatalf("interfaceLimitsToTF(nil): %v", diags)
	}
	if !il.IsNull() {
		t.Error("expected null interface_limits for empty input")
	}
	uo, diags := userOverridesToTF(ctx, map[string]quantadmingo.GetGovernanceConfig200ResponseSpendLimitsUserOverridesValue{})
	if diags.HasError() {
		t.Fatalf("userOverridesToTF(empty): %v", diags)
	}
	if !uo.IsNull() {
		t.Error("expected null user_overrides for empty input")
	}
	if _, diags := interfaceLimitsFromRawMap(ctx, nil); diags.HasError() {
		t.Fatalf("interfaceLimitsFromRawMap(nil): %v", diags)
	}
}
