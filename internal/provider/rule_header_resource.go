package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_headers"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource = (*ruleHeaderResource)(nil)
	_ resource.ResourceWithConfigure = (*ruleHeaderResource)(nil)
)

func NewRuleHeaderResource() resource.Resource {
	return &ruleHeaderResource{}
}

type ruleHeaderResource struct {
	client *client.Client
}

func (r *ruleHeaderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_header"
}

func (r *ruleHeaderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_rule_headers.RuleHeadersResourceSchema(ctx)
}

func (r *ruleHeaderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleHeaderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_rule_headers.RuleHeadersModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	resp.Diagnostics.Append(callRuleHeaderCreate(ctx, r, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleHeaderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_headers.RuleHeadersModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callRuleHeaderRead...(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleHeaderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_rule_headers.RuleHeadersModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	resp.Diagnostics.Append(callRuleHeaderUpdate(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleHeaderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_rule_headers.RuleHeadersModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callRuleHeaderDelete(ctx, r, &data)...)
}

func callRuleHeaderCreate(ctx context.Context, r *ruleHeaderResource, rule *resource_rule_headers.RuleHeadersModel) (diags diag.Diagnostics) {
	if rule.Project.IsNull() || rule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing rule.project attribute",
			"To create a rule you must define the project.",
		)
		return
	}

	req := *openapi.NewRuleHeaderRequest()

	setRuleHeaderCountryFilters(ctx, rule, &req)
	setRuleHeaderMethodFilters(ctx, rule, &req)
	setRuleHeaderIpFilters(ctx, rule, &req)

	var urls []string
	rule.Url.ElementsAs(ctx, urls, false)
	req.Url = urls

	if rule.Domain.IsNull() {
		req.Domain = utils.GetRuleAny()
	} else {
		req.Domain = rule.Domain.ValueStringPointer()
	}

	req.OnlyWithCookie = rule.OnlyWithCookie.ValueStringPointer()
	req.Disabled = rule.Disabled.ValueBoolPointer()

	var headers map[string]interface{}

	providedHeaders, d := types.ObjectValueFrom(ctx, rule.Headers.AttributeTypes(ctx), rule)
	diags.Append(d...)

	if diags.HasError() {
		return
	}

	diags.Append(providedHeaders.As(ctx, &headers, basetypes.ObjectAsOptions{})...)

	if diags.HasError() {
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}


	api, _, err := r.client.Instance.RulesAPI.CreateRuleHeaders(r.client.AuthContext, org, rule.Project.ValueString()).RuleHeaderRequest(req).Execute()

	if err != nil {
		diags.AddError("Unable to create header rule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	rule.Uuid = types.StringValue(api.GetUuid())

	return
}

func callRuleHeaderRead(ctx context.Context, r *ruleHeaderResource, rule *resource_rule_headers.RuleHeadersModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"To read rule information, uuid must be provided.",
		)
	}

	if rule.Project.IsNull() || rule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing rule.project attribute",
			"To read rule information, project must be provided.",
		)
	}

	if diags.HasError() {
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	api, _, err := r.client.Instance.RulesAPI.GetRuleHeaders(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to read rule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	rule.Uuid = types.StringValue(api.GetUuid())

	// @todo parse the config object and assign to TF state.

	return
}

func callRuleHeaderUpdate(ctx context.Context, r *ruleHeaderResource, rule *resource_rule_headers.RuleHeadersModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"To read rule information, uuid must be provided.",
		)
	}

	if rule.Project.IsNull() || rule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing rule.project attribute",
			"To read rule information, project must be provided.",
		)
	}

	if diags.HasError() {
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	req := *openapi.NewRuleHeaderRequest()

	setRuleHeaderCountryFilters(ctx, rule, &req)
	setRuleHeaderMethodFilters(ctx, rule, &req)
	setRuleHeaderIpFilters(ctx, rule, &req)

	if rule.Domain.IsNull() {
		req.Domain = utils.GetRuleAny()
	} else {
		req.Domain = rule.Domain.ValueStringPointer()
	}

	req.OnlyWithCookie = rule.OnlyWithCookie.ValueStringPointer()
	req.Disabled = rule.Disabled.ValueBoolPointer()

	var headers map[string]interface{}

	providedHeaders, d := types.ObjectValueFrom(ctx, rule.Headers.AttributeTypes(ctx), rule)
	diags.Append(d...)

	if diags.HasError() {
		return
	}

	diags.Append(providedHeaders.As(ctx, &headers, basetypes.ObjectAsOptions{})...)

	_, _, err := r.client.Instance.RulesAPI.UpdateRuleHeaders(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).RuleHeaderRequest(req).Execute()

	if err != nil {
		diags.AddError("Unable to update rule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	return
}

func callRuleHeaderDelete(ctx context.Context, r *ruleHeaderResource, rule *resource_rule_headers.RuleHeadersModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"To read rule information, uuid must be provided.",
		)
	}

	if rule.Project.IsNull() || rule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing rule.project attribute",
			"To read rule information, project must be provided.",
		)
	}

	if diags.HasError() {
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	_, _, err := r.client.Instance.RulesAPI.DeleteRuleHeaders(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to delete rule", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

// Work out the country filters.
func setRuleHeaderCountryFilters(ctx context.Context, rule *resource_rule_headers.RuleHeadersModel, req *openapi.RuleHeaderRequest) (diags diag.Diagnostics) {
	if rule.Country.IsNull() {
		req.Country = utils.GetRuleAny()
	} else if rule.Country.ValueString() != "country_is" && rule.Country.ValueString() != "country_is_not" && rule.Country.ValueString() != "any" {
		diags.AddAttributeError(
			path.Root("country"),
			"Invalid string value for 'country'",
			"Country must equal one of the following: country_is, country_is_not, any.",
		)
		return
	} else {
		req.Country = rule.Country.ValueStringPointer()
		var countryList []string

		switch rule.Country.ValueStringPointer() {
			case utils.GetFilterIs("country"):
				rule.CountryIs.ElementsAs(ctx, &countryList, false)
				req.CountryIs = countryList
			case utils.GetFilterIsNot("country"):
				rule.CountryIsNot.ElementsAs(ctx, &countryList, false)
				req.CountryIsNot = countryList
		}
	}

	return
}

func setRuleHeaderMethodFilters(ctx context.Context, rule *resource_rule_headers.RuleHeadersModel, req *openapi.RuleHeaderRequest) (diags diag.Diagnostics) {
	if rule.Method.IsNull() {
		req.Method = utils.GetRuleAny()
	} else if !utils.IsValidRuleFilter(rule.Method.ValueString()) {
		diags.AddAttributeError(
			path.Root("method"),
			"Invalid string value for 'method'",
			"Method must equal one of the following: country_is, country_is_not, any.",
		)
	} else {
		req.Method = rule.Method.ValueStringPointer()
		var list []string
		switch rule.Method.ValueStringPointer() {
			case utils.GetFilterIs("method"):
				rule.MethodIs.ElementsAs(ctx, &list, false)
				req.MethodIs = list
			case utils.GetFilterIsNot("method"):
				rule.MethodIsNot.ElementsAs(ctx, &list, false)
				req.MethodIsNot = list
		}
	}

	return
}

func setRuleHeaderIpFilters(ctx context.Context, rule *resource_rule_headers.RuleHeadersModel, req *openapi.RuleHeaderRequest) (diags diag.Diagnostics) {
	if rule.Ip.IsNull() {
		req.Ip = utils.GetRuleAny()
	} else if !utils.IsValidRuleFilter(rule.Ip.ValueString()) {
		diags.AddAttributeError(
			path.Root("ip"),
			"Invalid string value for 'method'",
			"Ip must equal one of the following: country_is, country_is_not, any.",
		)
	} else {
		req.Ip = rule.Ip.ValueStringPointer()
		var list []string
		switch rule.Ip.ValueStringPointer() {
			case utils.GetFilterIs("ip"):
				rule.IpIs.ElementsAs(ctx, &list, false)
				req.IpIs = list
			case utils.GetFilterIsNot("ip"):
				rule.IpIsNot.ElementsAs(ctx, &list, false)
				req.IpIsNot = list
		}
	}

	return
}
