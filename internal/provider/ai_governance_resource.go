package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_governance"
)

var (
	_ resource.Resource                = (*aiGovernanceResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiGovernanceResource)(nil)
	_ resource.ResourceWithImportState = (*aiGovernanceResource)(nil)
)

func NewAiGovernanceResource() resource.Resource {
	return &aiGovernanceResource{}
}

type aiGovernanceResource struct {
	client *client.Client
}

func (r *aiGovernanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_governance"
}

func (r *aiGovernanceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_ai_governance.AiGovernanceResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	// version and success change on every update — must not carry state forward.
	clearPlanModifiers(s.Attributes, "version", "success")
	resp.Schema = s
}

func (r *aiGovernanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *aiGovernanceResource) getOrg(data *resource_ai_governance.AiGovernanceModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

// Create is implemented as a PUT since governance is a singleton resource.
func (r *aiGovernanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_governance.AiGovernanceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callGovernancePutAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiGovernanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_governance.AiGovernanceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callGovernanceReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the governance config was not found, remove from state so Terraform plans recreation.
	if data.Organisation.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiGovernanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ai_governance.AiGovernanceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry forward the version from state for optimistic concurrency.
	var state resource_ai_governance.AiGovernanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Version = state.Version

	resp.Diagnostics.Append(callGovernancePutAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete resets governance to safe defaults (cannot truly delete a singleton).
func (r *aiGovernanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_governance.AiGovernanceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := r.getOrg(&data)

	resetReq := quantadmingo.NewUpdateGovernanceConfigRequest(true, "unrestricted")

	_, httpResp, err := r.client.Instance.AIGovernanceAPI.UpdateGovernanceConfig(r.client.AuthContext, org).
		UpdateGovernanceConfigRequest(*resetReq).Execute()
	if err != nil {
		if httpResp != nil {
			body, _ := io.ReadAll(httpResp.Body)
			resp.Diagnostics.AddError("Unable to reset AI governance",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			resp.Diagnostics.AddError("Unable to reset AI governance", fmt.Sprintf("Error: %s", err.Error()))
		}
	}
}

// ImportState reads the current governance config. No ID is needed since it is
// a singleton scoped to the provider's organization.
func (r *aiGovernanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_ai_governance.AiGovernanceModel

	// The import ID is the organization name (or empty to use provider default).
	if req.ID != "" {
		data.Organisation = types.StringValue(req.ID)
	}

	resp.Diagnostics.Append(callGovernanceReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

func callGovernancePutAPI(ctx context.Context, r *aiGovernanceResource, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq := quantadmingo.NewUpdateGovernanceConfigRequest(
		data.AiEnabled.ValueBool(),
		data.ModelPolicy.ValueString(),
	)

	// model_list
	if !data.ModelList.IsNull() && !data.ModelList.IsUnknown() {
		var models []string
		diags.Append(data.ModelList.ElementsAs(ctx, &models, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetModelList(models)
	}

	// mandatory_guardrail_preset
	if !data.MandatoryGuardrailPreset.IsNull() && !data.MandatoryGuardrailPreset.IsUnknown() {
		sdkReq.SetMandatoryGuardrailPreset(data.MandatoryGuardrailPreset.ValueString())
	}

	// mandatory_filter_policies
	if !data.MandatoryFilterPolicies.IsNull() && !data.MandatoryFilterPolicies.IsUnknown() {
		var policies []string
		diags.Append(data.MandatoryFilterPolicies.ElementsAs(ctx, &policies, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetMandatoryFilterPolicies(policies)
	}

	// spend_limits — SpendLimits is a SpendLimitsValue with fields directly accessible.
	if !data.SpendLimits.IsNull() && !data.SpendLimits.IsUnknown() {
		sl := data.SpendLimits
		sdkSL := quantadmingo.NewGetGovernanceConfig200ResponseSpendLimits()
		if !sl.MonthlyBudgetCents.IsNull() && !sl.MonthlyBudgetCents.IsUnknown() {
			sdkSL.SetMonthlyBudgetCents(int32(sl.MonthlyBudgetCents.ValueInt64()))
		}
		if !sl.DailyBudgetCents.IsNull() && !sl.DailyBudgetCents.IsUnknown() {
			sdkSL.SetDailyBudgetCents(int32(sl.DailyBudgetCents.ValueInt64()))
		}
		if !sl.PerUserMonthlyBudgetCents.IsNull() && !sl.PerUserMonthlyBudgetCents.IsUnknown() {
			sdkSL.SetPerUserMonthlyBudgetCents(int32(sl.PerUserMonthlyBudgetCents.ValueInt64()))
		}
		if !sl.PerUserDailyBudgetCents.IsNull() && !sl.PerUserDailyBudgetCents.IsUnknown() {
			sdkSL.SetPerUserDailyBudgetCents(int32(sl.PerUserDailyBudgetCents.ValueInt64()))
		}
		if !sl.PerTokenMonthlyBudgetCents.IsNull() && !sl.PerTokenMonthlyBudgetCents.IsUnknown() {
			sdkSL.SetPerTokenMonthlyBudgetCents(int32(sl.PerTokenMonthlyBudgetCents.ValueInt64()))
		}
		if !sl.PerTokenDailyBudgetCents.IsNull() && !sl.PerTokenDailyBudgetCents.IsUnknown() {
			sdkSL.SetPerTokenDailyBudgetCents(int32(sl.PerTokenDailyBudgetCents.ValueInt64()))
		}
		if !sl.WarningThresholdPercent.IsNull() && !sl.WarningThresholdPercent.IsUnknown() {
			sdkSL.SetWarningThresholdPercent(int32(sl.WarningThresholdPercent.ValueInt64()))
		}
		if !sl.InterfaceLimits.IsNull() && !sl.InterfaceLimits.IsUnknown() {
			ilSDK, d := interfaceLimitsToSDK(ctx, sl.InterfaceLimits)
			diags.Append(d...)
			sdkSL.SetInterfaceLimits(ilSDK)
		}
		if !sl.UserOverrides.IsNull() && !sl.UserOverrides.IsUnknown() {
			uoSDK, d := userOverridesToSDK(ctx, sl.UserOverrides)
			diags.Append(d...)
			sdkSL.SetUserOverrides(uoSDK)
		}
		if !sl.TokenOverrides.IsNull() && !sl.TokenOverrides.IsUnknown() {
			toSDK, d := tokenOverridesToSDK(ctx, sl.TokenOverrides)
			diags.Append(d...)
			sdkSL.SetTokenOverrides(toSDK)
		}
		sdkReq.SetSpendLimits(*sdkSL)
	}

	// version (optimistic concurrency)
	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		sdkReq.SetVersion(int32(data.Version.ValueInt64()))
	}

	sdkResp, httpResp, err := r.client.Instance.AIGovernanceAPI.UpdateGovernanceConfig(r.client.AuthContext, org).
		UpdateGovernanceConfigRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to update AI governance",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to update AI governance", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Map the typed SDK response directly — no HTTP body re-read needed.
	diags.Append(mapGovernanceConfigFromMap(ctx, sdkResp.GetConfig(), org, data)...)

	// Computed-only fields from the response envelope.
	data.Success = types.BoolValue(sdkResp.GetSuccess())

	return
}

func callGovernanceReadAPI(ctx context.Context, r *aiGovernanceResource, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AIGovernanceAPI.GetGovernanceConfig(r.client.AuthContext, org).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			// Signal "not found" by nulling Organisation — caller handles state removal.
			data.Organisation = types.StringNull()
			return
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read AI governance",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to read AI governance", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Map the typed SDK response to the Terraform model.
	diags.Append(mapGovernanceGetResponse(ctx, sdkResp, org, data)...)
	return
}

// mapGovernanceGetResponse maps the typed GetGovernanceConfig200Response onto
// the Terraform model. This is used for Read and ImportState.
func mapGovernanceGetResponse(ctx context.Context, resp *quantadmingo.GetGovernanceConfig200Response, org string, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	data.Organisation = types.StringValue(org)
	data.AiEnabled = types.BoolValue(resp.GetAiEnabled())
	data.ModelPolicy = types.StringValue(resp.GetModelPolicy())
	data.Version = types.Int64Value(int64(resp.GetVersion()))

	// org_id — from the response or fall back to the org parameter.
	if orgId, ok := resp.GetOrgIdOk(); ok && orgId != nil && *orgId != "" {
		data.OrgId = types.StringValue(*orgId)
	} else {
		data.OrgId = types.StringValue(org)
	}

	// Computed-only response metadata.
	data.Success = types.BoolValue(true) // GET succeeded
	data.Config = resource_ai_governance.NewConfigValueNull()

	// model_list
	if ml := resp.GetModelList(); len(ml) > 0 {
		mlVal, d := types.ListValueFrom(ctx, types.StringType, ml)
		diags.Append(d...)
		data.ModelList = mlVal
	} else {
		data.ModelList = types.ListNull(types.StringType)
	}

	// mandatory_guardrail_preset
	if preset, ok := resp.GetMandatoryGuardrailPresetOk(); ok && preset != nil && *preset != "" {
		data.MandatoryGuardrailPreset = types.StringValue(*preset)
	} else {
		data.MandatoryGuardrailPreset = types.StringNull()
	}

	// mandatory_filter_policies
	if mfp := resp.GetMandatoryFilterPolicies(); len(mfp) > 0 {
		mfpVal, d := types.ListValueFrom(ctx, types.StringType, mfp)
		diags.Append(d...)
		data.MandatoryFilterPolicies = mfpVal
	} else {
		data.MandatoryFilterPolicies = types.ListNull(types.StringType)
	}

	// spend_limits — construct SpendLimitsValue via NewSpendLimitsValue.
	if slPtr, ok := resp.GetSpendLimitsOk(); ok && slPtr != nil {
		ilTF, ild := interfaceLimitsToTF(ctx, slPtr.GetInterfaceLimits())
		diags.Append(ild...)
		uoTF, uod := userOverridesToTF(ctx, slPtr.GetUserOverrides())
		diags.Append(uod...)
		toTF, tod := tokenOverridesToTF(ctx, slPtr.GetTokenOverrides())
		diags.Append(tod...)
		slAttrTypes := resource_ai_governance.SpendLimitsValue{}.AttributeTypes(ctx)
		slAttrs := map[string]attr.Value{
			"monthly_budget_cents":           nullableInt32ToInt64(slPtr.MonthlyBudgetCents),
			"daily_budget_cents":             nullableInt32ToInt64(slPtr.DailyBudgetCents),
			"per_user_monthly_budget_cents":  nullableInt32ToInt64(slPtr.PerUserMonthlyBudgetCents),
			"per_user_daily_budget_cents":    nullableInt32ToInt64(slPtr.PerUserDailyBudgetCents),
			"per_token_monthly_budget_cents": nullableInt32ToInt64(slPtr.PerTokenMonthlyBudgetCents),
			"per_token_daily_budget_cents":   nullableInt32ToInt64(slPtr.PerTokenDailyBudgetCents),
			"warning_threshold_percent":      nullableInt32ToInt64(slPtr.WarningThresholdPercent),
			"interface_limits":               ilTF,
			"user_overrides":                 uoTF,
			"token_overrides":                toTF,
		}
		slVal, d := resource_ai_governance.NewSpendLimitsValue(slAttrTypes, slAttrs)
		diags.Append(d...)
		data.SpendLimits = slVal
	} else {
		data.SpendLimits = resource_ai_governance.NewSpendLimitsValueNull()
	}

	return
}

// nullableInt32ToInt64 converts an SDK NullableInt32 to a Terraform Int64 value.
func nullableInt32ToInt64(n quantadmingo.NullableInt32) types.Int64 {
	if n.IsSet() && n.Get() != nil {
		return types.Int64Value(int64(*n.Get()))
	}
	return types.Int64Null()
}

// mapGovernanceConfigFromMap maps the config map from the UpdateGovernanceConfig
// SDK response onto the Terraform model. The PUT response returns config as
// map[string]interface{}, so we extract fields using helper functions.
func mapGovernanceConfigFromMap(ctx context.Context, configMap map[string]interface{}, org string, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	data.Organisation = types.StringValue(org)

	if v, ok := configMap["aiEnabled"]; ok {
		if b, ok := v.(bool); ok {
			data.AiEnabled = types.BoolValue(b)
		}
	}

	data.ModelPolicy = optionalStringFromConfigMap(configMap, "modelPolicy")
	data.Version = optionalInt64FromMap(configMap, "version")

	// org_id — from the config map or fall back to the org parameter.
	if s := optionalStringFromConfigMap(configMap, "orgId"); !s.IsNull() {
		data.OrgId = s
	} else {
		data.OrgId = types.StringValue(org)
	}

	// Computed-only: config is an opaque empty object, set null.
	data.Config = resource_ai_governance.NewConfigValueNull()

	// model_list
	if v, ok := configMap["modelList"]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			models := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					models = append(models, s)
				}
			}
			ml, d := types.ListValueFrom(ctx, types.StringType, models)
			diags.Append(d...)
			data.ModelList = ml
		} else {
			data.ModelList = types.ListNull(types.StringType)
		}
	} else {
		data.ModelList = types.ListNull(types.StringType)
	}

	// mandatory_guardrail_preset
	if s := optionalStringFromConfigMap(configMap, "mandatoryGuardrailPreset"); !s.IsNull() {
		data.MandatoryGuardrailPreset = s
	} else {
		data.MandatoryGuardrailPreset = types.StringNull()
	}

	// mandatory_filter_policies
	if v, ok := configMap["mandatoryFilterPolicies"]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			policies := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					policies = append(policies, s)
				}
			}
			mfp, d := types.ListValueFrom(ctx, types.StringType, policies)
			diags.Append(d...)
			data.MandatoryFilterPolicies = mfp
		} else {
			data.MandatoryFilterPolicies = types.ListNull(types.StringType)
		}
	} else {
		data.MandatoryFilterPolicies = types.ListNull(types.StringType)
	}

	// spend_limits — construct SpendLimitsValue via NewSpendLimitsValue.
	if v, ok := configMap["spendLimits"]; ok && v != nil {
		if slMap, ok := v.(map[string]interface{}); ok && len(slMap) > 0 {
			ilTF, ild := interfaceLimitsFromRawMap(ctx, slMap["interfaceLimits"])
			diags.Append(ild...)
			uoTF, uod := userOverridesFromRawMap(ctx, slMap["userOverrides"])
			diags.Append(uod...)
			toTF, tod := tokenOverridesFromRawMap(ctx, slMap["tokenOverrides"])
			diags.Append(tod...)
			slAttrTypes := resource_ai_governance.SpendLimitsValue{}.AttributeTypes(ctx)
			slAttrs := map[string]attr.Value{
				"monthly_budget_cents":           optionalInt64FromMap(slMap, "monthlyBudgetCents"),
				"daily_budget_cents":             optionalInt64FromMap(slMap, "dailyBudgetCents"),
				"per_user_monthly_budget_cents":  optionalInt64FromMap(slMap, "perUserMonthlyBudgetCents"),
				"per_user_daily_budget_cents":    optionalInt64FromMap(slMap, "perUserDailyBudgetCents"),
				"per_token_monthly_budget_cents": optionalInt64FromMap(slMap, "perTokenMonthlyBudgetCents"),
				"per_token_daily_budget_cents":   optionalInt64FromMap(slMap, "perTokenDailyBudgetCents"),
				"warning_threshold_percent":      optionalInt64FromMap(slMap, "warningThresholdPercent"),
				"interface_limits":               ilTF,
				"user_overrides":                 uoTF,
				"token_overrides":                toTF,
			}
			slVal, d := resource_ai_governance.NewSpendLimitsValue(slAttrTypes, slAttrs)
			diags.Append(d...)
			data.SpendLimits = slVal
		} else {
			data.SpendLimits = resource_ai_governance.NewSpendLimitsValueNull()
		}
	} else {
		data.SpendLimits = resource_ai_governance.NewSpendLimitsValueNull()
	}

	return
}

// optionalStringFromConfigMap extracts a string from a map[string]interface{},
// returning types.StringNull() if absent or empty.
func optionalStringFromConfigMap(m map[string]interface{}, key string) types.String {
	v, ok := m[key]
	if !ok || v == nil {
		return types.StringNull()
	}
	if s, ok := v.(string); ok && s != "" {
		return types.StringValue(s)
	}
	return types.StringNull()
}

// optionalInt64FromMap extracts an int64 value from a JSON-decoded map.
// JSON numbers decode as float64; this handles the conversion.
func optionalInt64FromMap(m map[string]interface{}, key string) types.Int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return types.Int64Null()
	}
	switch n := v.(type) {
	case float64:
		return types.Int64Value(int64(n))
	case int64:
		return types.Int64Value(n)
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return types.Int64Null()
		}
		return types.Int64Value(i)
	default:
		return types.Int64Null()
	}
}
