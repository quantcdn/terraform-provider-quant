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

	// Update API call logic
	resp.Diagnostics.Append(callProjectUpdateAPI(ctx, r, &data)...)

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

	// @todo: add support for s3 sync.

	apiProject, _, err := r.client.Instance.ProjectsAPI.ProjectsCreate(r.client.AuthContext, r.client.Organization).ProjectRequest(req).Execute()

	if err != nil {
		diags.AddError(
			"Unable to add the project",
			fmt.Sprintf("Error: project exists with this name"),
		)
		return
	}

	project.Id = types.Int64Value(int64(apiProject.GetId()))
	project.MachineName = types.StringValue(apiProject.GetMachineName())
	project.Project = types.StringValue(apiProject.GetMachineName())
	project.CreatedAt = types.StringValue(apiProject.GetCreatedAt())
	project.UpdatedAt = types.StringValue(apiProject.GetUpdatedAt())
	project.Uuid = types.StringValue(apiProject.GetUuid())

	// project.ProjectType = types.StringValue(apiProject.GetProjectType())
	project.GitUrl = types.StringValue(apiProject.GetGitUrl())
	project.OrganizationId = types.Int64Value(int64(apiProject.GetOrganizationId()))
	project.ParentProjectId = types.Int64Value(int64(apiProject.GetParentProjectId()))
	project.Region = types.StringValue(apiProject.GetRegion())

	if apiProject.GetSecurityScore() == "" {
		project.SecurityScore = types.StringNull()
	} else {
		project.SecurityScore = types.StringValue(apiProject.GetSecurityScore())
	}

	project.Organization = types.StringValue(r.client.Organization)

	if apiProject.GetCreatedAt() == "" {
		project.DeletedAt = types.StringNull()
	} else {
		project.DeletedAt = types.StringValue(apiProject.GetDeletedAt())
	}

	if project.AllowQueryParams.IsNull() || project.AllowQueryParams.IsUnknown() {
		project.AllowQueryParams = types.BoolNull()
	}

	return diags
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
	if !project.Organization.IsNull() {
		org = *project.Organization.ValueStringPointer()
	}

	req := *openapiclient.NewProjectRequestWithDefaults()

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

	api, _, err := r.client.Instance.ProjectsAPI.ProjectsUpdate(r.client.AuthContext, org, project.MachineName.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to update project", fmt.Sprintf("Error: %s", err.Error()))
	}

	project.UpdatedAt = types.StringValue(api.GetUpdatedAt())

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
	if !project.Organization.IsNull() {
		org = *project.Organization.ValueStringPointer()
	}

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

	// @todo: Support basic auth details from API.

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
	if !project.Organization.IsNull() {
		org = project.Organization.ValueString()
	}

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
