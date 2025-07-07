package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_redirect"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource                     = (*ruleRedirectResource)(nil)
	_ resource.ResourceWithConfigure        = (*ruleRedirectResource)(nil)
	_ resource.ResourceWithImportState      = (*ruleRedirectResource)(nil)
	_ resource.ResourceWithConfigValidators = (*ruleRedirectResource)(nil)
)

func NewRuleRedirectResource() resource.Resource {
	return &ruleRedirectResource{}
}

type ruleRedirectResource struct {
	client *client.Client
}

func (r *ruleRedirectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_redirect"
}

func (r *ruleRedirectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_rule_redirect.RuleRedirectResourceSchema(ctx)
}

func (r *ruleRedirectResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return RuleBaseConfigValidator()
}

func (r *ruleRedirectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			"Expected *internal.Client, got: %T. Please report this issue to the provider developers",
		)
	}
	r.client = client
}

func (r *ruleRedirectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_rule_redirect.RuleRedirectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	diags := callRuleRedirectCreateAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = callRuleRedirectReadAPI(ctx, r, &data)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(diags...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleRedirectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_redirect.RuleRedirectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	diags := callRuleRedirectReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleRedirectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resource_rule_redirect.RuleRedirectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state resource_rule_redirect.RuleRedirectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.RuleId = state.RuleId

	// Update API call logic
	diags := callRuleRedirectUpdateAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = callRuleRedirectReadAPI(ctx, r, &plan)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(diags...)

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleRedirectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_rule_redirect.RuleRedirectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callRuleRedirectDeleteAPI(ctx, r, &data)...)
}

func (r *ruleRedirectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_redirect.RuleRedirectModel
	var err error
	data.Project, data.RuleId, err = utils.GetRuleImportId(req.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			err.Error(),
		)
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callRuleRedirectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// callRuleRedirectCreateAPI calls the API endpoint to create a rule
// resource in Quant.
func callRuleRedirectCreateAPI(ctx context.Context, r *ruleRedirectResource, rule *resource_rule_redirect.RuleRedirectModel) (diags diag.Diagnostics) {
	req := *quantadmingo.NewRuleRedirectRequestWithDefaults()
	req.SetName(rule.Name.ValueString())

	var domains []string
	for _, domain := range rule.Domain.Elements() {
		if strVal, ok := domain.(types.String); ok {
			domains = append(domains, strVal.ValueString())
		}
	}
	req.SetDomain(domains)

	var urls []string
	for _, url := range rule.Url.Elements() {
		if strVal, ok := url.(types.String); ok {
			urls = append(urls, strVal.ValueString())
		}
	}
	req.SetUrl(urls)

	if !rule.Country.IsNull() {
		req.SetCountry(rule.Country.ValueString())
		var countryList []string
		if rule.Country.ValueString() == "country_is" {
			for _, country := range rule.CountryIs.Elements() {
				if strVal, ok := country.(types.String); ok {
					countryList = append(countryList, strVal.ValueString())
				}
			}
			req.SetCountryIs(countryList)
		} else if rule.Country.ValueString() == "country_is_not" {
			for _, country := range rule.CountryIsNot.Elements() {
				if strVal, ok := country.(types.String); ok {
					countryList = append(countryList, strVal.ValueString())
				}
			}
			req.SetCountryIsNot(countryList)
		}
	}

	if !rule.Ip.IsNull() {
		req.SetIp(rule.Ip.ValueString())
		var iplist []string
		if rule.Ip.ValueString() == "ip_is" {
			for _, ip := range rule.IpIs.Elements() {
				if strVal, ok := ip.(types.String); ok {
					iplist = append(iplist, strVal.ValueString())
				}
			}
			req.SetIpIs(iplist)
		} else if rule.Ip.ValueString() == "ip_is_not" {
			for _, ip := range rule.IpIsNot.Elements() {
				if strVal, ok := ip.(types.String); ok {
					iplist = append(iplist, strVal.ValueString())
				}
			}
			req.SetIpIsNot(iplist)
		}
	}

	if !rule.Method.IsNull() {
		req.SetMethod(rule.Method.ValueString())
		var methodList []string
		if rule.Method.ValueString() == "method_is" {
			for _, method := range rule.MethodIs.Elements() {
				if strVal, ok := method.(types.String); ok {
					methodList = append(methodList, strVal.ValueString())
				}
			}
			req.SetMethodIs(methodList)
		} else if rule.Method.ValueString() == "method_is_not" {
			for _, method := range rule.MethodIsNot.Elements() {
				if strVal, ok := method.(types.String); ok {
					methodList = append(methodList, strVal.ValueString())
				}
			}
			req.SetMethodIsNot(methodList)
		}
	}

	req.SetRedirectCode(rule.RedirectCode.ValueString())
	req.SetRedirectTo(rule.RedirectTo.ValueString())

	res, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectCreate(r.client.AuthContext, r.client.Organization, rule.Project.ValueString()).RuleRedirectRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	rule.Uuid = types.StringValue(res.GetUuid())
	rule.RuleId = types.StringValue(res.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)
	if res.Weight != nil {
		rule.Weight = types.Int64Value(int64(*res.Weight))
	} else {
		rule.Weight = types.Int64Value(0)
	}
	rule.Action = types.StringValue("redirect")
	rule.Rule = types.StringValue("")

	domainList, diag := types.ListValueFrom(ctx, types.StringType, domains)
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	rule.Domain = domainList

	if rule.Method.IsNull() || rule.Method.IsUnknown() {
		rule.Method = types.StringNull()
		emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		rule.MethodIs = emptyList
		rule.MethodIsNot = emptyList
	}

	if rule.Country.IsNull() || rule.Country.IsUnknown() {
		rule.Country = types.StringNull()
		emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		rule.CountryIs = emptyList
		rule.CountryIsNot = emptyList
	}

	if rule.Ip.IsNull() || rule.Ip.IsUnknown() {
		rule.Ip = types.StringNull()
		emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		rule.IpIs = emptyList
		rule.IpIsNot = emptyList
	}

	return
}

// callRuleRedirectReadAPI
func callRuleRedirectReadAPI(ctx context.Context, r *ruleRedirectResource, rule *resource_rule_redirect.RuleRedirectModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	api, res, err := r.client.Instance.RulesRedirectAPI.RulesRedirectRead(r.client.AuthContext, r.client.Organization, rule.Project.ValueString(), rule.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to read rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	// Set required fields
	rule.Name = types.StringValue(*api.Name)
	rule.Uuid = types.StringValue(api.Uuid)
	rule.RuleId = types.StringValue(api.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)
	if api.Weight != nil {
		rule.Weight = types.Int64Value(int64(*api.Weight))
	} else {
		rule.Weight = types.Int64Value(0)
	}
	rule.Action = types.StringValue("redirect")
	rule.OnlyWithCookie = types.StringValue(api.GetOnlyWithCookie())

	// Values for fields that are not present in the API response
	rule.Rule = types.StringValue("")

	// Initialize empty lists for all optional fields
	emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})

	domainList, diag := types.ListValueFrom(ctx, types.StringType, api.Domain)
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	rule.Domain = domainList

	// Handle Method fields
	if api.Method != nil && *api.Method != "" {
		rule.Method = types.StringValue(*api.Method)
		if len(api.MethodIs) > 0 {
			rule.MethodIs, _ = types.ListValueFrom(ctx, types.StringType, api.MethodIs)
		} else {
			rule.MethodIs = emptyList
		}
		if len(api.MethodIsNot) > 0 {
			rule.MethodIsNot, _ = types.ListValueFrom(ctx, types.StringType, api.MethodIsNot)
		} else {
			rule.MethodIsNot = emptyList
		}
	} else {
		rule.Method = types.StringNull()
		rule.MethodIs = emptyList
		rule.MethodIsNot = emptyList
	}

	// Handle Country fields
	if api.Country != nil && *api.Country != "" {
		rule.Country = types.StringValue(*api.Country)
		if len(api.CountryIs) > 0 {
			rule.CountryIs, _ = types.ListValueFrom(ctx, types.StringType, api.CountryIs)
		} else {
			rule.CountryIs = emptyList
		}
		if len(api.CountryIsNot) > 0 {
			rule.CountryIsNot, _ = types.ListValueFrom(ctx, types.StringType, api.CountryIsNot)
		} else {
			rule.CountryIsNot = emptyList
		}
	} else {
		rule.Country = types.StringNull()
		rule.CountryIs = emptyList
		rule.CountryIsNot = emptyList
	}

	// Handle IP fields
	if api.Ip != nil && *api.Ip != "" {
		rule.Ip = types.StringValue(*api.Ip)
		if len(api.IpIs) > 0 {
			rule.IpIs, _ = types.ListValueFrom(ctx, types.StringType, api.IpIs)
		} else {
			rule.IpIs = emptyList
		}
		if len(api.IpIsNot) > 0 {
			rule.IpIsNot, _ = types.ListValueFrom(ctx, types.StringType, api.IpIsNot)
		} else {
			rule.IpIsNot = emptyList
		}
	} else {
		rule.Ip = types.StringNull()
		rule.IpIs = emptyList
		rule.IpIsNot = emptyList
	}

	// Handle boolean fields
	rule.Disabled = types.BoolValue(api.GetDisabled())

	// Handle required fields
	domains, _ := types.ListValueFrom(ctx, types.StringType, api.Domain)
	rule.Domain = domains

	urls, _ := types.ListValueFrom(ctx, types.StringType, api.Url)
	rule.Url = urls

	// Handle redirect specific fields
	rule.RedirectCode = types.StringValue(api.ActionConfig.StatusCode)
	rule.RedirectTo = types.StringValue(api.ActionConfig.To)

	return
}

// callRuleRedirectUpdateAPI
func callRuleRedirectUpdateAPI(ctx context.Context, r *ruleRedirectResource, rule *resource_rule_redirect.RuleRedirectModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	req := *quantadmingo.NewRuleRedirectRequestUpdateWithDefaults()
	req.SetName(rule.Name.ValueString())

	var domains []string
	for _, domain := range rule.Domain.Elements() {
		if strVal, ok := domain.(types.String); ok {
			domains = append(domains, strVal.ValueString())
		}
	}
	req.SetDomain(domains)

	var urls []string
	for _, url := range rule.Url.Elements() {
		if strVal, ok := url.(types.String); ok {
			urls = append(urls, strVal.ValueString())
		}
	}
	req.SetUrl(urls)

	req.SetCountry(rule.Country.ValueString())
	var countryList []string
	for _, country := range rule.CountryIs.Elements() {
		if strVal, ok := country.(types.String); ok {
			countryList = append(countryList, strVal.ValueString())
		}
	}
	req.SetCountryIs(countryList)
	countryList = []string{}
	for _, country := range rule.CountryIsNot.Elements() {
		if strVal, ok := country.(types.String); ok {
			countryList = append(countryList, strVal.ValueString())
		}
	}
	req.SetCountryIsNot(countryList)

	req.SetIp(rule.Ip.ValueString())
	var iplist []string
	for _, ip := range rule.IpIs.Elements() {
		if strVal, ok := ip.(types.String); ok {
			iplist = append(iplist, strVal.ValueString())
		}
	}
	req.SetIpIs(iplist)
	iplist = []string{}
	for _, ip := range rule.IpIsNot.Elements() {
		if strVal, ok := ip.(types.String); ok {
			iplist = append(iplist, strVal.ValueString())
		}
	}
	req.SetIpIsNot(iplist)

	req.SetMethod(rule.Method.ValueString())
	var methodList []string
	for _, method := range rule.MethodIs.Elements() {
		if strVal, ok := method.(types.String); ok {
			methodList = append(methodList, strVal.ValueString())
		}
	}
	req.SetMethodIs(methodList)
	methodList = []string{}
	for _, method := range rule.MethodIsNot.Elements() {
		if strVal, ok := method.(types.String); ok {
			methodList = append(methodList, strVal.ValueString())
		}
	}
	req.SetMethodIsNot(methodList)

	req.SetRedirectCode(rule.RedirectCode.ValueString())
	req.SetRedirectTo(rule.RedirectTo.ValueString())

	_, res, err := r.client.Instance.RulesRedirectAPI.RulesRedirectUpdate(r.client.AuthContext, r.client.Organization, rule.Project.ValueString(), rule.RuleId.ValueString()).RuleRedirectRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	return
}

// callRuleRedirectDeleteAPI calls the delete API endpoint with for a given resource.
func callRuleRedirectDeleteAPI(ctx context.Context, r *ruleRedirectResource, rule *resource_rule_redirect.RuleRedirectModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
	}

	org := r.client.Organization
	_, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
