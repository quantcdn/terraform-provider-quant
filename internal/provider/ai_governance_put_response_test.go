package provider

import (
	"context"
	"testing"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_governance"
)

// The portal answers a governance PUT with {"success":true,"config":{"config":{...}}}.
// The state after apply must hold the inner governance object, not nulls.
func TestGovernanceConfigOfNestedAnswer(t *testing.T) {
	ctx := context.Background()
	nested := map[string]interface{}{
		"config": map[string]interface{}{
			"aiEnabled":   true,
			"modelPolicy": "allowlist",
			"modelList":   []interface{}{"anthropic.claude-sonnet-5-5"},
			"version":     float64(7),
			"spendLimits": map[string]interface{}{
				"monthlyBudgetCents": float64(580000),
				"userOverrides": map[string]interface{}{
					"1040": map[string]interface{}{"monthlyCents": float64(60000)},
				},
			},
		},
	}
	var data resource_ai_governance.AiGovernanceModel
	diags := mapGovernanceConfigFromMap(ctx, governanceConfigOf(nested), "salsa-digital", &data)
	if diags.HasError() {
		t.Fatalf("map: %v", diags)
	}
	if data.ModelPolicy.ValueString() != "allowlist" {
		t.Errorf("model_policy = %q", data.ModelPolicy.ValueString())
	}
	if data.Version.ValueInt64() != 7 {
		t.Errorf("version = %d", data.Version.ValueInt64())
	}
	if data.ModelList.IsNull() || len(data.ModelList.Elements()) != 1 {
		t.Errorf("model_list = %v", data.ModelList)
	}
	if data.SpendLimits.IsNull() {
		t.Fatal("spend_limits is null")
	}
	if data.SpendLimits.MonthlyBudgetCents.ValueInt64() != 580000 {
		t.Errorf("monthly_budget_cents = %v", data.SpendLimits.MonthlyBudgetCents)
	}
	if data.SpendLimits.UserOverrides.IsNull() || len(data.SpendLimits.UserOverrides.Elements()) != 1 {
		t.Errorf("user_overrides = %v", data.SpendLimits.UserOverrides)
	}
}

// A flat answer (the documented shape) maps the same way.
func TestGovernanceConfigOfFlatAnswer(t *testing.T) {
	flat := map[string]interface{}{"aiEnabled": true, "modelPolicy": "unrestricted"}
	if got := governanceConfigOf(flat); got["modelPolicy"] != "unrestricted" {
		t.Errorf("flat answer changed: %v", got)
	}
}
