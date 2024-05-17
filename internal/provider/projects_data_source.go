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
	_ datasource.DataSource              = (*projectsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*projectsDataSource)(nil)
)

func NewProjectsDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

type projectsDataSource struct {
	client *client.Client
}

type projectsDataSourceModel struct {
	Projects []projectModel `tfsdk:"projects"`
}

type projectModel struct {
	Name        types.String `tfsdk:"name"`
	MachineName types.String `tfsdk:"machine_name"`
}

func (d *projectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unepxected resource configure type",
			fmt.Sprintf("Expected *internal.Client, got: %T. Please report this issue to the provider developers", req.ProviderData),
		)
	}
	d.client = client
}

func (d *projectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"machine_name": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data projectsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	projects, _, err := d.client.Instance.ProjectsAPI.ProjectsList(d.client.AuthContext, d.client.Organization).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Quant projects for %s", d.client.Organization),
			err.Error(),
		)
		return
	}

	for _, p := range projects {
		project := projectModel{
			Name:        types.StringValue(p.GetName()),
			MachineName: types.StringValue(p.GetMachineName()),
		}
		data.Projects = append(data.Projects, project)
	}

	// // Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
