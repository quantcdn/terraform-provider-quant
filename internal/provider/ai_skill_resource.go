package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

// ValidateConfig checks that exactly one of content or source is set.
func (r *aiSkillResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data resource_ai_skill.AiSkillModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasContent := !data.Content.IsNull() && !data.Content.IsUnknown()
	hasSource := !data.Source.IsNull() && !data.Source.IsUnknown()

	if hasContent && hasSource {
		resp.Diagnostics.AddError(
			"Conflicting configuration",
			"Only one of \"content\" or \"source\" may be specified, not both.",
		)
	}

	if !hasContent && !hasSource {
		resp.Diagnostics.AddError(
			"Missing required configuration",
			"Exactly one of \"content\" or \"source\" must be specified.",
		)
	}
}

func (r *aiSkillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_skill.AiSkillModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		// Inline skill — POST to /skills
		resp.Diagnostics.Append(callSkillCreateInlineAPI(ctx, r, &data)...)
	} else {
		// Import from source — POST to /skills/import, then PUT metadata
		resp.Diagnostics.Append(callSkillImportAPI(ctx, r, &data)...)
	}

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
	if data.Id.IsNull() {
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

	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		// Inline skill — PUT to update fields
		resp.Diagnostics.Append(callSkillUpdateInlineAPI(ctx, r, &data)...)
	} else {
		// Import source — check if version changed and sync
		var state resource_ai_skill.AiSkillModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(callSkillUpdateImportAPI(ctx, r, &data, &state)...)
	}

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
	data.Id = types.StringValue(req.ID)

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

func callSkillImportAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	// Extract source model
	var src resource_ai_skill.SourceModel
	diags.Append(data.Source.As(ctx, &src, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return
	}

	sdkSource := quantadmingo.NewImportSkillRequestSource(src.Type.ValueString())
	if !src.Repo.IsNull() && !src.Repo.IsUnknown() {
		repo := src.Repo.ValueString()
		sdkSource.Repo = &repo
	}
	if !src.Path.IsNull() && !src.Path.IsUnknown() {
		path := src.Path.ValueString()
		sdkSource.Path = &path
	}
	if !src.Url.IsNull() && !src.Url.IsUnknown() {
		url := src.Url.ValueString()
		sdkSource.Url = &url
	}
	if !src.Version.IsNull() && !src.Version.IsUnknown() {
		version := src.Version.ValueString()
		sdkSource.Version = &version
	}

	sdkReq := quantadmingo.NewImportSkillRequest(*sdkSource)

	importResp, httpResp, err := r.client.Instance.AISkillsAPI.ImportSkill(r.client.AuthContext, org).
		ImportSkillRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to import AI skill",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to import AI skill", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	diags.Append(mapSkillFromMap(ctx, importResp.GetSkill(), org, data)...)
	if diags.HasError() {
		return
	}

	// After import, PUT to set name/description/tags if provided.
	updateReq, updateDiags := buildSkillMetadataUpdate(ctx, data)
	diags.Append(updateDiags...)
	if diags.HasError() {
		return
	}

	if updateReq != nil {
		putResp, putHttpResp, putErr := r.client.Instance.AISkillsAPI.UpdateSkill(r.client.AuthContext, org, data.Id.ValueString()).
			UpdateSkillRequest(*updateReq).Execute()
		if putErr != nil {
			if putHttpResp != nil {
				respBody, _ := io.ReadAll(putHttpResp.Body)
				diags.AddError("Unable to update AI skill after import",
					fmt.Sprintf("API returned %d: %s", putHttpResp.StatusCode, string(respBody)))
			} else {
				diags.AddError("Unable to update AI skill after import", fmt.Sprintf("Error: %s", putErr.Error()))
			}
			return
		}

		diags.Append(mapSkillFromMap(ctx, putResp.GetSkill(), org, data)...)
	}

	return
}

func callSkillReadAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AISkillsAPI.GetSkill(r.client.AuthContext, org, data.Id.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			// Signal "not found" by nulling Id — caller handles state removal.
			data.Id = types.StringNull()
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

	sdkResp, httpResp, err := r.client.Instance.AISkillsAPI.UpdateSkill(r.client.AuthContext, org, data.Id.ValueString()).
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

func callSkillUpdateImportAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel, state *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	// Carry forward the ID from state.
	data.Id = state.Id

	// Check if the version changed — if so, sync.
	var newSrc, oldSrc resource_ai_skill.SourceModel
	diags.Append(data.Source.As(ctx, &newSrc, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return
	}

	versionChanged := true
	if !state.Source.IsNull() && !state.Source.IsUnknown() {
		diags.Append(state.Source.As(ctx, &oldSrc, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}
		versionChanged = newSrc.Version.ValueString() != oldSrc.Version.ValueString()
	}

	if versionChanged {
		syncSdkResp, syncHttpResp, syncErr := r.client.Instance.AISkillsAPI.SyncSkill(r.client.AuthContext, org, data.Id.ValueString()).Execute()
		if syncErr != nil {
			if syncHttpResp != nil {
				respBody, _ := io.ReadAll(syncHttpResp.Body)
				diags.AddError("Unable to sync AI skill",
					fmt.Sprintf("API returned %d: %s", syncHttpResp.StatusCode, string(respBody)))
			} else {
				diags.AddError("Unable to sync AI skill", fmt.Sprintf("Error: %s", syncErr.Error()))
			}
			return
		}

		diags.Append(mapSkillFromMap(ctx, syncSdkResp.GetSkill(), org, data)...)
		if diags.HasError() {
			return
		}
	}

	// Also PUT to update name/description/tags if provided.
	updateReq, updateDiags := buildSkillMetadataUpdate(ctx, data)
	diags.Append(updateDiags...)
	if diags.HasError() {
		return
	}

	if updateReq != nil {
		putResp, putHttpResp, putErr := r.client.Instance.AISkillsAPI.UpdateSkill(r.client.AuthContext, org, data.Id.ValueString()).
			UpdateSkillRequest(*updateReq).Execute()
		if putErr != nil {
			if putHttpResp != nil {
				respBody, _ := io.ReadAll(putHttpResp.Body)
				diags.AddError("Unable to update AI skill",
					fmt.Sprintf("API returned %d: %s", putHttpResp.StatusCode, string(respBody)))
			} else {
				diags.AddError("Unable to update AI skill", fmt.Sprintf("Error: %s", putErr.Error()))
			}
			return
		}

		diags.Append(mapSkillFromMap(ctx, putResp.GetSkill(), org, data)...)
	} else if !versionChanged {
		// Nothing changed, just read current state.
		diags.Append(callSkillReadAPI(ctx, r, data)...)
	}

	return
}

func callSkillDeleteAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	_, httpResp, err := r.client.Instance.AISkillsAPI.DeleteSkill(r.client.AuthContext, org, data.Id.ValueString()).Execute()
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

	data.Id = types.StringValue(sk.GetSkillId())
	data.Organization = types.StringValue(org)
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

	// source
	if src := sk.GetSource(); len(src) > 0 {
		srcModel := resource_ai_skill.SourceModel{
			Type:    optionalStringFromMap(src, "type"),
			Repo:    optionalStringFromMap(src, "repo"),
			Path:    optionalStringFromMap(src, "path"),
			Url:     optionalStringFromMap(src, "url"),
			Version: optionalStringFromMap(src, "version"),
		}
		objVal, d := types.ObjectValueFrom(ctx, resource_ai_skill.SourceAttrTypes(), srcModel)
		diags.Append(d...)
		data.Source = objVal
	} else if data.Source.IsNull() || data.Source.IsUnknown() {
		data.Source = types.ObjectNull(resource_ai_skill.SourceAttrTypes())
	}

	// namespace
	if ns, ok := sk.GetNamespaceOk(); ok && ns != nil && *ns != "" {
		data.Namespace = types.StringValue(*ns)
	} else {
		data.Namespace = types.StringNull()
	}

	// installed_at
	if installedAt, ok := sk.GetInstalledAtOk(); ok && installedAt != nil {
		data.InstalledAt = types.StringValue(installedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.InstalledAt = types.StringNull()
	}

	// updated_at
	if updatedAt, ok := sk.GetUpdatedAtOk(); ok && updatedAt != nil {
		data.UpdatedAt = types.StringValue(updatedAt.Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.UpdatedAt = types.StringNull()
	}

	return
}

// mapSkillFromMap maps a skill map[string]interface{} (from Create/Import/Update/Sync
// SDK responses) onto the Terraform model.
func mapSkillFromMap(ctx context.Context, skillMap map[string]interface{}, org string, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	data.Id = optionalStringFromMap(skillMap, "skillId")
	data.Organization = types.StringValue(org)
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

	// source
	if v, ok := skillMap["source"]; ok && v != nil {
		if srcMap, ok := v.(map[string]interface{}); ok && len(srcMap) > 0 {
			srcModel := resource_ai_skill.SourceModel{
				Type:    optionalStringFromMap(srcMap, "type"),
				Repo:    optionalStringFromMap(srcMap, "repo"),
				Path:    optionalStringFromMap(srcMap, "path"),
				Url:     optionalStringFromMap(srcMap, "url"),
				Version: optionalStringFromMap(srcMap, "version"),
			}
			objVal, d := types.ObjectValueFrom(ctx, resource_ai_skill.SourceAttrTypes(), srcModel)
			diags.Append(d...)
			data.Source = objVal
		} else if data.Source.IsNull() || data.Source.IsUnknown() {
			data.Source = types.ObjectNull(resource_ai_skill.SourceAttrTypes())
		}
	} else if data.Source.IsNull() || data.Source.IsUnknown() {
		data.Source = types.ObjectNull(resource_ai_skill.SourceAttrTypes())
	}

	// namespace
	if s := optionalStringFromMap(skillMap, "namespace"); !s.IsNull() {
		data.Namespace = s
	} else {
		data.Namespace = types.StringNull()
	}

	// installed_at
	if s := optionalStringFromMap(skillMap, "installedAt"); !s.IsNull() {
		data.InstalledAt = s
	} else {
		data.InstalledAt = types.StringNull()
	}

	// updated_at
	if s := optionalStringFromMap(skillMap, "updatedAt"); !s.IsNull() {
		data.UpdatedAt = s
	} else {
		data.UpdatedAt = types.StringNull()
	}

	return
}

// buildSkillMetadataUpdate builds an UpdateSkillRequest from the Terraform model's
// name, description, tags, and trigger_condition fields. Returns nil if no update
// is needed (all optional fields are null/unknown).
func buildSkillMetadataUpdate(ctx context.Context, data *resource_ai_skill.AiSkillModel) (*quantadmingo.UpdateSkillRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	updateReq := quantadmingo.NewUpdateSkillRequest()
	needsUpdate := false

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		updateReq.Name = &name
		needsUpdate = true
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		updateReq.Description = &desc
		needsUpdate = true
	}
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		tc := data.TriggerCondition.ValueString()
		updateReq.TriggerCondition = &tc
		needsUpdate = true
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return nil, diags
		}
		updateReq.Tags = tags
		needsUpdate = true
	}

	if !needsUpdate {
		return nil, diags
	}
	return updateReq, diags
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
