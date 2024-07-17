package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_project"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapiclient "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = (*projectResource)(nil)
	_ resource.ResourceWithConfigure = (*projectResource)(nil)
	_ resource.ResourceWithImportState = (*projectResource)(nil)
)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *client.Client
}

func (r *projectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_project.ProjectResourceSchema(ctx)
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *internal.Client, got: %T. Please report this issue to the provider developers", req.ProviderData),
		)
	}

	r.client = client
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_project.ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callProjectCreateAPI(ctx, r, &data)...)

	if (resp.Diagnostics.HasError()) {
		return
	}

	// Read the API results back into the model for Terraform state.
	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_project.ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_project.ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var stateData resource_project.ProjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	data.MachineName = stateData.MachineName

	// Update API call logic
	resp.Diagnostics.Append(callProjectUpdateAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_project.ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callProjectDeleteAPI(ctx, r, &data)...)
}

// Import state for a given machine name.
func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_project.ProjectModel
	data.MachineName = types.StringValue(req.ID)

	// Read API call logic
	resp.Diagnostics.Append(callProjectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create project request.
func callProjectCreateAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	req := *openapiclient.NewProjectRequestWithDefaults()

	if project.Name.IsNull() || project.Name.IsUnknown() {
		diags.AddAttributeError(
			path.Root("name"),
			"Missing project.name attribute",
			"Cannot create a project without a name.",
		)
		return
	}

	if project.BasicAuthUsername.IsNull() && !project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication username",
			"To enable basic authentication both username and password must be provided.",
		)
		return
	}

	if !project.BasicAuthUsername.IsNull() && project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication password",
			"To enable basic authentication both username and password must be provided.",
		)
		return
	}

	req.SetName(project.Name.ValueString())
	req.SetAllowQueryParams(project.AllowQueryParams.ValueBool())
	req.SetBasicAuthPassword(project.BasicAuthPassword.ValueString())
	req.SetBasicAuthUsername(project.BasicAuthUsername.ValueString())
	req.SetBasicAuthPreviewOnly(project.BasicAuthPreviewOnly.ValueString())

	if project.Region.IsNull() || project.Region.IsUnknown() {
		project.Region = types.StringValue("au")
	}

	req.SetRegion(project.Region.ValueString())

	res, _, err := r.client.Instance.ProjectsAPI.ProjectsCreate(r.client.AuthContext, r.client.Organization).ProjectRequest(req).Execute()

	if err != nil {
		diags.AddError(
			"Unable to add the project",
			fmt.Sprintf("Error: project exists with this name"),
		)
		return
	}

	project.MachineName = types.StringValue(res.GetMachineName())
	return
}

func callProjectUpdateAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		diags.AddAttributeError(
			path.Root("machine_name"),
			"Missing project.machine_name attribute",
			"To read project information the project machine name needs to be known, plese import the terraform state.",
		)
		return
	}

	org := r.client.Organization
	req := *openapiclient.NewProjectRequestUpdateWithDefaults()

	if project.BasicAuthUsername.IsNull() && !project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication username",
			"To enable basic authentication both username and password must be provided.",
		)
		return diags
	}

	if !project.BasicAuthUsername.IsNull() && project.BasicAuthPassword.IsNull() {
		diags.AddError(
			"Missing basic authentication password",
			"To enable basic authentication both username and password must be provided.",
		)
		return diags
	}

	req.SetName(project.Name.ValueString())
	req.SetAllowQueryParams(project.AllowQueryParams.ValueBool())
	req.SetBasicAuthPassword(project.BasicAuthPassword.ValueString())
	req.SetBasicAuthUsername(project.BasicAuthUsername.ValueString())
	req.SetBasicAuthPreviewOnly(project.BasicAuthPreviewOnly.ValueString())

	api := r.client.Instance.ProjectsAPI.ProjectsUpdate(r.client.AuthContext, org, project.MachineName.ValueString())
	_, _, err := api.ProjectRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError("Unable to update project", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func callProjectReadAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		diags.AddError(
			"Unable to read project",
			"The project machine name is unknown, this may indicate an issue with your local state.",
		)
		return
	}

	org := r.client.Organization
	api, _, err := r.client.Instance.ProjectsAPI.ProjectsRead(r.client.AuthContext, org, project.MachineName.ValueString()).Execute()

	if err != nil {
		diags.Append(diag.NewErrorDiagnostic(
			"Unable to read project data from API",
			fmt.Sprintf("There was an issue with the request when reqesting project information form the API, please check the error and update your configuration.\nError: %s", err.Error()),
		))
		return diags
	}

	project.Id = types.Int64Value(int64(api.GetId()))
	project.MachineName = types.StringValue(api.GetMachineName())
	project.CreatedAt = types.StringValue(api.GetCreatedAt())
	project.UpdatedAt = types.StringValue(api.GetUpdatedAt())
	project.Uuid = types.StringValue(api.GetUuid())

	project.SecurityScore = types.StringValue(api.GetSecurityScore())
	project.GitUrl = types.StringValue(api.GetGitUrl())
	project.OrganizationId = types.Int64Value(int64(api.GetOrganizationId()))

	// API doesn't currently return these values
	if project.AllowQueryParams.IsNull() || project.AllowQueryParams.IsUnknown() {
		project.AllowQueryParams = types.BoolNull()
	}

	if project.BasicAuthPassword.IsNull() || project.BasicAuthPassword.IsUnknown() {
		project.BasicAuthPassword = types.StringNull()
	}

	if project.BasicAuthUsername.IsNull() || project.BasicAuthUsername.IsUnknown() {
		project.BasicAuthUsername = types.StringNull()
	}

	if project.BasicAuthPreviewOnly.IsNull() || project.BasicAuthPreviewOnly.IsUnknown() {
		project.BasicAuthPreviewOnly = types.StringNull()
	}

	if project.CustomS3SyncAccessKey.IsNull() || project.CustomS3SyncAccessKey.IsUnknown() {
		project.CustomS3SyncAccessKey = types.StringNull()
	}

	if project.CustomS3SyncRegion.IsNull() || project.CustomS3SyncRegion.IsUnknown() {
		project.CustomS3SyncRegion = types.StringNull()
	}

	if project.CustomS3SyncSecretKey.IsNull() || project.CustomS3SyncSecretKey.IsUnknown() {
		project.CustomS3SyncSecretKey = types.StringNull()
	}

	if project.CustomS3SyncBucket.IsNull() || project.CustomS3SyncBucket.IsUnknown() {
		project.CustomS3SyncBucket = types.StringNull()
	}

	if project.Project.IsNull() || project.Project.IsUnknown() {
		project.Project = types.StringNull()
	}

	if project.ParentProjectId.IsNull() || project.ParentProjectId.IsUnknown() {
		project.ParentProjectId = types.Int64Null()
	}

	if project.Organization.IsNull() || project.Organization.IsUnknown() {
		project.Organization = types.StringNull()
	}

	if project.DeletedAt.IsNull() || project.DeletedAt.IsUnknown() {
		project.DeletedAt = types.StringNull()
	}

	return diags
}

func callProjectDeleteAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		diags.AddAttributeError(
			path.Root("machine_name"),
			"Missing project.machine_name attribute",
			"To read project information the project machine name needs to be known, plese import the terraform state.",
		)
		return
	}

	org := r.client.Organization
	_, _, err := r.client.Instance.ProjectsAPI.ProjectsDelete(r.client.AuthContext, org, project.MachineName.ValueString()).Execute()

	if err != nil {
		diags.AddError(
			"Unable to delete project",
			fmt.Sprintf("Error: %s", err.Error()),
		)
		return
	}

	return
}
