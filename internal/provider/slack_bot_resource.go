package provider

import (
	"context"
	"fmt"
	"io"
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

func callSlackBotCreateAPI(ctx context.Context, r *slackBotResource, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq := quantadmingo.NewCreateSlackBotRequest(
		data.AgentId.ValueString(),
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
	if !data.SlashCommands.IsNull() && !data.SlashCommands.IsUnknown() {
		var cmds []string
		diags.Append(data.SlashCommands.ElementsAs(ctx, &cmds, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetSlashCommands(cmds)
	}

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

	if !data.AgentId.IsNull() && !data.AgentId.IsUnknown() {
		sdkReq.SetAgentId(data.AgentId.ValueString())
	}
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
	if !data.SlashCommands.IsNull() && !data.SlashCommands.IsUnknown() {
		var cmds []string
		diags.Append(data.SlashCommands.ElementsAs(ctx, &cmds, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.SetSlashCommands(cmds)
	}

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

// mapSlackBotFromMap converts a generic map (Create/Update response) into
// the TF model by first deserialising into a typed struct.
func mapSlackBotFromMap(ctx context.Context, m map[string]interface{}, org string, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	if m == nil {
		return
	}
	bot := &quantadmingo.GetSlackBot200ResponseBot{}
	if v, ok := m["botId"].(string); ok {
		bot.BotId = &v
	}
	if v, ok := m["agentId"].(string); ok {
		bot.AgentId = &v
	}
	if v, ok := m["setupType"].(string); ok {
		bot.SetupType = &v
	}
	if v, ok := m["status"].(string); ok {
		bot.Status = &v
	}
	if v, ok := m["connected"].(bool); ok {
		bot.Connected = &v
	}
	if v, ok := m["sessionTtlDays"].(float64); ok {
		n := int32(v)
		bot.SessionTtlDays = &n
	}
	if v, ok := m["keywordsEnabled"].(bool); ok {
		bot.KeywordsEnabled = &v
	}
	if v, ok := m["allowedChannels"].([]interface{}); ok {
		bot.AllowedChannels = stringSliceFromInterface(v)
	}
	if v, ok := m["keywords"].([]interface{}); ok {
		bot.Keywords = stringSliceFromInterface(v)
	}
	if v, ok := m["slashCommands"].([]interface{}); ok {
		bot.SlashCommands = stringSliceFromInterface(v)
	}
	if v, ok := m["createdAt"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			bot.CreatedAt = &t
		}
	}
	if v, ok := m["updatedAt"].(string); ok && v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			bot.UpdatedAt = &t
		}
	}
	return mapSlackBotResponse(ctx, bot, org, data)
}

// mapSlackBotResponse maps a typed GetSlackBot200ResponseBot into the TF model.
func mapSlackBotResponse(ctx context.Context, bot *quantadmingo.GetSlackBot200ResponseBot, org string, data *resource_slack_bot.SlackBotModel) (diags diag.Diagnostics) {
	if bot == nil {
		return
	}

	data.BotId = optionalStringPtrValue(bot.BotId)
	data.AgentId = optionalStringPtrValue(bot.AgentId)
	data.SetupType = optionalStringPtrValue(bot.SetupType)

	if bot.SessionTtlDays != nil {
		data.SessionTtlDays = types.Int64Value(int64(*bot.SessionTtlDays))
	} else {
		data.SessionTtlDays = types.Int64Null()
	}
	if bot.KeywordsEnabled != nil {
		data.KeywordsEnabled = types.BoolValue(*bot.KeywordsEnabled)
	} else {
		data.KeywordsEnabled = types.BoolNull()
	}

	data.AllowedChannels = stringListFromSlice(ctx, bot.AllowedChannels)
	data.Keywords = stringListFromSlice(ctx, bot.Keywords)
	data.SlashCommands = stringListFromSlice(ctx, bot.SlashCommands)

	// Nested bot object — all computed, mirrors top-level plus read-only
	// metadata (status, connected, timestamps).
	botAttrTypes := resource_slack_bot.BotValue{}.AttributeTypes(ctx)
	botAttrs := map[string]attr.Value{
		"bot_id":           data.BotId,
		"agent_id":         data.AgentId,
		"setup_type":       data.SetupType,
		"status":           optionalStringPtrValue(bot.Status),
		"connected":        boolPtrToValue(bot.Connected),
		"session_ttl_days": data.SessionTtlDays,
		"allowed_channels": data.AllowedChannels,
		"keywords_enabled": data.KeywordsEnabled,
		"keywords":         data.Keywords,
		"slash_commands":   data.SlashCommands,
		"created_at":       timePtrToString(bot.CreatedAt),
		"updated_at":       timePtrToString(bot.UpdatedAt),
	}
	botValue, d := resource_slack_bot.NewBotValue(botAttrTypes, botAttrs)
	diags.Append(d...)
	if !diags.HasError() {
		data.Bot = botValue
	}

	// Silence unused import
	_ = basetypes.StringValue{}
	return
}
