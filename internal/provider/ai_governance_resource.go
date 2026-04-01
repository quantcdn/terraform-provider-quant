package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	resp.Schema = resource_ai_governance.AiGovernanceResourceSchema(ctx)
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
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
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
	if data.Organization.IsNull() {
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
		data.Organization = types.StringValue(req.ID)
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

	// spend_limits
	if !data.SpendLimits.IsNull() && !data.SpendLimits.IsUnknown() {
		var sl resource_ai_governance.SpendLimitsModel
		diags.Append(data.SpendLimits.As(ctx, &sl, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}
		slMap := map[string]interface{}{}
		if !sl.MonthlyBudgetCents.IsNull() && !sl.MonthlyBudgetCents.IsUnknown() {
			slMap["monthlyBudgetCents"] = sl.MonthlyBudgetCents.ValueInt64()
		}
		if !sl.DailyBudgetCents.IsNull() && !sl.DailyBudgetCents.IsUnknown() {
			slMap["dailyBudgetCents"] = sl.DailyBudgetCents.ValueInt64()
		}
		if !sl.PerUserMonthlyBudgetCents.IsNull() && !sl.PerUserMonthlyBudgetCents.IsUnknown() {
			slMap["perUserMonthlyBudgetCents"] = sl.PerUserMonthlyBudgetCents.ValueInt64()
		}
		if !sl.PerUserDailyBudgetCents.IsNull() && !sl.PerUserDailyBudgetCents.IsUnknown() {
			slMap["perUserDailyBudgetCents"] = sl.PerUserDailyBudgetCents.ValueInt64()
		}
		if !sl.WarningThresholdPercent.IsNull() && !sl.WarningThresholdPercent.IsUnknown() {
			slMap["warningThresholdPercent"] = sl.WarningThresholdPercent.ValueInt64()
		}
		sdkReq.SetSpendLimits(slMap)
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
	return
}

func callGovernanceReadAPI(ctx context.Context, r *aiGovernanceResource, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AIGovernanceAPI.GetGovernanceConfig(r.client.AuthContext, org).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			// Signal "not found" by nulling Organization — caller handles state removal.
			data.Organization = types.StringNull()
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
	data.Organization = types.StringValue(org)
	data.AiEnabled = types.BoolValue(resp.GetAiEnabled())
	data.ModelPolicy = types.StringValue(resp.GetModelPolicy())
	data.Version = types.Int64Value(int64(resp.GetVersion()))

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

	// spend_limits
	if slPtr, ok := resp.GetSpendLimitsOk(); ok && slPtr != nil {
		sl := resource_ai_governance.SpendLimitsModel{
			MonthlyBudgetCents:        nullableInt32ToInt64(slPtr.MonthlyBudgetCents),
			DailyBudgetCents:          nullableInt32ToInt64(slPtr.DailyBudgetCents),
			PerUserMonthlyBudgetCents: nullableInt32ToInt64(slPtr.PerUserMonthlyBudgetCents),
			PerUserDailyBudgetCents:   nullableInt32ToInt64(slPtr.PerUserDailyBudgetCents),
			WarningThresholdPercent:   nullableInt32ToInt64(slPtr.WarningThresholdPercent),
		}
		objVal, d := types.ObjectValueFrom(ctx, resource_ai_governance.SpendLimitsAttrTypes(), sl)
		diags.Append(d...)
		data.SpendLimits = objVal
	} else {
		data.SpendLimits = types.ObjectNull(resource_ai_governance.SpendLimitsAttrTypes())
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
	data.Organization = types.StringValue(org)

	if v, ok := configMap["aiEnabled"]; ok {
		if b, ok := v.(bool); ok {
			data.AiEnabled = types.BoolValue(b)
		}
	}

	data.ModelPolicy = optionalStringFromConfigMap(configMap, "modelPolicy")
	data.Version = optionalInt64FromMap(configMap, "version")

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

	// spend_limits
	if v, ok := configMap["spendLimits"]; ok && v != nil {
		if slMap, ok := v.(map[string]interface{}); ok && len(slMap) > 0 {
			sl := resource_ai_governance.SpendLimitsModel{
				MonthlyBudgetCents:        optionalInt64FromMap(slMap, "monthlyBudgetCents"),
				DailyBudgetCents:          optionalInt64FromMap(slMap, "dailyBudgetCents"),
				PerUserMonthlyBudgetCents: optionalInt64FromMap(slMap, "perUserMonthlyBudgetCents"),
				PerUserDailyBudgetCents:   optionalInt64FromMap(slMap, "perUserDailyBudgetCents"),
				WarningThresholdPercent:   optionalInt64FromMap(slMap, "warningThresholdPercent"),
			}
			objVal, d := types.ObjectValueFrom(ctx, resource_ai_governance.SpendLimitsAttrTypes(), sl)
			diags.Append(d...)
			data.SpendLimits = objVal
		} else {
			data.SpendLimits = types.ObjectNull(resource_ai_governance.SpendLimitsAttrTypes())
		}
	} else {
		data.SpendLimits = types.ObjectNull(resource_ai_governance.SpendLimitsAttrTypes())
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
