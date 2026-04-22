package provider

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent_overlay"
)

var (
	_ resource.Resource                = (*aiAgentOverlayResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiAgentOverlayResource)(nil)
	_ resource.ResourceWithImportState = (*aiAgentOverlayResource)(nil)
)

func NewAiAgentOverlayResource() resource.Resource {
	return &aiAgentOverlayResource{}
}

type aiAgentOverlayResource struct {
	client *client.Client
}

func (r *aiAgentOverlayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_agent_overlay"
}

func (r *aiAgentOverlayResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_ai_agent_overlay.AiAgentOverlayResourceSchema(ctx)
}

func (r *aiAgentOverlayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aiAgentOverlayResource) getOrg(data *resource_ai_agent_overlay.AiAgentOverlayModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *aiAgentOverlayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_agent_overlay.AiAgentOverlayModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.upsert(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAgentOverlayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_agent_overlay.AiAgentOverlayModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readDiags := r.read(ctx, &data)
	// If the overlay was not found, remove from state and return without error.
	for _, d := range readDiags {
		if d.Severity() == diag.SeverityError && d.Summary() == "Overlay not found" {
			resp.State.RemoveResource(ctx)
			return
		}
	}
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAgentOverlayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ai_agent_overlay.AiAgentOverlayModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry forward the prior version from state for optimistic concurrency.
	var state resource_ai_agent_overlay.AiAgentOverlayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Version = state.Version

	resp.Diagnostics.Append(r.upsert(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAgentOverlayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_agent_overlay.AiAgentOverlayModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org := r.getOrg(&data)
	agentId := data.AgentId.ValueString()
	if agentId == "" {
		resp.Diagnostics.AddError("Missing agent_id", "agent_id is required to delete an AI agent overlay.")
		return
	}

	_, httpResp, err := r.client.Instance.AIAgentsAPI.DeleteAgentOverlay(r.client.AuthContext, org, agentId).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		if httpResp != nil {
			body, _ := io.ReadAll(httpResp.Body)
			resp.Diagnostics.AddError("Unable to delete AI agent overlay",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			resp.Diagnostics.AddError("Unable to delete AI agent overlay", fmt.Sprintf("Error: %s", err.Error()))
		}
	}
}

func (r *aiAgentOverlayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form '{organisation}/{agent_id}', got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organisation"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("agent_id"), parts[1])...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

// upsert builds an UpsertAgentOverlayRequest from the model and calls the API.
// Version is passed through for optimistic concurrency when present.
func (r *aiAgentOverlayResource) upsert(ctx context.Context, data *resource_ai_agent_overlay.AiAgentOverlayModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	if data.AgentId.IsNull() || data.AgentId.IsUnknown() || data.AgentId.ValueString() == "" {
		diags.AddError("Missing agent_id", "agent_id is required to upsert an AI agent overlay.")
		return
	}
	agentId := data.AgentId.ValueString()

	sdkReq := quantadmingo.NewUpsertAgentOverlayRequest()

	// model_id
	if !data.ModelId.IsNull() && !data.ModelId.IsUnknown() {
		sdkReq.SetModelId(data.ModelId.ValueString())
	}

	// temperature — types.Number -> *float32
	if !data.Temperature.IsNull() && !data.Temperature.IsUnknown() {
		bf := data.Temperature.ValueBigFloat()
		if bf != nil {
			f32, _ := bf.Float32()
			sdkReq.SetTemperature(f32)
		}
	}

	// max_tokens
	if !data.MaxTokens.IsNull() && !data.MaxTokens.IsUnknown() {
		sdkReq.SetMaxTokens(int32(data.MaxTokens.ValueInt64()))
	}

	// disabled_skills
	if !data.DisabledSkills.IsNull() && !data.DisabledSkills.IsUnknown() {
		var vals []string
		diags.Append(data.DisabledSkills.ElementsAs(ctx, &vals, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetDisabledSkills(vals)
	}

	// additional_skills
	if !data.AdditionalSkills.IsNull() && !data.AdditionalSkills.IsUnknown() {
		var vals []string
		diags.Append(data.AdditionalSkills.ElementsAs(ctx, &vals, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAdditionalSkills(vals)
	}

	// additional_tools
	if !data.AdditionalTools.IsNull() && !data.AdditionalTools.IsUnknown() {
		var vals []string
		diags.Append(data.AdditionalTools.ElementsAs(ctx, &vals, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAdditionalTools(vals)
	}

	// disabled_tools
	if !data.DisabledTools.IsNull() && !data.DisabledTools.IsUnknown() {
		var vals []string
		diags.Append(data.DisabledTools.ElementsAs(ctx, &vals, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetDisabledTools(vals)
	}

	// system_prompt_append
	if !data.SystemPromptAppend.IsNull() && !data.SystemPromptAppend.IsUnknown() {
		sdkReq.SetSystemPromptAppend(data.SystemPromptAppend.ValueString())
	}

	// allowed_collections
	if !data.AllowedCollections.IsNull() && !data.AllowedCollections.IsUnknown() {
		var vals []string
		diags.Append(data.AllowedCollections.ElementsAs(ctx, &vals, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedCollections(vals)
	}

	// guardrail_preset
	if !data.GuardrailPreset.IsNull() && !data.GuardrailPreset.IsUnknown() {
		sdkReq.SetGuardrailPreset(data.GuardrailPreset.ValueString())
	}

	// version — only pass when non-null and > 0 (compare-and-swap).
	if !data.Version.IsNull() && !data.Version.IsUnknown() && data.Version.ValueInt64() > 0 {
		sdkReq.SetVersion(int32(data.Version.ValueInt64()))
	}

	_, httpResp, err := r.client.Instance.AIAgentsAPI.UpsertAgentOverlay(r.client.AuthContext, org, agentId).
		UpsertAgentOverlayRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusConflict {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Overlay version conflict",
				fmt.Sprintf("The overlay was modified concurrently (HTTP 409). Refresh state (terraform refresh) and retry. Response: %s", string(body)))
			return
		}
		if httpResp != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to upsert AI agent overlay",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to upsert AI agent overlay", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	return
}

// read fetches the overlay and populates the model. On 404 it returns a diag
// with summary "Overlay not found" so the caller can handle state removal.
func (r *aiAgentOverlayResource) read(ctx context.Context, data *resource_ai_agent_overlay.AiAgentOverlayModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	if data.AgentId.IsNull() || data.AgentId.IsUnknown() || data.AgentId.ValueString() == "" {
		diags.AddError("Missing agent_id", "agent_id is required to read an AI agent overlay.")
		return
	}
	agentId := data.AgentId.ValueString()

	apiResp, httpResp, err := r.client.Instance.AIAgentsAPI.GetAgentOverlay(r.client.AuthContext, org, agentId).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			diags.AddError("Overlay not found",
				fmt.Sprintf("No overlay exists for agent %q in organisation %q.", agentId, org))
			return
		}
		if httpResp != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read AI agent overlay",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to read AI agent overlay", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Organisation & agent_id are known now.
	data.Organisation = types.StringValue(org)
	data.AgentId = types.StringValue(agentId)

	// Populate Base (computed nested struct) from the base-agent metadata.
	diags.Append(populateBase(ctx, apiResp.Base, data)...)
	if diags.HasError() {
		return
	}

	// Populate Overlay (computed nested struct) and top-level overlay fields.
	overlay := apiResp.Overlay.Get()
	diags.Append(populateOverlay(ctx, overlay, data)...)

	return
}

// populateBase sets data.Base from the SDK base response.
func populateBase(ctx context.Context, base *quantadmingo.GetAgentOverlay200ResponseBase, data *resource_ai_agent_overlay.AiAgentOverlayModel) (diags diag.Diagnostics) {
	if base == nil {
		data.Base = resource_ai_agent_overlay.NewBaseValueNull()
		return
	}

	attrs := map[string]attr.Value{
		"agent_id":           stringValueOrEmpty(base.AgentId),
		"allowed_tools":      stringListFromSlice(ctx, base.AllowedTools),
		"assigned_skill_ids": stringListFromSlice(ctx, base.AssignedSkillIds),
		"model_id":           stringValueOrEmpty(base.ModelId),
		"name":               stringValueOrEmpty(base.Name),
	}

	baseV, d := resource_ai_agent_overlay.NewBaseValue(
		resource_ai_agent_overlay.BaseValue{}.AttributeTypes(ctx),
		attrs,
	)
	diags.Append(d...)
	data.Base = baseV
	return
}

// populateOverlay sets both the top-level overlay fields and the computed
// data.Overlay nested struct from the SDK overlay response.
func populateOverlay(ctx context.Context, overlay *quantadmingo.GetAgentOverlay200ResponseOverlay, data *resource_ai_agent_overlay.AiAgentOverlayModel) (diags diag.Diagnostics) {
	if overlay == nil {
		// No overlay — null the top-level overlay fields and the nested Overlay.
		data.ModelId = types.StringNull()
		data.Temperature = types.NumberNull()
		data.MaxTokens = types.Int64Null()
		data.DisabledSkills = types.ListNull(types.StringType)
		data.AdditionalSkills = types.ListNull(types.StringType)
		data.AdditionalTools = types.ListNull(types.StringType)
		data.DisabledTools = types.ListNull(types.StringType)
		data.SystemPromptAppend = types.StringNull()
		data.AllowedCollections = types.ListNull(types.StringType)
		data.GuardrailPreset = types.StringNull()
		data.Version = types.Int64Null()
		data.Overlay = resource_ai_agent_overlay.NewOverlayValueNull()
		return
	}

	// model_id
	if overlay.ModelId != nil {
		data.ModelId = types.StringValue(*overlay.ModelId)
	} else {
		data.ModelId = types.StringNull()
	}

	// temperature — *float32 -> types.Number
	if overlay.Temperature != nil {
		rounded := math.Round(float64(*overlay.Temperature)*100) / 100
		data.Temperature = types.NumberValue(big.NewFloat(rounded))
	} else {
		data.Temperature = types.NumberNull()
	}

	// max_tokens
	if overlay.MaxTokens != nil {
		data.MaxTokens = types.Int64Value(int64(*overlay.MaxTokens))
	} else {
		data.MaxTokens = types.Int64Null()
	}

	data.DisabledSkills = stringListFromSlice(ctx, overlay.DisabledSkills)
	data.AdditionalSkills = stringListFromSlice(ctx, overlay.AdditionalSkills)
	data.AdditionalTools = stringListFromSlice(ctx, overlay.AdditionalTools)
	data.DisabledTools = stringListFromSlice(ctx, overlay.DisabledTools)

	// system_prompt_append
	if overlay.SystemPromptAppend != nil {
		data.SystemPromptAppend = types.StringValue(*overlay.SystemPromptAppend)
	} else {
		data.SystemPromptAppend = types.StringNull()
	}

	data.AllowedCollections = stringListFromSlice(ctx, overlay.AllowedCollections)

	// guardrail_preset
	if overlay.GuardrailPreset != nil {
		data.GuardrailPreset = types.StringValue(*overlay.GuardrailPreset)
	} else {
		data.GuardrailPreset = types.StringNull()
	}

	// version
	if overlay.Version != nil {
		data.Version = types.Int64Value(int64(*overlay.Version))
	} else {
		data.Version = types.Int64Null()
	}

	if diags.HasError() {
		return
	}

	// Build the computed nested Overlay value mirroring the top-level fields.
	overlayAttrs := map[string]attr.Value{
		"additional_skills":    data.AdditionalSkills,
		"additional_tools":     data.AdditionalTools,
		"allowed_collections":  data.AllowedCollections,
		"disabled_skills":      data.DisabledSkills,
		"disabled_tools":       data.DisabledTools,
		"guardrail_preset":     data.GuardrailPreset,
		"max_tokens":           data.MaxTokens,
		"model_id":             data.ModelId,
		"system_prompt_append": data.SystemPromptAppend,
		"temperature":          data.Temperature,
		"version":              data.Version,
	}

	overlayV, d := resource_ai_agent_overlay.NewOverlayValue(
		resource_ai_agent_overlay.OverlayValue{}.AttributeTypes(ctx),
		overlayAttrs,
	)
	diags.Append(d...)
	data.Overlay = overlayV
	return
}

// ---------------------------------------------------------------------------
// small helpers
// ---------------------------------------------------------------------------

func stringValueOrEmpty(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}
