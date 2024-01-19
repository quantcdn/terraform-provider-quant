package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmin "github.com/quantcdn/quant-admin-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &projectResource{}
	_ resource.ResourceWithConfigure = &projectResource{}
)

// NewprojectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *client.Client
}

type projectResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	MachineName          types.String `tfsdk:"machine_name"`
	Region               types.String `tfsdk:"region"`
	AllowQueryParams     types.Bool   `tfsdk:"allow_query_params"`
	BasicAuthUsername    types.String `tfsdk:"basic_auth_username"`
	BasicAuthPassword    types.String `tfsdk:"basic_auth_password"`
	BasicAuthPreviewOnly types.Bool   `tfsdk:"basic_auth_preview_only"`
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *quant.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "A unique project name",
				Required:            true,
			},
			"machine_name": schema.StringAttribute{
				Computed: true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The AWS region to provison the project in",
				Optional:            true,
				Default:             stringdefault.StaticString("au"),
				Computed:            true,
			},
			"allow_query_params": schema.BoolAttribute{
				MarkdownDescription: "Allow query params for the project",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Computed:            true,
			},
			"basic_auth_username": schema.StringAttribute{
				MarkdownDescription: "HTTP Authentication username",
				Optional:            true,
			},
			"basic_auth_password": schema.StringAttribute{
				MarkdownDescription: "HTTP authentication password",
				Optional:            true,
			},
			"basic_auth_preview_only": schema.BoolAttribute{
				MarkdownDescription: "Protect the autogenerated URL only with HTTP authentication",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Admin.ProjectsAPI

	// Construct the project object for the API.
	p := *quantadmin.NewProjectRequest()

	p.SetName(plan.Name.ValueString())
	p.SetRegion(plan.Region.ValueString())

	p.SetAllowQueryParams(plan.AllowQueryParams.ValueBool())

	var previewOnly string
	if plan.BasicAuthPreviewOnly.ValueBool() {
		previewOnly = "true"
	} else {
		previewOnly = "false"
	}
	p.SetBasicAuthPreviewOnly(previewOnly)

	if plan.BasicAuthPassword.IsNull() && !plan.BasicAuthUsername.IsNull() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("basic_auth_password"),
			"Basic auth password needs to be configured",
			"Both basic auth username and password must be set, missing basic auth password",
		)
	}

	if !plan.BasicAuthPassword.IsNull() && plan.BasicAuthUsername.IsNull() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("basic_auth_username"),
			"Basic auth username needs to be configured",
			"Both basic auth username and password must be set, missing basic auth username",
		)
	}

	if !plan.BasicAuthUsername.IsNull() && !plan.BasicAuthPassword.IsNull() {
		p.SetBasicAuthUsername(plan.BasicAuthUsername.ValueString())
		p.SetBasicAuthPassword(plan.BasicAuthPassword.ValueString())
	}

	organization := r.client.Organization
	res, d, err := client.OrganizationsOrganizationProjectsPost(r.client.Auth, organization).ProjectRequest(p).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project\n"+helpers.ErrorFromAPIBody(d.Body),
		)
		return
	}

	plan.MachineName = types.StringValue(*res.Data.Projects[0].MachineName)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.MachineName.IsNull() || state.MachineName.IsUnknown() {
		resp.Diagnostics.AddError(
			"Error reading project data",
			"Could not read Quant project data, project machine name unknown.",
		)
		return
	}

	organization := r.client.Organization
	project := state.MachineName.ValueString()

	client := r.client.Admin.ProjectsAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectGet(context.Background(), organization, project).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project data",
			"Could not read Quant project "+state.MachineName.ValueString()+": "+err.Error(),
		)
		return
	}

	state.AllowQueryParams = types.BoolValue(false)
	state.MachineName = types.StringValue(*res.Data.Projects[0].MachineName)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := *quantadmin.NewProjectRequest()
	p.SetName(plan.Name.ValueString())
	p.SetRegion(plan.Region.ValueString())

	p.SetAllowQueryParams(plan.AllowQueryParams.ValueBool())

	var previewOnly string
	if plan.BasicAuthPreviewOnly.ValueBool() {
		previewOnly = "true"
	} else {
		previewOnly = "false"
	}
	p.SetBasicAuthPreviewOnly(previewOnly)

	if plan.BasicAuthPassword.IsNull() && !plan.BasicAuthUsername.IsNull() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("basic_auth_password"),
			"Basic auth password needs to be configured",
			"Both basic auth username and password must be set, missing basic auth password",
		)
	}

	if !plan.BasicAuthPassword.IsNull() && plan.BasicAuthUsername.IsNull() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("basic_auth_username"),
			"Basic auth username needs to be configured",
			"Both basic auth username and password must be set, missing basic auth username",
		)
	}

	if !plan.BasicAuthUsername.IsNull() && !plan.BasicAuthPassword.IsNull() {
		p.SetBasicAuthUsername(plan.BasicAuthUsername.ValueString())
		p.SetBasicAuthPassword(plan.BasicAuthPassword.ValueString())
	}

	organization := r.client.Organization
	project := plan.MachineName.ValueString()

	client := r.client.Admin.ProjectsAPI

	_, _, err := client.OrganizationsOrganizationProjectsProjectPatch(r.client.Auth, organization, project).ProjectRequest(p).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Quant Projet",
			"Could not update order, unexpected error: "+err.Error(),
		)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.MachineName.IsNull() {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Invalid state: project machine name is unknown.",
		)
		return
	}

	organization := r.client.Organization
	project := state.MachineName.ValueString()

	client := r.client.Admin.ProjectsAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectDelete(r.client.Auth, organization, project).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}
