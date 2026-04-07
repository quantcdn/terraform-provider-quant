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
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_slack_bot"
)

var (
	_ resource.Resource                = (*slackBotResource)(nil)
	_ resource.ResourceWithConfigure   = (*slackBotResource)(nil)
	_ resource.ResourceWithImportState = (*slackBotResource)(nil)
)

func NewSlackBotResource() resource.Resource {
	return &slackBotResource{}
}

type slackBotResource struct {
	client *client.Client
}

func (r *slackBotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slack_bot"
}

func (r *slackBotResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_slack_bot.SlackBotResourceSchema(ctx)
}

func (r *slackBotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *slackBotResource) getOrg(data *resource_slack_bot.SlackBotModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

func (r *slackBotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_slack_bot.SlackBotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSlackBotCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *slackBotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_slack_bot.SlackBotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSlackBotReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.BotId.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *slackBotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_slack_bot.SlackBotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve BotId from state (server-generated).
	var state resource_slack_bot.SlackBotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.BotId = state.BotId

	resp.Diagnostics.Append(callSlackBotUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *slackBotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_slack_bot.SlackBotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSlackBotDeleteAPI(ctx, r, &data)...)
}

func (r *slackBotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_slack_bot.SlackBotModel
	data.BotId = types.StringValue(req.ID)

	resp.Diagnostics.Append(callSlackBotReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

// slackBotExtraFields gathers all inline agent config fields from the TF model
// into a map that can be fed to the SDK request's AdditionalProperties (the SDK
// struct hasn't been regenerated for the new API shape yet).
func slackBotExtraFields(ctx context.Context, data *resource_slack_bot.SlackBotModel) (extra map[string]interface{}, diags diag.Diagnostics) {
	extra = make(map[string]interface{})

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		extra["name"] = data.Name.ValueString()
	}
	if !data.SystemPrompt.IsNull() && !data.SystemPrompt.IsUnknown() {
		extra["systemPrompt"] = data.SystemPrompt.ValueString()
	}
	if !data.ModelId.IsNull() && !data.ModelId.IsUnknown() {
		extra["modelId"] = data.ModelId.ValueString()
	}
	if !data.Temperature.IsNull() && !data.Temperature.IsUnknown() {
		f, _ := data.Temperature.ValueBigFloat().Float64()
		extra["temperature"] = f
	}
	if !data.MaxTokens.IsNull() && !data.MaxTokens.IsUnknown() {
		extra["maxTokens"] = data.MaxTokens.ValueInt64()
	}
	if !data.LongContext.IsNull() && !data.LongContext.IsUnknown() {
		extra["longContext"] = data.LongContext.ValueBool()
	}
	if !data.GuardrailPreset.IsNull() && !data.GuardrailPreset.IsUnknown() {
		extra["guardrailPreset"] = data.GuardrailPreset.ValueString()
	}
	if !data.HomeTabContent.IsNull() && !data.HomeTabContent.IsUnknown() {
		extra["homeTabContent"] = data.HomeTabContent.ValueString()
	}
	if !data.AllowGuests.IsNull() && !data.AllowGuests.IsUnknown() {
		extra["allowGuests"] = data.AllowGuests.ValueBool()
	}

	// String list fields.
	listFields := map[string]*types.List{
		"allowedTools":       &data.AllowedTools,
		"assignedSkills":     &data.AssignedSkills,
		"allowedCollections": &data.AllowedCollections,
		"allowedSubAgents":   &data.AllowedSubAgents,
		"allowedUsers":       &data.AllowedUsers,
		"deniedUsers":        &data.DeniedUsers,
		"filterPolicies":     &data.FilterPolicies,
	}
	for key, listVal := range listFields {
		if !listVal.IsNull() && !listVal.IsUnknown() {
			var vals []string
			diags.Append(listVal.ElementsAs(ctx, &vals, false)...)
			if diags.HasError() {
				return
			}
			extra[key] = vals
		}
	}

	return
}

func callSlackBotCreateAPI(ctx context.Context, r *slackBotResource, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	// The SDK constructor still requires (agentId, setupType). The API now
	// auto-creates the backing agent, so we pass an empty agentId — it will
	// be ignored by the server. The real agent config fields are sent via
	// AdditionalProperties.
	sdkReq := quantadmingo.NewCreateSlackBotRequest(
		"", // agentId — server creates agent internally
		data.SetupType.ValueString(),
	)

	if !data.SessionTtlDays.IsNull() && !data.SessionTtlDays.IsUnknown() {
		sdkReq.SetSessionTtlDays(int32(data.SessionTtlDays.ValueInt64()))
	}
	if !data.AllowedChannels.IsNull() && !data.AllowedChannels.IsUnknown() {
		var chans []string
		diags.Append(data.AllowedChannels.ElementsAs(ctx, &chans, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedChannels(chans)
	}
	if !data.KeywordsEnabled.IsNull() && !data.KeywordsEnabled.IsUnknown() {
		sdkReq.SetKeywordsEnabled(data.KeywordsEnabled.ValueBool())
	}
	if !data.Keywords.IsNull() && !data.Keywords.IsUnknown() {
		var kw []string
		diags.Append(data.Keywords.ElementsAs(ctx, &kw, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetKeywords(kw)
	}

	// New inline agent config fields — not yet in SDK struct, sent via
	// AdditionalProperties.
	extra, d := slackBotExtraFields(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	sdkReq.AdditionalProperties = extra

	sdkResp, httpResp, err := r.client.Instance.AISlackBotsAPI.CreateSlackBot(r.client.AuthContext, org).
		CreateSlackBotRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to create Slack bot", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to create Slack bot", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	if sdkResp != nil && sdkResp.Bot != nil {
		diags.Append(mapSlackBotFromMap(ctx, sdkResp.Bot, org, data)...)
	}
	data.Organisation = types.StringValue(org)
	return
}

func callSlackBotReadAPI(ctx context.Context, r *slackBotResource, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AISlackBotsAPI.GetSlackBot(r.client.AuthContext, org, data.BotId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			data.BotId = types.StringNull()
			return
		}
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read Slack bot", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to read Slack bot", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	if sdkResp != nil && sdkResp.Bot != nil {
		diags.Append(mapSlackBotResponse(ctx, sdkResp.Bot, org, data)...)
	}
	data.Organisation = types.StringValue(org)
	return
}

func callSlackBotUpdateAPI(ctx context.Context, r *slackBotResource, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq := quantadmingo.NewUpdateSlackBotRequest()

	if !data.SessionTtlDays.IsNull() && !data.SessionTtlDays.IsUnknown() {
		sdkReq.SetSessionTtlDays(int32(data.SessionTtlDays.ValueInt64()))
	}
	if !data.AllowedChannels.IsNull() && !data.AllowedChannels.IsUnknown() {
		var chans []string
		diags.Append(data.AllowedChannels.ElementsAs(ctx, &chans, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetAllowedChannels(chans)
	}
	if !data.KeywordsEnabled.IsNull() && !data.KeywordsEnabled.IsUnknown() {
		sdkReq.SetKeywordsEnabled(data.KeywordsEnabled.ValueBool())
	}
	if !data.Keywords.IsNull() && !data.Keywords.IsUnknown() {
		var kw []string
		diags.Append(data.Keywords.ElementsAs(ctx, &kw, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetKeywords(kw)
	}

	// New inline agent config fields via AdditionalProperties.
	extra, d := slackBotExtraFields(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	sdkReq.AdditionalProperties = extra

	sdkResp, httpResp, err := r.client.Instance.AISlackBotsAPI.UpdateSlackBot(r.client.AuthContext, org, data.BotId.ValueString()).
		UpdateSlackBotRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to update Slack bot", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to update Slack bot", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	if sdkResp != nil && sdkResp.Bot != nil {
		diags.Append(mapSlackBotFromMap(ctx, sdkResp.Bot, org, data)...)
	}
	data.Organisation = types.StringValue(org)
	return
}

func callSlackBotDeleteAPI(ctx context.Context, r *slackBotResource, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	_, httpResp, err := r.client.Instance.AISlackBotsAPI.DeleteSlackBot(r.client.AuthContext, org, data.BotId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to delete Slack bot", fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to delete Slack bot", fmt.Sprintf("Error: %s", err.Error()))
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Response mapping helpers
// ---------------------------------------------------------------------------

// mapSlackBotFromMap converts a generic map (Create/Update response) into
// the TF model. The API returns a flattened object with both bot-level and
// agent-level fields at the top level.
func mapSlackBotFromMap(ctx context.Context, m map[string]interface{}, org string, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	if m == nil {
		return
	}

	// --- Top-level configurable fields ---
	data.BotId = mapStringFromAny(m, "botId")
	data.SetupType = mapStringFromAny(m, "setupType")
	data.Name = mapStringFromAny(m, "name")
	data.SystemPrompt = mapStringFromAny(m, "systemPrompt")
	data.ModelId = mapStringFromAny(m, "modelId")
	data.GuardrailPreset = mapStringFromAny(m, "guardrailPreset")
	data.HomeTabContent = mapStringFromAny(m, "homeTabContent")

	data.SessionTtlDays = mapInt64FromAny(m, "sessionTtlDays")
	data.MaxTokens = mapInt64FromAny(m, "maxTokens")

	data.KeywordsEnabled = mapBoolFromAny(m, "keywordsEnabled")
	data.AllowGuests = mapBoolFromAny(m, "allowGuests")
	data.LongContext = mapBoolFromAny(m, "longContext")

	data.Temperature = mapNumberFromAny(m, "temperature")

	data.AllowedChannels = mapStringListFromAny(ctx, m, "allowedChannels")
	data.Keywords = mapStringListFromAny(ctx, m, "keywords")
	data.AllowedTools = mapStringListFromAny(ctx, m, "allowedTools")
	data.AssignedSkills = mapStringListFromAny(ctx, m, "assignedSkills")
	data.AllowedCollections = mapStringListFromAny(ctx, m, "allowedCollections")
	data.AllowedSubAgents = mapStringListFromAny(ctx, m, "allowedSubAgents")
	data.AllowedUsers = mapStringListFromAny(ctx, m, "allowedUsers")
	data.DeniedUsers = mapStringListFromAny(ctx, m, "deniedUsers")
	data.FilterPolicies = mapStringListFromAny(ctx, m, "filterPolicies")

	// AgentAccessControl — empty object for now.
	data.AgentAccessControl = resource_slack_bot.NewAgentAccessControlValueMust(
		resource_slack_bot.AgentAccessControlValue{}.AttributeTypes(ctx),
		map[string]attr.Value{},
	)

	// --- Nested computed bot object (mirrors all fields + read-only metadata) ---
	diags.Append(buildBotObject(ctx, m, data)...)

	// Silence unused import.
	_ = basetypes.StringValue{}
	return
}

// mapSlackBotResponse maps a typed GetSlackBot200ResponseBot into the TF model.
// New fields that aren't in the SDK struct are read from AdditionalProperties.
func mapSlackBotResponse(ctx context.Context, bot *quantadmingo.GetSlackBot200ResponseBot, org string, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	if bot == nil {
		return
	}

	// Build a unified map from typed fields + additional properties so we
	// can use the same mapping logic as mapSlackBotFromMap.
	m := make(map[string]interface{})

	// Typed fields the SDK knows about.
	if bot.BotId != nil {
		m["botId"] = *bot.BotId
	}
	if bot.SetupType != nil {
		m["setupType"] = *bot.SetupType
	}
	if bot.Status != nil {
		m["status"] = *bot.Status
	}
	if bot.Connected != nil {
		m["connected"] = *bot.Connected
	}
	if bot.SessionTtlDays != nil {
		m["sessionTtlDays"] = float64(*bot.SessionTtlDays)
	}
	if bot.KeywordsEnabled != nil {
		m["keywordsEnabled"] = *bot.KeywordsEnabled
	}
	if bot.AllowedChannels != nil {
		iface := make([]interface{}, len(bot.AllowedChannels))
		for i, v := range bot.AllowedChannels {
			iface[i] = v
		}
		m["allowedChannels"] = iface
	}
	if bot.Keywords != nil {
		iface := make([]interface{}, len(bot.Keywords))
		for i, v := range bot.Keywords {
			iface[i] = v
		}
		m["keywords"] = iface
	}
	if bot.CreatedAt != nil {
		m["createdAt"] = bot.CreatedAt.Format(time.RFC3339)
	}
	if bot.UpdatedAt != nil {
		m["updatedAt"] = bot.UpdatedAt.Format(time.RFC3339)
	}

	// New fields from AdditionalProperties.
	for k, v := range bot.AdditionalProperties {
		m[k] = v
	}

	diags.Append(mapSlackBotFromMap(ctx, m, org, data)...)
	return
}

// buildBotObject constructs the nested computed `bot` attribute from the
// flat API response map.
func buildBotObject(ctx context.Context, m map[string]interface{}, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	botAttrTypes := resource_slack_bot.BotValue{}.AttributeTypes(ctx)
	botAttrs := map[string]attr.Value{
		"bot_id":              data.BotId,
		"setup_type":          data.SetupType,
		"name":                data.Name,
		"system_prompt":       data.SystemPrompt,
		"model_id":            data.ModelId,
		"guardrail_preset":    data.GuardrailPreset,
		"home_tab_content":    data.HomeTabContent,
		"session_ttl_days":    data.SessionTtlDays,
		"max_tokens":          data.MaxTokens,
		"keywords_enabled":    data.KeywordsEnabled,
		"allow_guests":        data.AllowGuests,
		"long_context":        data.LongContext,
		"temperature":         data.Temperature,
		"allowed_channels":    data.AllowedChannels,
		"keywords":            data.Keywords,
		"allowed_tools":       data.AllowedTools,
		"assigned_skills":     data.AssignedSkills,
		"allowed_collections": data.AllowedCollections,
		"allowed_sub_agents":  data.AllowedSubAgents,
		"allowed_users":       data.AllowedUsers,
		"denied_users":        data.DeniedUsers,
		"filter_policies":     data.FilterPolicies,
		// Read-only metadata.
		"status":    mapStringFromAny(m, "status"),
		"connected": mapBoolFromAny(m, "connected"),
		"created_at": func() basetypes.StringValue {
			if v, ok := m["createdAt"].(string); ok && v != "" {
				return types.StringValue(v)
			}
			return types.StringNull()
		}(),
		"updated_at": func() basetypes.StringValue {
			if v, ok := m["updatedAt"].(string); ok && v != "" {
				return types.StringValue(v)
			}
			return types.StringNull()
		}(),
		"agent_access_control": func() basetypes.ObjectValue {
			v, d := types.ObjectValue(
				resource_slack_bot.AgentAccessControlValue{}.AttributeTypes(ctx),
				map[string]attr.Value{},
			)
			diags.Append(d...)
			return v
		}(),
	}

	botValue, d := resource_slack_bot.NewBotValue(botAttrTypes, botAttrs)
	diags.Append(d...)
	if !diags.HasError() {
		data.Bot = botValue
	}
	return
}

// ---------------------------------------------------------------------------
// Generic map → TF value helpers
// ---------------------------------------------------------------------------

func mapStringFromAny(m map[string]interface{}, key string) types.String {
	if v, ok := m[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

func mapBoolFromAny(m map[string]interface{}, key string) basetypes.BoolValue {
	if v, ok := m[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}

func mapInt64FromAny(m map[string]interface{}, key string) types.Int64 {
	switch v := m[key].(type) {
	case float64:
		return types.Int64Value(int64(v))
	case int64:
		return types.Int64Value(v)
	case int32:
		return types.Int64Value(int64(v))
	case int:
		return types.Int64Value(int64(v))
	}
	return types.Int64Null()
}

func mapNumberFromAny(m map[string]interface{}, key string) basetypes.NumberValue {
	switch v := m[key].(type) {
	case float64:
		return types.NumberValue(big.NewFloat(v))
	case float32:
		return types.NumberValue(big.NewFloat(float64(v)))
	}
	return types.NumberNull()
}

func mapStringListFromAny(ctx context.Context, m map[string]interface{}, key string) types.List {
	switch v := m[key].(type) {
	case []interface{}:
		strs := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				strs = append(strs, s)
			}
		}
		return stringListFromSlice(ctx, strs)
	case []string:
		return stringListFromSlice(ctx, v)
	}
	return types.ListNull(types.StringType)
}
