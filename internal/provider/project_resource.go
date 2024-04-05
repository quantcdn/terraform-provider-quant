package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_project"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	resp.Diagnostics.Append(callProjectApi(ctx, r, &data)...)

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
}

func callProjectApi(ctx context.Context, r *projectResource, project *resource_project.ProjectModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if project.MachineName.IsNull() || project.MachineName.IsUnknown() {
		d := diag.NewErrorDiagnostic(
			"Project machine name is misssing or unknown",
			"The provider tried to access project information from the API with missing or unknown values, please check your local state.",
		)
		diags = append(diags, d)
		return diags
	}

	resp, _, err := r.client.Instance.ProjectsAPI.GetProject(r.client.AuthContext, r.client.Organization, project.MachineName.String()).Execute()
	if err != nil {
		diags = append(diags, diag.NewErrorDiagnostic(
			"Unable to fetch project from source",
			err.Error(),
		))
		return diags
	}

	source := resp.GetData().Project

	// if project.AllowQueryParams.IsNull() {
	// 	project.AllowQueryParams = source.
	// }

	if project.CreatedAt.IsNull() {
		project.CreatedAt = types.StringValue(*source.CreatedAt)
	}

	if project.Id.IsNull() {
		project.Id = types.Int64Value(int64(source.GetId()))
	}

	if project.Uuid.IsNull() {
		// project.Uuid = types.StringValue(source.GetUuid())
	}

}
