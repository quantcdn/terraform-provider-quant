package provid

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmin "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = &domainResource{}
	_ resource.ResourceWithConfigure = &domainResource{}
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *client.Client
}

type domainResourceModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	MachineName types.String `tfsdk:"machine_name"`
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "A FQDN to attach to the project",
				Required:            true,
			},
			"machine_name": schema.StringAttribute{
				MarkdownDescription: "The project machine name",
				Required:            true,
			},
		},
	}
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Admin.DefaultApi

	d := quantadmin.NewDomainCreate(plan.Domain.ValueString())

	_, _, err := client.CreateDomain(r.client.Context).QuantOrganisation(r.client.Organization).QuantProject(plan.MachineName.ValueString()).DomainCreate(*d).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating domain",
			"Could not create domain, unexepected error: "+err.Error(),
		)
		return
	}

	plan.Id = *d.Data.Domain.Id

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Admin.DefaultApi
	res, _, err := client.GetProject(r.client.Context).QuantOrganisation(r.client.Organization).QuantProject(state.MachineName.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project data",
			"Could not read Quant project "+state.MachineName.ValueString()+": "+err.Error(),
		)
		return
	}

	for _, domain := range res.Data.Project.Domains {
		if types.StringValue(*domain.Domain) == state.Domain {
			state.Domain = types.StringValue(*domain.Domain)
			state.MachineName = types.StringValue(*res.Data.Project.MachineName)
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
