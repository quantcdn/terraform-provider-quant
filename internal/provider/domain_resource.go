package provider

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/mapper"
	"terraform-provider-quant/internal/resource_domain"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"terraform-provider-quant/internal/utils"
)

var (
	_ resource.Resource                = (*domainResource)(nil)
	_ resource.ResourceWithConfigure   = (*domainResource)(nil)
	_ resource.ResourceWithImportState = (*domainResource)(nil)
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
	s := resource_domain.DomainResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	resp.Schema = s
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
	}

	r.client = client
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_domain.DomainModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callDomainCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the API results back into the model for Terraform state.
	resp.Diagnostics.Append(callDomainReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_domain.DomainModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callDomainReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is a no-op — the V2 API does not support domain updates.
// Mutable fields should use RequiresReplace plan modifiers so that Terraform
// forces destroy+recreate instead of calling Update.
func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_domain.DomainModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-read current state from the API.
	resp.Diagnostics.Append(callDomainReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_domain.DomainModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callDomainDeleteAPI(ctx, r, &data)...)
}

// Import state for a given domain ID.
func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_domain.DomainModel
	var err error

	data.Project, data.Id, err = utils.GetDomainImportId(req.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse import ID. Error: %s", err.Error()),
		)
	}

	diags := callDomainReadAPI(ctx, r, &data)

	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Error reading domain",
			fmt.Sprintf("Could not read domain. Error: %s", diags.Errors()),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create domain request.
func callDomainCreateAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	if domain.Domain.IsNull() || domain.Domain.IsUnknown() {
		diags.AddAttributeError(
			path.Root("domain"),
			"Missing domain.domain attribute",
			"Cannot create a domain without a domain.",
		)
		return
	}

	// Map TF model fields to SDK request via reflection.
	req := quantadmingo.NewV2DomainRequestWithDefaults()
	diags.Append(mapper.ToSDK(ctx, domain, req)...)
	if diags.HasError() {
		return
	}

	org := r.client.Organization
	project := domain.Project.ValueString()
	apiResp, _, err := r.client.Instance.DomainsAPI.DomainsCreate(r.client.AuthContext, org, project).V2DomainRequest(*req).Execute()
	if err != nil {
		diags.AddError(
			"Error creating domain",
			"Could not create domain, unexpected error: "+err.Error(),
		)
		return
	}

	domain.Id = types.Int64Value(int64(apiResp.GetId()))

	createStateConf := retry.StateChangeConf{
		Pending: []string{"pending", "creating"},
		Target:  []string{"ready"},
		Refresh: func() (interface{}, string, error) {
			apiResp, resp, err := r.client.Instance.DomainsAPI.DomainsRead(r.client.AuthContext, org, project, strconv.FormatInt(int64(apiResp.GetId()), 10)).Execute()
			if err != nil {
				if resp != nil && resp.StatusCode == 404 {
					return nil, "pending", nil
				}
				return nil, "", fmt.Errorf("error checking domain status: %v", err)
			}
			return apiResp, "ready", nil
		},
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		diags.AddError("Unable to create domain", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func callDomainReadAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	if domain.Id.IsNull() || domain.Id.IsUnknown() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing domain.id attribute",
			"To read domain information the domain ID needs to be known, please import the terraform state.",
		)
		return
	}

	org := r.client.Organization
	project := domain.Project.ValueString()
	apiResp, _, err := r.client.Instance.DomainsAPI.DomainsRead(r.client.AuthContext, org, project, strconv.FormatInt(domain.Id.ValueInt64(), 10)).Execute()
	if err != nil {
		diags.AddError(
			"Error reading domain",
			"Could not read domain, unexpected error: "+err.Error(),
		)
		return
	}

	// Map SDK response fields to TF model via reflection.
	// This covers: Id, Domain, DnsEngaged (scalar fields with matching getters).
	diags.Append(mapper.FromSDK(ctx, apiResp, domain)...)

	domain.Organization = types.StringValue(org)

	// Nested object lists (dns_go_live_records, dns_validation_records) are not
	// handled by the mapper — set to null since the V2 API does not reliably
	// return these in the standard read response.
	dnsGoLiveRecordsElemType := resource_domain.DnsGoLiveRecordsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_domain.DnsGoLiveRecordsValue{}.AttributeTypes(ctx),
		},
	}
	domain.DnsGoLiveRecords = types.ListNull(dnsGoLiveRecordsElemType)

	dnsValidationRecordsElemType := resource_domain.DnsValidationRecordsType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_domain.DnsValidationRecordsValue{}.AttributeTypes(ctx),
		},
	}
	domain.DnsValidationRecords = types.ListNull(dnsValidationRecordsElemType)

	return diags
}

func callDomainDeleteAPI(ctx context.Context, r *domainResource, domain *resource_domain.DomainModel) (diags diag.Diagnostics) {
	if domain.Domain.IsNull() || domain.Domain.IsUnknown() {
		diags.AddError(
			"Unable to delete domain",
			"Domain name is required to delete domain information",
		)
		return
	}

	org := r.client.Organization
	project := domain.Project.ValueString()
	_, err := r.client.Instance.DomainsAPI.DomainsDelete(r.client.AuthContext, org, project, strconv.FormatInt(domain.Id.ValueInt64(), 10)).Execute()
	if err != nil {
		diags.AddError(
			"Error deleting domain",
			"Could not delete domain, unexpected error: "+err.Error(),
		)
		return
	}

	return diags
}
