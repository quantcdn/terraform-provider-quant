package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_auth"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = (*ruleAuthResource)(nil)
	_ resource.ResourceWithConfigure = (*ruleAuthResource)(nil)
)

func NewRuleAuthResource() resource.Resource {
	return &ruleAuthResource{}
}

type ruleAuthResource struct {
	client *client.Client
}

func (r *ruleAuthResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_auth"
}

func (r *ruleAuthResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_rule_auth.RuleAuthResourceSchema(ctx)
}

func (r *ruleAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_rule_auth.RuleAuthModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	resp.Diagnostics.Append(callRuleAuthCreate(ctx, r, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_auth.RuleAuthModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callRuleAuthRead(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_rule_auth.RuleAuthModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	resp.Diagnostics.Append(callRuleAuthUpdate(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_rule_auth.RuleAuthModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callRuleAuthDelete(ctx, r, &data)...)
}

func callRuleAuthCreate(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
	if rule.Project.IsNull() || rule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing rule.project attribute",
			"To create a rule you must define the project.",
		)
	}

	if rule.AuthUser.IsNull() || rule.AuthUser.IsUnknown() {
		diags.AddAttributeError(
			path.Root("auth_user"),
			"Missing rule.auth_user attribute",
			"To create an authentication rule you need to provide auth_user.",
		)
	}

	if rule.AuthPass.IsNull() || rule.AuthPass.IsUnknown() {
		diags.AddAttributeError(
			path.Root("auth_pass"),
			"Missing rule.auth_pass attribute",
			"To create an authentication rule you need to provide auth_pass.",
		)
	}

	if diags.HasError() {
		// If the initial valiadtion for the rule fails, return all errors.
		return
	}

	req := *openapi.NewRuleAuthRequestWithDefaults()

	setRuleAuthCountryFilters(ctx, rule, &req)
	setRuleAuthMethodFilters(ctx, rule, &req)
	setRuleAuthIpFilters(ctx, rule, &req)

	var urls []string
	rule.Urls.ElementsAs(ctx, urls, false)
	req.SetUrls(urls)

	if rule.Domain.IsNull() {
		req.SetDomain(*utils.GetRuleAny())
	} else {
		req.SetDomain(rule.Domain.ValueString())
	}

	// req.OnlyWithCookie = rule.OnlyWithCookie.ValueStringPointer()

	req.SetDisabled(rule.Disabled.ValueBool())

	req.SetAuthPass(rule.AuthPass.ValueString())
	req.SetAuthUser(rule.AuthUser.ValueString())

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	api, _, err := r.client.Instance.RulesAuthAPI.RulesAuthCreate(r.client.AuthContext, org, rule.Project.ValueString()).RuleAuthRequest(req).Execute()

	if err != nil {
		diags.AddError("Unable to create authentication rule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	rule.Uuid = types.StringValue(api.GetUuid())
	return
}

func callRuleAuthRead(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
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

	api, _, err := r.client.Instance.RulesAuthAPI.RulesAuthRead(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to read rule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	rule.Uuid = types.StringValue(api.GetUuid())

	// @todo parse the config from the API.

	return
}

func callRuleAuthUpdate(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
	if rule.Project.IsNull() || rule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing rule.project attribute",
			"To create a rule you must define the project.",
		)
	}

	if rule.AuthUser.IsNull() || rule.AuthUser.IsUnknown() {
		diags.AddAttributeError(
			path.Root("auth_user"),
			"Missing rule.auth_user attribute",
			"To create an authentication rule you need to provide auth_user.",
		)
	}

	if rule.AuthPass.IsNull() || rule.AuthPass.IsUnknown() {
		diags.AddAttributeError(
			path.Root("auth_pass"),
			"Missing rule.auth_pass attribute",
			"To create an authentication rule you need to provide auth_pass.",
		)
	}

	if diags.HasError() {
		// If the initial valiadtion for the rule fails, return all errors.
		return
	}

	req := *openapi.NewRuleAuthRequestWithDefaults()

	setRuleAuthCountryFilters(ctx, rule, &req)
	setRuleAuthMethodFilters(ctx, rule, &req)
	setRuleAuthIpFilters(ctx, rule, &req)

	var urls []string
	rule.Urls.ElementsAs(ctx, urls, false)
	req.SetUrls(urls)

	if rule.Domain.IsNull() {
		req.SetDomain(*utils.GetRuleAny())
	} else {
		req.SetDomain(rule.Domain.ValueString())
	}

	// req.OnlyWithCookie = rule.OnlyWithCookie.ValueStringPointer()

	req.SetDisabled(rule.Disabled.ValueBool())

	req.SetAuthPass(rule.AuthPass.ValueString())
	req.SetAuthUser(rule.AuthUser.ValueString())

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	api, _, err := r.client.Instance.RulesAuthAPI.RulesAuthUpdate(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).RuleAuthRequest(req).Execute()

	if err != nil {
		diags.AddError("Unable to read rule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	rule.Uuid = types.StringValue(api.GetUuid())

	return
}

func callRuleAuthDelete(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
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

	_, _, err := r.client.Instance.RulesAuthAPI.RulesAuthDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to delete rule", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func setRuleAuthCountryFilters(ctx context.Context, rule *resource_rule_auth.RuleAuthModel, req *openapi.RuleAuthRequest) (diags diag.Diagnostics) {
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

func setRuleAuthMethodFilters(ctx context.Context, rule *resource_rule_auth.RuleAuthModel, req *openapi.RuleAuthRequest) (diags diag.Diagnostics) {
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

func setRuleAuthIpFilters(ctx context.Context, rule *resource_rule_auth.RuleAuthModel, req *openapi.RuleAuthRequest) (diags diag.Diagnostics) {
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
