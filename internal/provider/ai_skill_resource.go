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

	body := map[string]interface{}{
		"name":    data.Name.ValueString(),
		"content": data.Content.ValueString(),
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		body["triggerCondition"] = data.TriggerCondition.ValueString()
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
		body["tags"] = tags
	}

	apiResp, err := doAIRequest(r.client, http.MethodPost,
		fmt.Sprintf("/api/v3/organisations/%s/ai/skills", org), body)
	if err != nil {
		diags.AddError("Unable to create AI skill", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to create AI skill",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseSkillResponse(ctx, apiResp, org, data)...)
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

	sourceMap := map[string]interface{}{
		"type":    src.Type.ValueString(),
		"version": src.Version.ValueString(),
	}
	if !src.Repo.IsNull() && !src.Repo.IsUnknown() {
		sourceMap["repo"] = src.Repo.ValueString()
	}
	if !src.Path.IsNull() && !src.Path.IsUnknown() {
		sourceMap["path"] = src.Path.ValueString()
	}
	if !src.Url.IsNull() && !src.Url.IsUnknown() {
		sourceMap["url"] = src.Url.ValueString()
	}

	body := map[string]interface{}{
		"source": sourceMap,
	}

	apiResp, err := doAIRequest(r.client, http.MethodPost,
		fmt.Sprintf("/api/v3/organisations/%s/ai/skills/import", org), body)
	if err != nil {
		diags.AddError("Unable to import AI skill", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to import AI skill",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseSkillResponse(ctx, apiResp, org, data)...)
	if diags.HasError() {
		return
	}

	// After import, PUT to set name/description/tags if provided.
	needsUpdate := false
	updateBody := map[string]interface{}{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		updateBody["name"] = data.Name.ValueString()
		needsUpdate = true
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		updateBody["description"] = data.Description.ValueString()
		needsUpdate = true
	}
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		updateBody["triggerCondition"] = data.TriggerCondition.ValueString()
		needsUpdate = true
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
		updateBody["tags"] = tags
		needsUpdate = true
	}

	if needsUpdate {
		putResp, err := doAIRequest(r.client, http.MethodPut,
			fmt.Sprintf("/api/v3/organisations/%s/ai/skills/%s", org, data.Id.ValueString()), updateBody)
		if err != nil {
			diags.AddError("Unable to update AI skill after import", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
		defer putResp.Body.Close()

		if putResp.StatusCode >= 300 {
			respBody, _ := io.ReadAll(putResp.Body)
			diags.AddError("Unable to update AI skill after import",
				fmt.Sprintf("API returned %d: %s", putResp.StatusCode, string(respBody)))
			return
		}

		diags.Append(parseSkillResponse(ctx, putResp, org, data)...)
	}

	return
}

func callSkillReadAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	apiResp, err := doAIRequest(r.client, http.MethodGet,
		fmt.Sprintf("/api/v3/organisations/%s/ai/skills/%s", org, data.Id.ValueString()), nil)
	if err != nil {
		diags.AddError("Unable to read AI skill", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == http.StatusNotFound {
		diags.AddError("AI skill not found",
			fmt.Sprintf("Skill '%s' not found.", data.Id.ValueString()))
		return
	}

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to read AI skill",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseSkillResponse(ctx, apiResp, org, data)...)
	return
}

func callSkillUpdateInlineAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	body := map[string]interface{}{
		"name":    data.Name.ValueString(),
		"content": data.Content.ValueString(),
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		body["triggerCondition"] = data.TriggerCondition.ValueString()
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
		body["tags"] = tags
	}

	apiResp, err := doAIRequest(r.client, http.MethodPut,
		fmt.Sprintf("/api/v3/organisations/%s/ai/skills/%s", org, data.Id.ValueString()), body)
	if err != nil {
		diags.AddError("Unable to update AI skill", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to update AI skill",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseSkillResponse(ctx, apiResp, org, data)...)
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
		syncResp, err := doAIRequest(r.client, http.MethodPost,
			fmt.Sprintf("/api/v3/organisations/%s/ai/skills/%s/sync", org, data.Id.ValueString()), nil)
		if err != nil {
			diags.AddError("Unable to sync AI skill", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
		defer syncResp.Body.Close()

		if syncResp.StatusCode >= 300 {
			respBody, _ := io.ReadAll(syncResp.Body)
			diags.AddError("Unable to sync AI skill",
				fmt.Sprintf("API returned %d: %s", syncResp.StatusCode, string(respBody)))
			return
		}

		diags.Append(parseSkillResponse(ctx, syncResp, org, data)...)
		if diags.HasError() {
			return
		}
	}

	// Also PUT to update name/description/tags if provided.
	updateBody := map[string]interface{}{}
	needsUpdate := false

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		updateBody["name"] = data.Name.ValueString()
		needsUpdate = true
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		updateBody["description"] = data.Description.ValueString()
		needsUpdate = true
	}
	if !data.TriggerCondition.IsNull() && !data.TriggerCondition.IsUnknown() {
		updateBody["triggerCondition"] = data.TriggerCondition.ValueString()
		needsUpdate = true
	}
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		diags.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return
		}
		updateBody["tags"] = tags
		needsUpdate = true
	}

	if needsUpdate {
		putResp, err := doAIRequest(r.client, http.MethodPut,
			fmt.Sprintf("/api/v3/organisations/%s/ai/skills/%s", org, data.Id.ValueString()), updateBody)
		if err != nil {
			diags.AddError("Unable to update AI skill", fmt.Sprintf("Error: %s", err.Error()))
			return
		}
		defer putResp.Body.Close()

		if putResp.StatusCode >= 300 {
			respBody, _ := io.ReadAll(putResp.Body)
			diags.AddError("Unable to update AI skill",
				fmt.Sprintf("API returned %d: %s", putResp.StatusCode, string(respBody)))
			return
		}

		diags.Append(parseSkillResponse(ctx, putResp, org, data)...)
	} else if !versionChanged {
		// Nothing changed, just read current state.
		diags.Append(callSkillReadAPI(ctx, r, data)...)
	}

	return
}

func callSkillDeleteAPI(ctx context.Context, r *aiSkillResource, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	apiResp, err := doAIRequest(r.client, http.MethodDelete,
		fmt.Sprintf("/api/v3/organisations/%s/ai/skills/%s", org, data.Id.ValueString()), nil)
	if err != nil {
		diags.AddError("Unable to delete AI skill", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == http.StatusNotFound {
		return // Already deleted
	}

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to delete AI skill",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
	}

	return
}

// parseSkillResponse reads the JSON response from the skills API and maps it
// back onto the Terraform model.
//
// Expected response shape:
//
//	{
//	  "skill": {
//	    "id": "uuid",
//	    "name": "my-skill",
//	    "description": "...",
//	    "tags": ["tag1"],
//	    "triggerCondition": "...",
//	    "content": "...",
//	    "source": { "type": "github", "repo": "...", "path": "...", "version": "..." },
//	    "namespace": "org/skill-name",
//	    "installedAt": "2026-03-30T...",
//	    "updatedAt": "2026-03-30T..."
//	  }
//	}
func parseSkillResponse(ctx context.Context, apiResp *http.Response, org string, data *resource_ai_skill.AiSkillModel) (diags diag.Diagnostics) {
	respBody, err := io.ReadAll(apiResp.Body)
	if err != nil {
		diags.AddError("Unable to read AI skill response", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	var envelope struct {
		Skill struct {
			Id               string                 `json:"id"`
			Name             string                 `json:"name"`
			Description      *string                `json:"description"`
			Tags             []string               `json:"tags"`
			TriggerCondition *string                `json:"triggerCondition"`
			Content          *string                `json:"content"`
			Source           map[string]interface{}  `json:"source"`
			Namespace        *string                `json:"namespace"`
			InstalledAt      *string                `json:"installedAt"`
			UpdatedAt        *string                `json:"updatedAt"`
		} `json:"skill"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		diags.AddError("Unable to parse AI skill response",
			fmt.Sprintf("Error: %s\nBody: %s", err.Error(), string(respBody)))
		return
	}

	sk := envelope.Skill

	data.Id = types.StringValue(sk.Id)
	data.Organization = types.StringValue(org)
	data.Name = types.StringValue(sk.Name)

	// description
	if sk.Description != nil && *sk.Description != "" {
		data.Description = types.StringValue(*sk.Description)
	} else if data.Description.IsNull() || data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	// tags
	if len(sk.Tags) > 0 {
		tl, d := types.ListValueFrom(ctx, types.StringType, sk.Tags)
		diags.Append(d...)
		data.Tags = tl
	} else if data.Tags.IsNull() || data.Tags.IsUnknown() {
		data.Tags = types.ListNull(types.StringType)
	}

	// trigger_condition
	if sk.TriggerCondition != nil && *sk.TriggerCondition != "" {
		data.TriggerCondition = types.StringValue(*sk.TriggerCondition)
	} else if data.TriggerCondition.IsNull() || data.TriggerCondition.IsUnknown() {
		data.TriggerCondition = types.StringNull()
	}

	// content
	if sk.Content != nil && *sk.Content != "" {
		data.Content = types.StringValue(*sk.Content)
	} else if data.Content.IsNull() || data.Content.IsUnknown() {
		data.Content = types.StringNull()
	}

	// source
	if sk.Source != nil && len(sk.Source) > 0 {
		srcModel := resource_ai_skill.SourceModel{
			Type:    optionalStringFromMap(sk.Source, "type"),
			Repo:    optionalStringFromMap(sk.Source, "repo"),
			Path:    optionalStringFromMap(sk.Source, "path"),
			Url:     optionalStringFromMap(sk.Source, "url"),
			Version: optionalStringFromMap(sk.Source, "version"),
		}
		objVal, d := types.ObjectValueFrom(ctx, resource_ai_skill.SourceAttrTypes(), srcModel)
		diags.Append(d...)
		data.Source = objVal
	} else if data.Source.IsNull() || data.Source.IsUnknown() {
		data.Source = types.ObjectNull(resource_ai_skill.SourceAttrTypes())
	}

	// namespace
	if sk.Namespace != nil && *sk.Namespace != "" {
		data.Namespace = types.StringValue(*sk.Namespace)
	} else {
		data.Namespace = types.StringNull()
	}

	// installed_at
	if sk.InstalledAt != nil && *sk.InstalledAt != "" {
		data.InstalledAt = types.StringValue(*sk.InstalledAt)
	} else {
		data.InstalledAt = types.StringNull()
	}

	// updated_at
	if sk.UpdatedAt != nil && *sk.UpdatedAt != "" {
		data.UpdatedAt = types.StringValue(*sk.UpdatedAt)
	} else {
		data.UpdatedAt = types.StringNull()
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
