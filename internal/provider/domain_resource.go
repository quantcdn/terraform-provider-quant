package provider

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_domain"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = (*domainResource)(nil)
	_ resource.ResourceWithConfigure = (*domainResource)(nil)
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *client.Client
}

func (r *domainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_domain.DomainResourceSchema(ctx)
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_domain.DomainModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	resp.Diagnostics.Append(callDomainCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_domain.DomainModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callDomainCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_domain.DomainModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callDomainUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_domain.DomainModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callDomainDeleteAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func callDomainCreateAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	req := *openapi.NewDomainRequestWithDefaults()

	req.Domain = domain.Domain.ValueString()

	org := r.client.Organization
	if !domain.Organization.IsNull() {
		org = domain.Organization.ValueString()
	}

	api, _, err := r.client.Instance.DomainsAPI.DomainsCreate(r.client.AuthContext, org, domain.Project.ValueString()).DomainRequest(req).Execute()
	if err != nil {
		diags.AddError("Unable to add domain", fmt.Sprintf("Error: %s", err.Error()))
	}

	domain.Id = types.Int64Value(int64(api.GetId()))
	domain.CreatedAt = types.StringValue(api.GetCreatedAt())
	domain.DnsEngaged = types.Int64Value(int64(api.GetDnsEngaged()))

	return
}

func callDomainUpdateAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	if domain.Id.IsNull() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing ID attribute",
			"Unable to update the domain because of missing ID",
		)
		return
	}

	if domain.Project.IsNull() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing project attribute",
			"Unable to update the domain because of a missing project",
		)
		return
	}

	org := r.client.Organization
	if !domain.Organization.IsNull() {
		org = domain.Organization.ValueString()
	}

	id := strconv.Itoa(int(domain.Id.ValueInt64()))
	api, _, err := r.client.Instance.DomainsAPI.DomainsUpdate(r.client.AuthContext, org, domain.Project.ValueString(), id).Execute()

	if err != nil {
		diags.AddError("Unable to update domain", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	domain.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	return
}

func callDomainReadAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	if domain.Id.IsNull() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing ID attribute",
			"Unable to update the domain because of missing ID",
		)
		return
	}

	if domain.Project.IsNull() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing project attribute",
			"Unable to update the domain because of a missing project",
		)
		return
	}

	org := r.client.Organization
	if !domain.Organization.IsNull() {
		org = domain.Organization.ValueString()
	}

	id := strconv.Itoa(int(domain.Id.ValueInt64()))

	api, _, err := r.client.Instance.DomainsAPI.DomainsRead(r.client.AuthContext, org, domain.Project.ValueString(), id).Execute()
	if err != nil {
		diags.AddError("Unable to read domain", fmt.Sprintf("Error: %s", err.Error()))
	}

	domain.Id = types.Int64Value(int64(api.GetId()))
	domain.Domain = types.StringValue(api.GetDomain())
	domain.CreatedAt = types.StringValue(api.GetCreatedAt())
	domain.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	domain.DnsEngaged = types.Int64Value(int64(api.GetDnsEngaged()))

	return
}

func callDomainDeleteAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	if domain.Id.IsNull() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing ID attribute",
			"Unable to update the domain because of missing ID",
		)
		return
	}

	if domain.Project.IsNull() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing project attribute",
			"Unable to update the domain because of a missing project",
		)
		return
	}

	org := r.client.Organization
	if !domain.Organization.IsNull() {
		org = domain.Organization.ValueString()
	}

	id := strconv.Itoa(int(domain.Id.ValueInt64()))

	_, _, err := r.client.Instance.DomainsAPI.DomainsDelete(r.client.AuthContext, org, domain.Project.ValueString(), id).Execute()

	if err != nil {
		diags.AddError("Unable to delete project", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	return
}
