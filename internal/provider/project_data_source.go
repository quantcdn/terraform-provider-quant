package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*projectDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)
)

func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

type projectDataSource struct {
	client *client.Client
}

type projectDataSourceModel struct {
	// Input attributes
	MachineName types.String `tfsdk:"machine_name"`
	WithToken   types.Bool   `tfsdk:"with_token"`

	// Computed attributes
	Id             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Uuid           types.String `tfsdk:"uuid"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	Region         types.String `tfsdk:"region"`
	OrganizationId types.Int64  `tfsdk:"organization_id"`
	SecurityScore  types.String `tfsdk:"security_score"`
	GitUrl         types.String `tfsdk:"git_url"`
	WriteToken     types.String `tfsdk:"write_token"`
}

func (d *projectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			fmt.Sprintf("Expected *internal.Client, got: %T. Please report this issue to the provider developers", req.ProviderData),
		)
	}
	d.client = client
}

func (d *projectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch details for a specific Quant project, including write token if requested.",
		Attributes: map[string]schema.Attribute{
			"machine_name": schema.StringAttribute{
				Required:    true,
				Description: "The machine name of the project to fetch.",
			},
			"with_token": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to include the write token in the response. Defaults to true for data sources.",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "The numeric ID of the project.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name of the project.",
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the project.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The creation timestamp of the project.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The last update timestamp of the project.",
			},
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The region where the project is deployed.",
			},
			"organization_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The organization ID that owns this project.",
			},
			"security_score": schema.StringAttribute{
				Computed:    true,
				Description: "The security score of the project.",
			},
			"git_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Git URL associated with the project.",
			},
			"write_token": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The write token for the project. Only populated if with_token is true.",
			},
		},
	}
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data projectDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default with_token to true for data sources (since the main use case is to get the token)
	withToken := true
	if !data.WithToken.IsNull() {
		withToken = data.WithToken.ValueBool()
	}

	// Make API call to fetch project details
	org := d.client.Organization
	api, _, err := d.client.Instance.ProjectsAPI.ProjectsRead(d.client.AuthContext, org, data.MachineName.ValueString()).WithToken(withToken).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read project data from API",
			fmt.Sprintf("There was an issue reading project '%s' from the API: %s", data.MachineName.ValueString(), err.Error()),
		)
		return
	}

	// Map API response to model
	data.Name = types.StringValue(api.GetName())
	data.MachineName = types.StringValue(api.GetMachineName())
	data.WithToken = types.BoolValue(withToken)
	
	// Set V2 API fields that are now supported
	if id, ok := api.GetIdOk(); ok && id != nil {
		data.Id = types.Int64Value(int64(*id))
	} else {
		data.Id = types.Int64Null()
	}
	
	if uuid, ok := api.GetUuidOk(); ok && uuid != nil {
		data.Uuid = types.StringValue(*uuid)
	} else {
		data.Uuid = types.StringNull()
	}
	
	if writeToken, ok := api.GetWriteTokenOk(); ok && writeToken != nil {
		data.WriteToken = types.StringValue(*writeToken)
	} else {
		data.WriteToken = types.StringNull()
	}
	
	// Fields not in V2 API - set to null
	data.CreatedAt = types.StringNull()
	data.UpdatedAt = types.StringNull()
	data.Region = types.StringNull()
	data.OrganizationId = types.Int64Null()
	data.SecurityScore = types.StringNull()
	data.GitUrl = types.StringNull()

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
