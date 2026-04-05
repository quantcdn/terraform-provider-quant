package provider

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent"
)

var (
	_ resource.Resource                = (*aiAgentResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiAgentResource)(nil)
	_ resource.ResourceWithImportState = (*aiAgentResource)(nil)
)

func NewAiAgentResource() resource.Resource {
	return &aiAgentResource{}
}

type aiAgentResource struct {
	client *client.Client
}

func (r *aiAgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_agent"
}

func (r *aiAgentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_ai_agent.AiAgentResourceSchema(ctx)
}

func (r *aiAgentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aiAgentResource) getOrg(data *resource_ai_agent.AiAgentModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

func (r *aiAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_agent.AiAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callAiAgentCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_agent.AiAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callAiAgentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the agent was not found, remove from state so TF plans recreation.
	if data.AgentId.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ai_agent.AiAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve AgentId (server-generated) from state.
	var state resource_ai_agent.AiAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AgentId = state.AgentId

	resp.Diagnostics.Append(callAiAgentUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_agent.AiAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callAiAgentDeleteAPI(ctx, r, &data)...)
}

// ImportState imports an agent by its ID.
func (r *aiAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_ai_agent.AiAgentModel
	data.AgentId = types.StringValue(req.ID)

	resp.Diagnostics.Append(callAiAgentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

func callAiAgentCreateAPI(ctx context.Context, r *aiAgentResource, data *resource_ai_agent.AiAgentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq := quantadmingo.NewCreateAIAgentRequest(
		data.Name.ValueString(),
		data.Description.ValueString(),
		data.SystemPrompt.ValueString(),
		data.ModelId.ValueString(),
	)

	if !data.Group.IsNull() && !data.Group.IsUnknown() {
		sdkReq.SetGroup(data.Group.ValueString())
	}
	if !data.Temperature.IsNull() && !data.Temperature.IsUnknown() {
		v, _ := data.Temperature.ValueBigFloat().Float32()
		sdkReq.SetTemperature(v)
	}
	if !data.MaxTokens.IsNull() && !data.MaxTokens.IsUnknown() {
		sdkReq.SetMaxTokens(int32(data.MaxTokens.ValueInt64()))
	}
	if !data.AllowedTools.IsNull() && !data.AllowedTools.IsUnknown() {
		var tools []string
		diags.Append(data.AllowedTools.ElementsAs(ctx, &tools, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedTools(tools)
	}
	if !data.AllowedCollections.IsNull() && !data.AllowedCollections.IsUnknown() {
		var collections []string
		diags.Append(data.AllowedCollections.ElementsAs(ctx, &collections, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedCollections(collections)
	}
	if !data.AssignedSkills.IsNull() && !data.AssignedSkills.IsUnknown() {
		var skills []string
		diags.Append(data.AssignedSkills.ElementsAs(ctx, &skills, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAssignedSkills(skills)
	}
	if !data.CreatedBy.IsNull() && !data.CreatedBy.IsUnknown() {
		sdkReq.SetCreatedBy(data.CreatedBy.ValueString())
	}
	if !data.LongContext.IsNull() && !data.LongContext.IsUnknown() {
		sdkReq.SetLongContext(data.LongContext.ValueBool())
	}
	if !data.GuardrailPreset.IsNull() && !data.GuardrailPreset.IsUnknown() {
		sdkReq.SetGuardrailPreset(data.GuardrailPreset.ValueString())
	}
	if !data.FilterPolicies.IsNull() && !data.FilterPolicies.IsUnknown() {
		var policies []string
		diags.Append(data.FilterPolicies.ElementsAs(ctx, &policies, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetFilterPolicies(policies)
	}

	sdkResp, httpResp, err := r.client.Instance.AIAgentsAPI.CreateAIAgent(r.client.AuthContext, org).
		CreateAIAgentRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to create AI agent", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to create AI agent", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Create response contains the created agent as a map — map it to state.
	if sdkResp != nil && sdkResp.Agent != nil {
		diags.Append(mapAiAgentFromMap(ctx, sdkResp.Agent, org, data)...)
	}

	// Set computed Success/Message fields.
	if sdkResp != nil && sdkResp.Success != nil {
		data.Success = types.BoolValue(*sdkResp.Success)
	} else {
		data.Success = types.BoolValue(true)
	}
	if sdkResp != nil && sdkResp.Message != nil {
		data.Message = types.StringValue(*sdkResp.Message)
	} else {
		data.Message = types.StringNull()
	}

	data.Organisation = types.StringValue(org)
	return
}

func callAiAgentReadAPI(ctx context.Context, r *aiAgentResource, data *resource_ai_agent.AiAgentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AIAgentsAPI.GetAIAgent(r.client.AuthContext, org, data.AgentId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			data.AgentId = types.StringNull()
			return
		}
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read AI agent", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to read AI agent", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	if sdkResp != nil && sdkResp.Agent != nil {
		diags.Append(mapAiAgentResponse(ctx, sdkResp.Agent, org, data)...)
	}

	data.Organisation = types.StringValue(org)
	data.Success = types.BoolValue(true)
	data.Message = types.StringNull()
	return
}

func callAiAgentUpdateAPI(ctx context.Context, r *aiAgentResource, data *resource_ai_agent.AiAgentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq := quantadmingo.NewUpdateAIAgentRequest()

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		sdkReq.SetName(data.Name.ValueString())
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		sdkReq.SetDescription(data.Description.ValueString())
	}
	if !data.Group.IsNull() && !data.Group.IsUnknown() {
		sdkReq.SetGroup(data.Group.ValueString())
	}
	if !data.SystemPrompt.IsNull() && !data.SystemPrompt.IsUnknown() {
		sdkReq.SetSystemPrompt(data.SystemPrompt.ValueString())
	}
	if !data.Temperature.IsNull() && !data.Temperature.IsUnknown() {
		v, _ := data.Temperature.ValueBigFloat().Float32()
		sdkReq.SetTemperature(v)
	}
	if !data.ModelId.IsNull() && !data.ModelId.IsUnknown() {
		sdkReq.SetModelId(data.ModelId.ValueString())
	}
	if !data.MaxTokens.IsNull() && !data.MaxTokens.IsUnknown() {
		sdkReq.SetMaxTokens(int32(data.MaxTokens.ValueInt64()))
	}
	if !data.AllowedTools.IsNull() && !data.AllowedTools.IsUnknown() {
		var tools []string
		diags.Append(data.AllowedTools.ElementsAs(ctx, &tools, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedTools(tools)
	}
	if !data.AllowedCollections.IsNull() && !data.AllowedCollections.IsUnknown() {
		var collections []string
		diags.Append(data.AllowedCollections.ElementsAs(ctx, &collections, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedCollections(collections)
	}
	if !data.AssignedSkills.IsNull() && !data.AssignedSkills.IsUnknown() {
		var skills []string
		diags.Append(data.AssignedSkills.ElementsAs(ctx, &skills, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAssignedSkills(skills)
	}
	if !data.LongContext.IsNull() && !data.LongContext.IsUnknown() {
		sdkReq.SetLongContext(data.LongContext.ValueBool())
	}
	if !data.GuardrailPreset.IsNull() && !data.GuardrailPreset.IsUnknown() {
		sdkReq.SetGuardrailPreset(data.GuardrailPreset.ValueString())
	}
	if !data.FilterPolicies.IsNull() && !data.FilterPolicies.IsUnknown() {
		var policies []string
		diags.Append(data.FilterPolicies.ElementsAs(ctx, &policies, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetFilterPolicies(policies)
	}

	sdkResp, httpResp, err := r.client.Instance.AIAgentsAPI.UpdateAIAgent(r.client.AuthContext, org, data.AgentId.ValueString()).
		UpdateAIAgentRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to update AI agent", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to update AI agent", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	if sdkResp != nil && sdkResp.Agent != nil {
		diags.Append(mapAiAgentFromMap(ctx, sdkResp.Agent, org, data)...)
	}

	data.Organisation = types.StringValue(org)
	data.Success = types.BoolValue(true)
	data.Message = types.StringNull()
	return
}

func callAiAgentDeleteAPI(ctx context.Context, r *aiAgentResource, data *resource_ai_agent.AiAgentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	_, httpResp, err := r.client.Instance.AIAgentsAPI.DeleteAIAgent(r.client.AuthContext, org, data.AgentId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to delete AI agent", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to delete AI agent", fmt.Sprintf("Error: %s", err.Error()))
		}
	}
	return
}

// mapAiAgentFromMap maps a generic map (from Create/Update response) into
// the TF model. Create/Update SDK responses use `map[string]interface{}` for
// the agent payload instead of the typed GetAIAgent200ResponseAgent.
func mapAiAgentFromMap(ctx context.Context, m map[string]interface{}, org string, data *resource_ai_agent.AiAgentModel) (diags diag.Diagnostics) {
	if m == nil {
		return
	}
	agent := &quantadmingo.GetAIAgent200ResponseAgent{}
	if v, ok := m["agentId"].(string); ok {
		agent.AgentId = &v
	}
	if v, ok := m["name"].(string); ok {
		agent.Name = &v
	}
	if v, ok := m["description"].(string); ok {
		agent.Description = &v
	}
	if v, ok := m["group"].(string); ok {
		agent.Group = &v
	}
	if v, ok := m["systemPrompt"].(string); ok {
		agent.SystemPrompt = &v
	}
	if v, ok := m["modelId"].(string); ok {
		agent.ModelId = &v
	}
	if v, ok := m["createdBy"].(string); ok {
		agent.CreatedBy = &v
	}
	if v, ok := m["guardrailPreset"].(string); ok {
		agent.GuardrailPreset = &v
	}
	if v, ok := m["temperature"].(float64); ok {
		f := float32(v)
		agent.Temperature = &f
	}
	if v, ok := m["maxTokens"].(float64); ok {
		n := int32(v)
		agent.MaxTokens = &n
	}
	if v, ok := m["longContext"].(bool); ok {
		agent.LongContext = &v
	}
	if v, ok := m["isGlobal"].(bool); ok {
		agent.IsGlobal = &v
	}
	if v, ok := m["hasOverlay"].(bool); ok {
		agent.HasOverlay = &v
	}
	if v, ok := m["allowedTools"].([]interface{}); ok {
		agent.AllowedTools = stringSliceFromInterface(v)
	}
	if v, ok := m["allowedCollections"].([]interface{}); ok {
		agent.AllowedCollections = stringSliceFromInterface(v)
	}
	if v, ok := m["assignedSkills"].([]interface{}); ok {
		agent.AssignedSkills = stringSliceFromInterface(v)
	}
	if v, ok := m["createdAt"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			agent.CreatedAt = &t
		}
	}
	if v, ok := m["updatedAt"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			agent.UpdatedAt = &t
		}
	}
	return mapAiAgentResponse(ctx, agent, org, data)
}

func stringSliceFromInterface(s []interface{}) []string {
	out := make([]string, 0, len(s))
	for _, v := range s {
		if str, ok := v.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

// mapAiAgentResponse maps an SDK agent response into the TF model, populating
// both top-level user-facing fields and the nested read-only `agent` object.
func mapAiAgentResponse(ctx context.Context, agent *quantadmingo.GetAIAgent200ResponseAgent, org string, data *resource_ai_agent.AiAgentModel) (diags diag.Diagnostics) {
	if agent == nil {
		return
	}

	// Top-level fields — these mirror user input but API is authoritative.
	data.AgentId = optionalStringPtrValue(agent.AgentId)
	data.Name = optionalStringPtrValue(agent.Name)
	data.Description = optionalStringPtrValue(agent.Description)
	data.Group = optionalStringPtrValue(agent.Group)
	data.SystemPrompt = optionalStringPtrValue(agent.SystemPrompt)
	data.ModelId = optionalStringPtrValue(agent.ModelId)
	data.CreatedBy = optionalStringPtrValue(agent.CreatedBy)
	data.GuardrailPreset = optionalStringPtrValue(agent.GuardrailPreset)

	if agent.Temperature != nil {
		data.Temperature = types.NumberValue(big.NewFloat(float64(*agent.Temperature)))
	} else {
		data.Temperature = types.NumberNull()
	}
	if agent.MaxTokens != nil {
		data.MaxTokens = types.Int64Value(int64(*agent.MaxTokens))
	} else {
		data.MaxTokens = types.Int64Null()
	}
	if agent.LongContext != nil {
		data.LongContext = types.BoolValue(*agent.LongContext)
	} else {
		data.LongContext = types.BoolNull()
	}

	data.AllowedTools = stringListFromSlice(ctx, agent.AllowedTools)
	data.AllowedCollections = stringListFromSlice(ctx, agent.AllowedCollections)
	data.AssignedSkills = stringListFromSlice(ctx, agent.AssignedSkills)
	// FilterPolicies is not on the response model — always set to a typed null
	// so the list has a concrete element type (bridge rejects dynamic type).
	if data.FilterPolicies.IsUnknown() || data.FilterPolicies.IsNull() || data.FilterPolicies.ElementType(ctx) == nil {
		data.FilterPolicies = types.ListNull(types.StringType)
	}

	// Nested agent object — all computed, mirrors the top-level fields plus
	// read-only metadata (agentId, isGlobal, hasOverlay, timestamps).
	agentAttrTypes := resource_ai_agent.AgentValue{}.AttributeTypes(ctx)
	agentAttrs := map[string]attr.Value{
		"agent_id":            data.AgentId,
		"name":                data.Name,
		"description":         data.Description,
		"group":               data.Group,
		"system_prompt":       data.SystemPrompt,
		"model_id":            data.ModelId,
		"temperature":         data.Temperature,
		"max_tokens":          data.MaxTokens,
		"allowed_tools":       data.AllowedTools,
		"allowed_collections": data.AllowedCollections,
		"assigned_skills":     data.AssignedSkills,
		"long_context":        data.LongContext,
		"guardrail_preset":    data.GuardrailPreset,
		"created_by":          data.CreatedBy,
		"is_global":           boolPtrToValue(agent.IsGlobal),
		"has_overlay":         boolPtrToValue(agent.HasOverlay),
		"created_at":          timePtrToString(agent.CreatedAt),
		"updated_at":          timePtrToString(agent.UpdatedAt),
	}
	agentValue, d := resource_ai_agent.NewAgentValue(agentAttrTypes, agentAttrs)
	diags.Append(d...)
	if !diags.HasError() {
		data.Agent = agentValue
	}

	return
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

func optionalStringPtrValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func boolPtrToValue(b *bool) basetypes.BoolValue {
	if b == nil {
		return basetypes.NewBoolNull()
	}
	return basetypes.NewBoolValue(*b)
}

func stringListFromSlice(ctx context.Context, s []string) types.List {
	if s == nil {
		return types.ListNull(types.StringType)
	}
	l, _ := types.ListValueFrom(ctx, types.StringType, s)
	return l
}

func timePtrToString(t *time.Time) basetypes.StringValue {
	if t == nil {
		return basetypes.NewStringNull()
	}
	return basetypes.NewStringValue(t.Format("2006-01-02T15:04:05Z07:00"))
}
