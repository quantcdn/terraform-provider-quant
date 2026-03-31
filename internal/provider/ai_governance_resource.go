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
	resetBody := map[string]interface{}{
		"aiEnabled":   true,
		"modelPolicy": "unrestricted",
	}

	apiResp, err := doAIRequest(r.client, http.MethodPut,
		fmt.Sprintf("/api/v3/organisations/%s/ai/governance", org), resetBody)
	if err != nil {
		resp.Diagnostics.AddError("Unable to reset AI governance", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		body, _ := io.ReadAll(apiResp.Body)
		resp.Diagnostics.AddError("Unable to reset AI governance",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(body)))
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

	body := map[string]interface{}{
		"aiEnabled":   data.AiEnabled.ValueBool(),
		"modelPolicy": data.ModelPolicy.ValueString(),
	}

	// model_list
	if !data.ModelList.IsNull() && !data.ModelList.IsUnknown() {
		var models []string
		diags.Append(data.ModelList.ElementsAs(ctx, &models, false)...)
		if diags.HasError() {
			return
		}
		body["modelList"] = models
	}

	// mandatory_guardrail_preset
	if !data.MandatoryGuardrailPreset.IsNull() && !data.MandatoryGuardrailPreset.IsUnknown() {
		body["mandatoryGuardrailPreset"] = data.MandatoryGuardrailPreset.ValueString()
	}

	// mandatory_filter_policies
	if !data.MandatoryFilterPolicies.IsNull() && !data.MandatoryFilterPolicies.IsUnknown() {
		var policies []string
		diags.Append(data.MandatoryFilterPolicies.ElementsAs(ctx, &policies, false)...)
		if diags.HasError() {
			return
		}
		body["mandatoryFilterPolicies"] = policies
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
		body["spendLimits"] = slMap
	}

	// version (optimistic concurrency)
	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		body["version"] = data.Version.ValueInt64()
	}

	apiResp, err := doAIRequest(r.client, http.MethodPut,
		fmt.Sprintf("/api/v3/organisations/%s/ai/governance", org), body)
	if err != nil {
		diags.AddError("Unable to update AI governance", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to update AI governance",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	// Parse response to get updated version and server-side defaults.
	diags.Append(parseGovernanceResponse(ctx, apiResp, org, data)...)
	return
}

func callGovernanceReadAPI(ctx context.Context, r *aiGovernanceResource, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	apiResp, err := doAIRequest(r.client, http.MethodGet,
		fmt.Sprintf("/api/v3/organisations/%s/ai/governance", org), nil)
	if err != nil {
		diags.AddError("Unable to read AI governance", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == http.StatusNotFound {
		diags.AddError("AI governance not found",
			fmt.Sprintf("No AI governance configuration found for organization '%s'.", org))
		return
	}

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to read AI governance",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseGovernanceResponse(ctx, apiResp, org, data)...)
	return
}

// parseGovernanceResponse reads the JSON response from the governance API and
// maps it back onto the Terraform model.
//
// Expected response shape:
//
//	{
//	  "config": {
//	    "aiEnabled": true,
//	    "modelPolicy": "unrestricted",
//	    "modelList": ["gpt-4"],
//	    "mandatoryGuardrailPreset": "strict",
//	    "mandatoryFilterPolicies": ["toxicity"],
//	    "spendLimits": { ... },
//	    "version": 3
//	  }
//	}
func parseGovernanceResponse(ctx context.Context, apiResp *http.Response, org string, data *resource_ai_governance.AiGovernanceModel) (diags diag.Diagnostics) {
	respBody, err := io.ReadAll(apiResp.Body)
	if err != nil {
		diags.AddError("Unable to read AI governance response", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	var envelope struct {
		Config struct {
			AiEnabled                bool                   `json:"aiEnabled"`
			ModelPolicy              string                 `json:"modelPolicy"`
			ModelList                []string               `json:"modelList"`
			MandatoryGuardrailPreset *string                `json:"mandatoryGuardrailPreset"`
			MandatoryFilterPolicies  []string               `json:"mandatoryFilterPolicies"`
			SpendLimits              map[string]interface{} `json:"spendLimits"`
			Version                  int64                  `json:"version"`
		} `json:"config"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		diags.AddError("Unable to parse AI governance response",
			fmt.Sprintf("Error: %s\nBody: %s", err.Error(), string(respBody)))
		return
	}

	cfg := envelope.Config

	data.Organization = types.StringValue(org)
	data.AiEnabled = types.BoolValue(cfg.AiEnabled)
	data.ModelPolicy = types.StringValue(cfg.ModelPolicy)
	data.Version = types.Int64Value(cfg.Version)

	// model_list
	if len(cfg.ModelList) > 0 {
		ml, d := types.ListValueFrom(ctx, types.StringType, cfg.ModelList)
		diags.Append(d...)
		data.ModelList = ml
	} else {
		data.ModelList = types.ListNull(types.StringType)
	}

	// mandatory_guardrail_preset
	if cfg.MandatoryGuardrailPreset != nil && *cfg.MandatoryGuardrailPreset != "" {
		data.MandatoryGuardrailPreset = types.StringValue(*cfg.MandatoryGuardrailPreset)
	} else {
		data.MandatoryGuardrailPreset = types.StringNull()
	}

	// mandatory_filter_policies
	if len(cfg.MandatoryFilterPolicies) > 0 {
		mfp, d := types.ListValueFrom(ctx, types.StringType, cfg.MandatoryFilterPolicies)
		diags.Append(d...)
		data.MandatoryFilterPolicies = mfp
	} else {
		data.MandatoryFilterPolicies = types.ListNull(types.StringType)
	}

	// spend_limits
	if cfg.SpendLimits != nil && len(cfg.SpendLimits) > 0 {
		sl := resource_ai_governance.SpendLimitsModel{
			MonthlyBudgetCents:        optionalInt64FromMap(cfg.SpendLimits, "monthlyBudgetCents"),
			DailyBudgetCents:          optionalInt64FromMap(cfg.SpendLimits, "dailyBudgetCents"),
			PerUserMonthlyBudgetCents: optionalInt64FromMap(cfg.SpendLimits, "perUserMonthlyBudgetCents"),
			PerUserDailyBudgetCents:   optionalInt64FromMap(cfg.SpendLimits, "perUserDailyBudgetCents"),
			WarningThresholdPercent:   optionalInt64FromMap(cfg.SpendLimits, "warningThresholdPercent"),
		}
		objVal, d := types.ObjectValueFrom(ctx, resource_ai_governance.SpendLimitsAttrTypes(), sl)
		diags.Append(d...)
		data.SpendLimits = objVal
	} else {
		data.SpendLimits = types.ObjectNull(resource_ai_governance.SpendLimitsAttrTypes())
	}

	return
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
