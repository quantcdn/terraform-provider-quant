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
	s := resource_ai_agent_overlay.AiAgentOverlayResourceSchema(ctx)
	clearValidators(s.Attributes, "guardrail_preset")
	resp.Schema = s
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

// upsert sends the overlay PUT. The API replaces the whole overlay, so a field
// the request leaves out is deleted. The request therefore starts from the
// stored overlay and then takes every value the plan sets: an attribute that
// the configuration does not set keeps its stored value (for example filter
// policies or MCP attachments set in the dashboard). To clear a list, set it
// to []; to clear a string, set it to "".
func (r *aiAgentOverlayResource) upsert(ctx context.Context, data *resource_ai_agent_overlay.AiAgentOverlayModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	if data.AgentId.IsNull() || data.AgentId.IsUnknown() || data.AgentId.ValueString() == "" {
		diags.AddError("Missing agent_id", "agent_id is required to upsert an AI agent overlay.")
		return
	}
	agentId := data.AgentId.ValueString()

	stored, d := r.storedOverlay(org, agentId)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	sdkReq := storedOverlayRequest(stored)
	applyPlannedScalars(data, sdkReq)
	diags.Append(applyPlannedLists(ctx, data, sdkReq)...)
	diags.Append(applyPlannedMcpServers(ctx, data, sdkReq)...)
	if diags.HasError() {
		return
	}

	// version — the state's version (compare-and-swap) when there is one, else
	// the version just read, so a change made between the read and the PUT
	// returns 409 instead of being overwritten.
	if !data.Version.IsNull() && !data.Version.IsUnknown() && data.Version.ValueInt64() > 0 {
		sdkReq.SetVersion(int32(data.Version.ValueInt64()))
	}

	_, httpResp, err := r.client.Instance.AIAgentsAPI.UpsertAgentOverlay(r.client.AuthContext, org, agentId).
		UpsertAgentOverlayRequest(*sdkReq).Execute()
	if err != nil {
		diags.Append(overlayUpsertError(httpResp, err)...)
	}
	return
}

// storedOverlay reads the overlay the API stores now. No overlay (404, or an
// empty overlay) returns nil without an error.
func (r *aiAgentOverlayResource) storedOverlay(org, agentId string) (*quantadmingo.GetAgentOverlay200ResponseOverlay, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiResp, httpResp, err := r.client.Instance.AIAgentsAPI.GetAgentOverlay(r.client.AuthContext, org, agentId).Execute()
	if err == nil {
		return apiResp.Overlay.Get(), diags
	}
	if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
		return nil, diags
	}
	if httpResp != nil {
		body, _ := io.ReadAll(httpResp.Body)
		diags.AddError("Unable to read the stored AI agent overlay",
			fmt.Sprintf("The overlay PUT replaces the whole overlay, so the provider reads it first and the read failed. API returned %d: %s", httpResp.StatusCode, string(body)))
		return nil, diags
	}
	diags.AddError("Unable to read the stored AI agent overlay", fmt.Sprintf("Error: %s", err.Error()))
	return nil, diags
}

// storedOverlayRequest starts a PUT request from every field of the stored
// overlay, including its version.
func storedOverlayRequest(o *quantadmingo.GetAgentOverlay200ResponseOverlay) *quantadmingo.UpsertAgentOverlayRequest {
	req := quantadmingo.NewUpsertAgentOverlayRequest()
	if o == nil {
		return req
	}
	req.ModelId = o.ModelId
	req.Temperature = o.Temperature
	req.MaxTokens = o.MaxTokens
	req.SystemPromptAppend = o.SystemPromptAppend
	req.DisabledSkills = o.DisabledSkills
	req.AdditionalSkills = o.AdditionalSkills
	req.AdditionalTools = o.AdditionalTools
	req.DisabledTools = o.DisabledTools
	req.AllowedCollections = o.AllowedCollections
	req.FilterPolicies = o.FilterPolicies
	req.McpServers = mcpAttachmentsFromStored(o.McpServers)
	req.Version = o.Version
	if o.GuardrailPreset != nil {
		req.SetGuardrailPreset(*o.GuardrailPreset)
	}
	if o.LongContext != nil {
		req.SetLongContext(*o.LongContext)
	}
	if effort, ok := o.GetReasoningEffortOk(); ok && effort != nil {
		req.SetReasoningEffort(*effort)
	}
	return req
}

// applyPlannedScalars sets each single-value attribute the plan knows.
func applyPlannedScalars(data *resource_ai_agent_overlay.AiAgentOverlayModel, req *quantadmingo.UpsertAgentOverlayRequest) {
	if known(data.ModelId) {
		req.SetModelId(data.ModelId.ValueString())
	}
	if known(data.ReasoningEffort) {
		req.SetReasoningEffort(data.ReasoningEffort.ValueString())
	}
	if known(data.Temperature) && data.Temperature.ValueBigFloat() != nil {
		f32, _ := data.Temperature.ValueBigFloat().Float32()
		req.SetTemperature(f32)
	}
	if known(data.MaxTokens) {
		req.SetMaxTokens(int32(data.MaxTokens.ValueInt64()))
	}
	if known(data.SystemPromptAppend) {
		req.SetSystemPromptAppend(data.SystemPromptAppend.ValueString())
	}
	if known(data.GuardrailPreset) {
		req.SetGuardrailPreset(data.GuardrailPreset.ValueString())
	}
	if known(data.LongContext) {
		req.SetLongContext(data.LongContext.ValueBool())
	}
}

// applyPlannedLists sets each string list the plan knows. A configured empty
// list replaces the stored one, which clears it.
func applyPlannedLists(ctx context.Context, data *resource_ai_agent_overlay.AiAgentOverlayModel, req *quantadmingo.UpsertAgentOverlayRequest) (diags diag.Diagnostics) {
	lists := []struct {
		value types.List
		set   func([]string)
	}{
		{data.DisabledSkills, req.SetDisabledSkills},
		{data.AdditionalSkills, req.SetAdditionalSkills},
		{data.AdditionalTools, req.SetAdditionalTools},
		{data.DisabledTools, req.SetDisabledTools},
		{data.AllowedCollections, req.SetAllowedCollections},
		{data.FilterPolicies, req.SetFilterPolicies},
	}
	for _, l := range lists {
		if !known(l.value) {
			continue
		}
		vals := []string{}
		diags.Append(l.value.ElementsAs(ctx, &vals, false)...)
		l.set(vals)
	}
	return
}

// applyPlannedMcpServers sets the MCP attachments when the plan knows them.
func applyPlannedMcpServers(ctx context.Context, data *resource_ai_agent_overlay.AiAgentOverlayModel, req *quantadmingo.UpsertAgentOverlayRequest) (diags diag.Diagnostics) {
	if !known(data.McpServers) {
		return
	}
	var elems []resource_ai_agent_overlay.McpServersValue
	diags.Append(data.McpServers.ElementsAs(ctx, &elems, false)...)
	out := make([]quantadmingo.CreateAIAgentRequestMcpServersInner, 0, len(elems))
	for _, e := range elems {
		a, d := mcpAttachment(ctx, e.Server, e.Tools)
		diags.Append(d...)
		out = append(out, a)
	}
	req.SetMcpServers(out)
	return
}

func overlayUpsertError(httpResp *http.Response, err error) (diags diag.Diagnostics) {
	if httpResp == nil {
		diags.AddError("Unable to upsert AI agent overlay", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode == http.StatusConflict {
		diags.AddError("Overlay version conflict",
			fmt.Sprintf("The overlay was modified concurrently (HTTP 409). Refresh state (terraform refresh) and retry. Response: %s", string(body)))
		return
	}
	diags.AddError("Unable to upsert AI agent overlay",
		fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
	return
}

// known reports whether a plan value is set (not null and not unknown).
func known(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
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
// data.Overlay nested struct from the SDK overlay response. A value cleared
// with "" or [] reads back as absent; the planned empty value then stays, so
// Terraform sees the result it planned.
func populateOverlay(ctx context.Context, overlay *quantadmingo.GetAgentOverlay200ResponseOverlay, data *resource_ai_agent_overlay.AiAgentOverlayModel) (diags diag.Diagnostics) {
	noOverlay := overlay == nil
	if noOverlay {
		overlay = &quantadmingo.GetAgentOverlay200ResponseOverlay{}
	}

	data.ModelId = keptString(overlay.ModelId, data.ModelId)
	data.SystemPromptAppend = keptString(overlay.SystemPromptAppend, data.SystemPromptAppend)
	data.GuardrailPreset = keptString(overlay.GuardrailPreset, data.GuardrailPreset)
	effort, _ := overlay.GetReasoningEffortOk()
	data.ReasoningEffort = optionalStringPtrValue(effort)
	data.Temperature = roundedNumber(overlay.Temperature)
	data.MaxTokens = int64PtrValue(overlay.MaxTokens)
	data.LongContext = boolPtrToValue(overlay.LongContext)
	data.Version = int64PtrValue(overlay.Version)

	data.DisabledSkills = keptList(ctx, overlay.DisabledSkills, data.DisabledSkills)
	data.AdditionalSkills = keptList(ctx, overlay.AdditionalSkills, data.AdditionalSkills)
	data.AdditionalTools = keptList(ctx, overlay.AdditionalTools, data.AdditionalTools)
	data.DisabledTools = keptList(ctx, overlay.DisabledTools, data.DisabledTools)
	data.AllowedCollections = keptList(ctx, overlay.AllowedCollections, data.AllowedCollections)
	data.FilterPolicies = keptList(ctx, overlay.FilterPolicies, data.FilterPolicies)

	mcp, d := overlayMcpServersValue(ctx, overlay.McpServers)
	diags.Append(d...)
	if overlay.McpServers == nil && plannedEmpty(data.McpServers) {
		mcp = data.McpServers
	}
	data.McpServers = mcp

	if noOverlay || diags.HasError() {
		data.Overlay = resource_ai_agent_overlay.NewOverlayValueNull()
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
		"reasoning_effort":     data.ReasoningEffort,
		"long_context":         data.LongContext,
		"filter_policies":      data.FilterPolicies,
		"mcp_servers":          data.McpServers,
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

// plannedEmpty reports whether a list value is a known, empty list.
func plannedEmpty(v types.List) bool {
	return known(v) && len(v.Elements()) == 0
}

// keptString is the stored string, or the planned "" when the API stores none.
func keptString(stored *string, planned types.String) types.String {
	if stored == nil && known(planned) && planned.ValueString() == "" {
		return planned
	}
	return optionalStringPtrValue(stored)
}

// keptList is the stored list, or the planned [] when the API stores none.
func keptList(ctx context.Context, stored []string, planned types.List) types.List {
	if stored == nil && plannedEmpty(planned) {
		return planned
	}
	return stringListFromSlice(ctx, stored)
}

// roundedNumber converts a float32 to a Number rounded to 2 decimal places, to
// avoid float32 to float64 drift (0.3 reads as 0.30000001192092896).
func roundedNumber(f *float32) types.Number {
	if f == nil {
		return types.NumberNull()
	}
	return types.NumberValue(big.NewFloat(math.Round(float64(*f)*100) / 100))
}

func int64PtrValue(n *int32) types.Int64 {
	if n == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*n))
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

// overlayMcpServersType is the element type of the overlay's mcp_servers list.
func overlayMcpServersType(ctx context.Context) attr.Type {
	return resource_ai_agent_overlay.McpServersType{
		ObjectType: types.ObjectType{AttrTypes: resource_ai_agent_overlay.McpServersValue{}.AttributeTypes(ctx)},
	}
}

// overlayMcpServersValue converts the stored attachments to the mcp_servers
// list. No attachments read as null.
func overlayMcpServersValue(ctx context.Context, stored []quantadmingo.GetAgentOverlay200ResponseOverlayMcpServersInner) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := overlayMcpServersType(ctx)
	if stored == nil {
		return types.ListNull(elemType), diags
	}
	attrTypes := resource_ai_agent_overlay.McpServersValue{}.AttributeTypes(ctx)
	elems := make([]attr.Value, 0, len(stored))
	for _, s := range stored {
		v, d := resource_ai_agent_overlay.NewMcpServersValue(attrTypes, mcpServerAttrs(ctx, s))
		diags.Append(d...)
		elems = append(elems, v)
	}
	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}
