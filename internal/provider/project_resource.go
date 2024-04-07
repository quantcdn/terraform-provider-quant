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
	_ resource.Resource = (*projectResource)(nil)
	_ resource.ResourceWithConfigure = (*projectResource) (nil)
)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct{
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

// Create project request.
func callProjectCreateAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
	req := *openapiclient.NewProjectRequest()

	req.AllowQueryParams = project.AllowQueryParams.ValueBoolPointer()

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

	req.BasicAuthPassword = project.BasicAuthPassword.ValueStringPointer()
	req.BasicAuthUsername = project.BasicAuthUsername.ValueStringPointer()
	req.BasicAuthPreviewOnly = project.BasicAuthPreviewOnly.ValueStringPointer()

	// Validate the S3 credentials for S3 sync, if one s3 value is provided
	// ensure that all S3 values are provided.
	if project.CustomS3SyncAccessKey.IsNull() || project.CustomS3SyncBucket.IsNull() || project.CustomS3SyncRegion.IsNull() || project.CustomS3SyncSecretKey.IsNull() {
		s3Detail := "To enable S3 synchronisation custom_s3_sync_access_key, custom_s3_sync_secret_key, "+
		"custom_s3_sync_bucket and custom_s3_sync_region must be provided."

		if !project.CustomS3SyncAccessKey.IsNull() {
			diags.AddError("Missing s3 sync access key", s3Detail)
		}

		if !project.CustomS3SyncBucket.IsNull() {
			diags.AddError("Missing s3 sync bucket", s3Detail)
		}
		if !project.CustomS3SyncSecretKey.IsNull() {
			diags.AddError("Missing s3 sync secret key", s3Detail)
		}
		if !project.CustomS3SyncRegion.IsNull() {
			diags.AddError("Missing s3 sync region", s3Detail)
		}
	}

	// @todo: API to support setting the git url.
	// req.GitUrl = project.GitUrl.ValueStringPointer()

	req.Name = project.Name.ValueStringPointer()

	apiProject, _, err :=  r.client.Instance.ProjectsAPI.CreateProject(r.client.AuthContext, r.client.Organization).ProjectRequest(req).Execute()

	if err != nil {
		diags.AddError(
			"Unable to add the project",
			fmt.Sprintf("Error: %s", err.Error()),
		)
	}

	project.Id = types.Int64Value(int64(apiProject.GetId()))
	project.MachineName = types.StringValue(apiProject.GetMachineName())
	project.CreatedAt = types.StringValue(apiProject.GetCreatedAt())
	project.Uuid = types.StringValue(apiProject.GetUuid())

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

	req := *openapiclient.NewProjectRequest()

	req.AllowQueryParams = project.AllowQueryParams.ValueBoolPointer()

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

	req.BasicAuthPassword = project.BasicAuthPassword.ValueStringPointer()
	req.BasicAuthUsername = project.BasicAuthUsername.ValueStringPointer()
	req.BasicAuthPreviewOnly = project.BasicAuthPreviewOnly.ValueStringPointer()

	// Validate the S3 credentials for S3 sync, if one s3 value is provided
	// ensure that all S3 values are provided.
	if project.CustomS3SyncAccessKey.IsNull() || project.CustomS3SyncBucket.IsNull() || project.CustomS3SyncRegion.IsNull() || project.CustomS3SyncSecretKey.IsNull() {
		s3Detail := "To enable S3 synchronisation custom_s3_sync_access_key, custom_s3_sync_secret_key, "+
		"custom_s3_sync_bucket and custom_s3_sync_region must be provided."

		if !project.CustomS3SyncAccessKey.IsNull() {
			diags.AddError("Missing s3 sync access key", s3Detail)
		}

		if !project.CustomS3SyncBucket.IsNull() {
			diags.AddError("Missing s3 sync bucket", s3Detail)
		}
		if !project.CustomS3SyncSecretKey.IsNull() {
			diags.AddError("Missing s3 sync secret key", s3Detail)
		}
		if !project.CustomS3SyncRegion.IsNull() {
			diags.AddError("Missing s3 sync region", s3Detail)
		}
	}

	api, _, err := r.client.Instance.ProjectsAPI.UpdateProject(r.client.AuthContext, org, project.MachineName.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to update project", fmt.Sprintf("Error: %s", err.Error()))
	}

	project.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	return
}

func callProjectReadAPI(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) (diags diag.Diagnostics) {
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

	api, _, err := r.client.Instance.ProjectsAPI.GetProject(r.client.AuthContext, org, project.MachineName.ValueString()).Execute()

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
	project.Uuid = types.StringValue(api.GetUuid())

	// @todo: API does not return these values.)
	// project.BasicAuthPassword = types.StringValue(api.GetBasicAuthPassword())
	// project.BasicAuthUsername = types.StringValue(api.GetBasicAuthUsername())
	// project.BasicAuthPreviewOnly = types.StringValue(api.GetBasicAuthPreviewOnly())

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

	_, _, err := r.client.Instance.ProjectsAPI.DeleteProject(r.client.AuthContext, org, project.MachineName.ValueString()).Execute()

	if err != nil {
		diags.AddError(
			"Unable to delete project",
			fmt.Sprintf("Error: %s", err.Error()),
		)
		return
	}
}
