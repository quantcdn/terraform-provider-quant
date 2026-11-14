package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_auto_mode"
)

var (
	_ resource.Resource                = (*aiAutoModeResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiAutoModeResource)(nil)
	_ resource.ResourceWithImportState = (*aiAutoModeResource)(nil)
)

// aiAutoModeGovernanceFirst is part of the 409 the API returns when the
// organisation has no governance settings yet.
const aiAutoModeGovernanceFirst = "governance settings first"

// aiAutoModeModel is the generated model plus the computed audit fields.
type aiAutoModeModel struct {
	resource_ai_auto_mode.AiAutoModeModel
	UpdatedAt types.String `tfsdk:"updated_at"`
	UpdatedBy types.String `tfsdk:"updated_by"`
}

// aiAutoModeRoles is the plain Go shape of group.roles.
type aiAutoModeRoles struct {
	Deep     []string `tfsdk:"deep"`
	Fast     []string `tfsdk:"fast"`
	Standard []string `tfsdk:"standard"`
}

func NewAiAutoModeResource() resource.Resource {
	return &aiAutoModeResource{}
}

type aiAutoModeResource struct {
	client *client.Client
}

func (r *aiAutoModeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_auto_mode"
}

func (r *aiAutoModeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_ai_auto_mode.AiAutoModeResourceSchema(ctx)
	s.Description = "The auto-mode setting of an organisation. One setting exists for each organisation."
	s.MarkdownDescription = s.Description
	s.Attributes["updated_at"] = schema.StringAttribute{
		Computed:    true,
		Description: "When the setting was last saved.",
	}
	s.Attributes["updated_by"] = schema.StringAttribute{
		Computed:    true,
		Description: "Who last saved the setting.",
	}
	addUseStateForUnknown(s.Attributes)
	// These change on every save, so they must not carry state forward.
	clearPlanModifiers(s.Attributes, "updated_at", "updated_by")
	makeAutoModeRevisionComputed(s.Attributes)
	resp.Schema = s
}

// makeAutoModeRevisionComputed makes group.revision computed only. The API
// bumps the revision on every save, so a configured value can never match
// the state after apply.
func makeAutoModeRevisionComputed(attrs map[string]schema.Attribute) {
	group, ok := attrs["group"].(schema.SingleNestedAttribute)
	if !ok {
		return
	}
	rev, ok := group.Attributes["revision"].(schema.Int64Attribute)
	if !ok {
		return
	}
	rev.Optional = false
	rev.Computed = true
	rev.Validators = nil
	rev.Description = "Revision of the setting. The provider sends it on update; a stale value returns 409."
	rev.MarkdownDescription = rev.Description
	group.Attributes["revision"] = rev
	attrs["group"] = group
}

func (r *aiAutoModeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *aiAutoModeResource) getOrg(data *aiAutoModeModel) string {
	return orgOr(data.Organisation, r.client.Organization)
}

// Create sends no revision. A 409 means a setting already exists.
func (r *aiAutoModeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data aiAutoModeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(callAutoModePutAPI(ctx, r, &data, nil)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiAutoModeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data aiAutoModeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, diags := callAutoModeReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Deleted outside Terraform: remove from state so the next plan creates it.
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update sends the revision from state, which the last Read refreshed.
func (r *aiAutoModeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state aiAutoModeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var revision *int32
	if isKnown(state.Group.Revision) {
		v := int32(state.Group.Revision.ValueInt64())
		revision = &v
	}
	resp.Diagnostics.Append(callAutoModePutAPI(ctx, r, &data, revision)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete resets the organisation to "not configured".
func (r *aiAutoModeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data aiAutoModeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.Instance.AIAutoModeAPI.DeleteAutoMode(r.client.AuthContext, r.getOrg(&data)).Execute()
	if err == nil || autoModeStatus(httpResp) == http.StatusNotFound {
		return
	}
	resp.Diagnostics.AddError("Unable to delete AI auto-mode setting", autoModeErrorText(httpResp, err))
}

// ImportState takes the organisation machine name as the ID. An empty ID
// uses the provider's organisation.
func (r *aiAutoModeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data aiAutoModeModel
	data.Organisation = types.StringValue(req.ID)
	found, diags := callAutoModeReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("No AI auto-mode setting to import",
			fmt.Sprintf("Organisation %q has no auto-mode setting. Remove the import and let terraform apply create it.", r.getOrg(&data)))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

func callAutoModePutAPI(ctx context.Context, r *aiAutoModeResource, data *aiAutoModeModel, revision *int32) diag.Diagnostics {
	org := r.getOrg(data)
	input, diags := autoModeInputFromModel(ctx, data, revision)
	if diags.HasError() {
		return diags
	}
	out, httpResp, err := r.client.Instance.AIAutoModeAPI.UpdateAutoMode(r.client.AuthContext, org).AIAutoModeInput(input).Execute()
	if err != nil {
		diags.AddError("Unable to save AI auto-mode setting", autoModePutErrorText(httpResp, err, org, revision == nil))
		return diags
	}
	if out == nil || out.AutoMode == nil {
		diags.AddError("Unable to save AI auto-mode setting", "The API response has no autoMode.")
		return diags
	}
	diags.Append(autoModeToModel(ctx, out.AutoMode, org, data)...)
	return diags
}

// callAutoModeReadAPI returns false when the organisation has no setting.
func callAutoModeReadAPI(ctx context.Context, r *aiAutoModeResource, data *aiAutoModeModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	org := r.getOrg(data)
	out, httpResp, err := r.client.Instance.AIAutoModeAPI.GetAutoMode(r.client.AuthContext, org).Execute()
	if err != nil {
		diags.AddError("Unable to read AI auto-mode setting", autoModeErrorText(httpResp, err))
		return false, diags
	}
	cfg := out.AutoMode.Get()
	if cfg == nil {
		return false, diags
	}
	diags.Append(autoModeToModel(ctx, cfg, org, data)...)
	return true, diags
}

// autoModeInputFromModel builds the PUT body. Unknown or null optional
// values are left out of the body.
func autoModeInputFromModel(ctx context.Context, data *aiAutoModeModel, revision *int32) (quantadmingo.AIAutoModeInput, diag.Diagnostics) {
	var roles aiAutoModeRoles
	diags := data.Group.Roles.As(ctx, &roles, basetypes.ObjectAsOptions{})
	group := quantadmingo.NewAIAutoModeInputGroup(*quantadmingo.NewAIAutoModeConfigGroupRoles(
		nonNil(roles.Fast), nonNil(roles.Standard), nonNil(roles.Deep)))
	group.Revision = revision
	input := quantadmingo.NewAIAutoModeInput(
		data.Enabled.ValueBool(),
		*group,
		int32(data.WatcherIntervalMinutes.ValueInt64()),
		data.Enforcement.ValueString(),
		data.AutonomousDefault.ValueBool(),
	)
	if isKnown(data.OrchestratorModel) {
		input.SetOrchestratorModel(data.OrchestratorModel.ValueString())
	}
	if isKnown(data.MaxParallelTasks) {
		input.SetMaxParallelTasks(int32(data.MaxParallelTasks.ValueInt64()))
	}
	return *input, diags
}

// autoModeToModel maps the API setting onto the model. Role order is kept.
func autoModeToModel(ctx context.Context, cfg *quantadmingo.AIAutoModeConfig, org string, data *aiAutoModeModel) diag.Diagnostics {
	group, diags := autoModeGroupValue(ctx, cfg.Group)
	data.Group = group
	data.Organisation = types.StringValue(org)
	data.Enabled = types.BoolValue(cfg.Enabled)
	data.OrchestratorModel = stringOrNull(cfg.OrchestratorModel)
	data.WatcherIntervalMinutes = types.Int64Value(int64(cfg.WatcherIntervalMinutes))
	data.MaxParallelTasks = types.Int64Null()
	if cfg.MaxParallelTasks != nil {
		data.MaxParallelTasks = types.Int64Value(int64(*cfg.MaxParallelTasks))
	}
	data.Enforcement = types.StringValue(cfg.Enforcement)
	data.AutonomousDefault = types.BoolValue(cfg.AutonomousDefault)
	data.UpdatedAt = stringOrNull(cfg.UpdatedAt)
	data.UpdatedBy = types.StringNull()
	if cfg.UpdatedBy != nil {
		data.UpdatedBy = stringOrNull(*cfg.UpdatedBy)
	}
	return diags
}

func autoModeGroupValue(ctx context.Context, g quantadmingo.AIAutoModeConfigGroup) (resource_ai_auto_mode.GroupValue, diag.Diagnostics) {
	roles, diags := types.ObjectValueFrom(ctx, resource_ai_auto_mode.RolesValue{}.AttributeTypes(ctx), aiAutoModeRoles{
		Fast:     nonNil(g.Roles.Fast),
		Standard: nonNil(g.Roles.Standard),
		Deep:     nonNil(g.Roles.Deep),
	})
	if diags.HasError() {
		return resource_ai_auto_mode.NewGroupValueNull(), diags
	}
	group, d := resource_ai_auto_mode.NewGroupValue(resource_ai_auto_mode.GroupValue{}.AttributeTypes(ctx), map[string]attr.Value{
		"revision": types.Int64Value(int64(g.Revision)),
		"roles":    roles,
	})
	diags.Append(d...)
	return group, diags
}

// nonNil keeps an empty role list as [] rather than null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// aiAutoModeError is the error body of the auto-mode API. It is decoded by
// hand so that a partial eligible object still shows.
type aiAutoModeError struct {
	Error    string `json:"error"`
	Eligible *struct {
		Orchestrator *aiModelEligibility `json:"orchestrator"`
		Roles        struct {
			Fast     []aiModelEligibility `json:"fast"`
			Standard []aiModelEligibility `json:"standard"`
			Deep     []aiModelEligibility `json:"deep"`
		} `json:"roles"`
	} `json:"eligible"`
}

// aiModelEligibility is ModelEligibility without the SDK's strict decoder.
type aiModelEligibility struct {
	ModelID    string `json:"modelId"`
	ResolvedID string `json:"resolvedId"`
	Eligible   bool   `json:"eligible"`
	Reason     string `json:"reason"`
}

func autoModeStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// readAutoModeError decodes the error body. The SDK leaves the body readable.
func readAutoModeError(resp *http.Response) (aiAutoModeError, string) {
	var e aiAutoModeError
	if resp == nil || resp.Body == nil {
		return e, ""
	}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &e)
	return e, string(body)
}

func autoModeErrorText(resp *http.Response, err error) string {
	if resp == nil {
		return err.Error()
	}
	e, raw := readAutoModeError(resp)
	if e.Error != "" {
		return fmt.Sprintf("API returned %d: %s", resp.StatusCode, e.Error)
	}
	return fmt.Sprintf("API returned %d: %s", resp.StatusCode, raw)
}

func autoModePutErrorText(resp *http.Response, err error, org string, create bool) string {
	if resp == nil {
		return err.Error()
	}
	e, raw := readAutoModeError(resp)
	msg := e.Error
	if msg == "" {
		msg = raw
	}
	switch {
	case resp.StatusCode == http.StatusBadRequest:
		return msg + ineligibleModelsText(e)
	case resp.StatusCode == http.StatusConflict && strings.Contains(msg, aiAutoModeGovernanceFirst):
		return fmt.Sprintf("%s\n\nApply quant_ai_governance for organisation %q first, and add depends_on = [quant_ai_governance.<name>] to this quant_ai_auto_mode resource.", msg, org)
	case resp.StatusCode == http.StatusConflict && create:
		return fmt.Sprintf("%s\n\nOrganisation %q already has an auto-mode setting. Import it instead of creating it:\n\n  terraform import quant_ai_auto_mode.<name> %s", msg, org, org)
	case resp.StatusCode == http.StatusConflict:
		return msg + "\n\nThe setting changed outside Terraform after the last refresh. Run terraform apply again."
	}
	return fmt.Sprintf("API returned %d: %s", resp.StatusCode, msg)
}

// ineligibleModelsText lists every ineligible model with its reason.
func ineligibleModelsText(e aiAutoModeError) string {
	if e.Eligible == nil {
		return ""
	}
	var lines []string
	if e.Eligible.Orchestrator != nil {
		lines = appendIneligible(lines, "orchestrator_model", []aiModelEligibility{*e.Eligible.Orchestrator})
	}
	lines = appendIneligible(lines, "group.roles.fast", e.Eligible.Roles.Fast)
	lines = appendIneligible(lines, "group.roles.standard", e.Eligible.Roles.Standard)
	lines = appendIneligible(lines, "group.roles.deep", e.Eligible.Roles.Deep)
	if len(lines) == 0 {
		return ""
	}
	return "\n\nIneligible models:\n" + strings.Join(lines, "\n")
}

func appendIneligible(lines []string, field string, models []aiModelEligibility) []string {
	for _, m := range models {
		if m.Eligible {
			continue
		}
		reason := m.Reason
		if reason == "" {
			reason = "no reason given"
		}
		lines = append(lines, fmt.Sprintf("  - %s: %s (%s): %s", field, m.ModelID, m.ResolvedID, reason))
	}
	return lines
}
