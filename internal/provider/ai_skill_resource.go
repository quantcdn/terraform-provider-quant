package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_skill"
)

var (
	_ resource.Resource                   = (*aiSkillResource)(nil)
	_ resource.ResourceWithConfigure      = (*aiSkillResource)(nil)
	_ resource.ResourceWithImportState    = (*aiSkillResource)(nil)
	_ resource.ResourceWithValidateConfig = (*aiSkillResource)(nil)
)

func NewAiSkillResource() resource.Resource {
	return &aiSkillResource{}
}

type aiSkillResource struct {
	client *client.Client
}

func (r *aiSkillResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_skill"
}

func (r *aiSkillResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_ai_skill.AiSkillResourceSchema(ctx)
}

func (r *aiSkillResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aiSkillResource) getOrg(data *resource_ai_skill.AiSkillModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

// ValidateConfig is a no-op; content is required by the schema.
func (r *aiSkillResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
}

func (r *aiSkillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_skill.AiSkillModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSkillCreateInlineAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiSkillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_skill.AiSkillModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSkillReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the skill was not found, remove from state so Terraform plans recreation.
	if data.SkillId.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiSkillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ai_skill.AiSkillModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSkillUpdateInlineAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiSkillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_skill.AiSkillModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callSkillDeleteAPI(ctx, r, &data)...)
}

// ImportState imports a skill by its UUID.
func (r *aiSkillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_ai_skill.AiSkillModel
	data.SkillId = types.StringValue(req.ID)

	resp.Diagnostics.Append(callSkillReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

func callSkillCreateInlineAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	triggerCondition := ""
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		triggerCondition = data.TriggerCondition.ValueString()
	}

	sdkReq := quantadmingo.NewCreateSkillRequest(
		data.Name.ValueString(),
		data.Content.ValueString(),
		triggerCondition,
	)

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		sdkReq.Description = &desc
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
		sdkReq.Tags = tags
	}

	sdkResp, httpResp, err := r.client.Instance.AISkillsAPI.CreateSkill(r.client.AuthContext, org).
		CreateSkillRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to create AI skill",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to create AI skill", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	diags.Append(mapSkillFromMap(ctx, sdkResp.GetSkill(), org, data)...)
	return
}

func callSkillReadAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AISkillsAPI.GetSkill(r.client.AuthContext, org, data.SkillId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			// Signal "not found" by nulling SkillId — caller handles state removal.
			data.SkillId = types.StringNull()
			return
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read AI skill",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to read AI skill", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Map the typed GetSkill200Response to Terraform model.
	diags.Append(mapGetSkillResponse(ctx, sdkResp, org, data)...)
	return
}

func callSkillUpdateInlineAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	updateReq := quantadmingo.NewUpdateSkillRequest()
	name := data.Name.ValueString()
	updateReq.Name = &name
	content := data.Content.ValueString()
	updateReq.Content = &content

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		updateReq.Description = &desc
	}
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		tc := data.TriggerCondition.ValueString()
		updateReq.TriggerCondition = &tc
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
		updateReq.Tags = tags
	}

	sdkResp, httpResp, err := r.client.Instance.AISkillsAPI.UpdateSkill(r.client.AuthContext, org, data.SkillId.ValueString()).
		UpdateSkillRequest(*updateReq).Execute()
	if err != nil {
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to update AI skill",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to update AI skill", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	diags.Append(mapSkillFromMap(ctx, sdkResp.GetSkill(), org, data)...)
	return
}

func callSkillDeleteAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	_, httpResp, err := r.client.Instance.AISkillsAPI.DeleteSkill(r.client.AuthContext, org, data.SkillId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to delete AI skill",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to delete AI skill", fmt.Sprintf("Error: %s", err.Error()))
		}
	}

	return
}

// mapGetSkillResponse maps the typed GetSkill200Response onto the Terraform model.
// Used for Read where the SDK provides fully typed response.
func mapGetSkillResponse(ctx context.Context, resp *quantadmingo.GetSkill200Response, org string, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	sk := resp.GetSkill()

	data.SkillId = types.StringValue(sk.GetSkillId())
	data.Organisation = types.StringValue(org)
	data.Name = types.StringValue(sk.GetName())

	// description
	if desc, ok := sk.GetDescriptionOk(); ok && desc != nil && *desc != "" {
		data.Description = types.StringValue(*desc)
	} else if data.Description.IsNull() || data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	// tags
	if tags := sk.GetTags(); len(tags) > 0 {
		tl, d := types.ListValueFrom(ctx, types.StringType, tags)
		diags.Append(d...)
		data.Tags = tl
	} else if data.Tags.IsNull() || data.Tags.IsUnknown() {
		data.Tags = types.ListNull(types.StringType)
	}

	// trigger_condition
	if tc, ok := sk.GetTriggerConditionOk(); ok && tc != nil && *tc != "" {
		data.TriggerCondition = types.StringValue(*tc)
	} else if data.TriggerCondition.IsNull() || data.TriggerCondition.IsUnknown() {
		data.TriggerCondition = types.StringNull()
	}

	// content
	if content, ok := sk.GetContentOk(); ok && content != nil && *content != "" {
		data.Content = types.StringValue(*content)
	} else if data.Content.IsNull() || data.Content.IsUnknown() {
		data.Content = types.StringNull()
	}

	// namespace
	if ns, ok := sk.GetNamespaceOk(); ok && ns != nil && *ns != "" {
		data.Namespace = types.StringValue(*ns)
	} else {
		data.Namespace = types.StringNull()
	}

	return
}

// mapSkillFromMap maps a skill map[string]interface{} (from Create/Import/Update/Sync
// SDK responses) onto the Terraform model.
func mapSkillFromMap(ctx context.Context, skillMap map[string]interface{}, org string, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	data.SkillId = optionalStringFromMap(skillMap, "skillId")
	data.Organisation = types.StringValue(org)
	data.Name = optionalStringFromMap(skillMap, "name")

	// description
	if s := optionalStringFromMap(skillMap, "description"); !s.IsNull() {
		data.Description = s
	} else if data.Description.IsNull() || data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	// tags
	if v, ok := skillMap["tags"]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			tags := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					tags = append(tags, s)
				}
			}
			tl, d := types.ListValueFrom(ctx, types.StringType, tags)
			diags.Append(d...)
			data.Tags = tl
		} else {
			data.Tags = types.ListNull(types.StringType)
		}
	} else if data.Tags.IsNull() || data.Tags.IsUnknown() {
		data.Tags = types.ListNull(types.StringType)
	}

	// trigger_condition
	if s := optionalStringFromMap(skillMap, "triggerCondition"); !s.IsNull() {
		data.TriggerCondition = s
	} else if data.TriggerCondition.IsNull() || data.TriggerCondition.IsUnknown() {
		data.TriggerCondition = types.StringNull()
	}

	// content
	if s := optionalStringFromMap(skillMap, "content"); !s.IsNull() {
		data.Content = s
	} else if data.Content.IsNull() || data.Content.IsUnknown() {
		data.Content = types.StringNull()
	}

	// namespace
	if s := optionalStringFromMap(skillMap, "namespace"); !s.IsNull() {
		data.Namespace = s
	} else {
		data.Namespace = types.StringNull()
	}

	return
}

// optionalStringFromMap extracts a string value from a JSON-decoded map.
func optionalStringFromMap(m map[string]interface{}, key string) types.String {
	v, ok := m[key]
	if !ok || v == nil {
		return types.StringNull()
	}
	if s, ok := v.(string); ok && s != "" {
		return types.StringValue(s)
	}
	return types.StringNull()
}
