package provider

import (
	"context"
	"fmt"
	"net/http"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_custom_response"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource                     = (*ruleCustomResponseResource)(nil)
	_ resource.ResourceWithConfigure        = (*ruleCustomResponseResource)(nil)
	_ resource.ResourceWithImportState      = (*ruleCustomResponseResource)(nil)
	_ resource.ResourceWithConfigValidators = (*ruleCustomResponseResource)(nil)
)

func NewRuleCustomResponseResource() resource.Resource {
	return &ruleCustomResponseResource{}
}

type ruleCustomResponseResource struct {
	client *client.Client
}

func (r *ruleCustomResponseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_custom_response"
}

func (r *ruleCustomResponseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_rule_custom_response.RuleCustomResponseResourceSchema(ctx)
}

func (r *ruleCustomResponseResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return RuleBaseConfigValidator()
}

func (r *ruleCustomResponseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleCustomResponseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Serialise rule modifications to avoid backend JSON races
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}
	var data resource_rule_custom_response.RuleCustomResponseModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags := callRuleCustomResponseCreateAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// No need to read immediately after create - we have all the data from the create response

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleCustomResponseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_custom_response.RuleCustomResponseModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags := callRuleCustomResponseReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleCustomResponseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Serialise rule modifications to avoid backend JSON races
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}
	var plan resource_rule_custom_response.RuleCustomResponseModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state resource_rule_custom_response.RuleCustomResponseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Preserve UUID and RuleId from state (needed for update API call)
	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	diags := callRuleCustomResponseUpdateAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read after update to populate computed fields correctly
	diags = callRuleCustomResponseReadAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleCustomResponseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Serialise rule modifications to avoid backend JSON races
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}
	var data resource_rule_custom_response.RuleCustomResponseModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleCustomResponseDeleteAPI(ctx, r, &data)...)
}

func (r *ruleCustomResponseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_custom_response.RuleCustomResponseModel
	var err error
	data.Project, data.RuleId, err = utils.GetRuleImportId(req.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(callRuleCustomResponseReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleCustomResponseCreateAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	req := *quantadmingo.NewV2RuleCustomResponseRequestWithDefaults()
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

	req.SetCustomResponseStatusCode(int32(rule.CustomResponseStatusCode.ValueInt64()))
	req.SetCustomResponseBody(rule.CustomResponseBody.ValueString())

	// Weight handling
	if !rule.Weight.IsNull() && !rule.Weight.IsUnknown() {
		weight := int32(rule.Weight.ValueInt64())
		req.SetWeight(weight)
	}

	res, _, err := r.client.Instance.RulesAPI.RulesCustomResponseCreate(r.client.AuthContext, r.client.Organization, rule.Project.ValueString()).V2RuleCustomResponseRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	rule.Uuid = types.StringValue(res.GetUuid())
	rule.RuleId = types.StringValue(res.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)
	rule.Action = types.StringValue("custom_response")
	rule.Rule = types.StringValue("")

	// Set only_with_cookie from API response or null if not provided
	if res.OnlyWithCookie != nil && *res.OnlyWithCookie != "" {
		rule.OnlyWithCookie = types.StringValue(*res.OnlyWithCookie)
	} else {
		rule.OnlyWithCookie = types.StringNull()
	}

	// Set conditional fields to null if not used
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

	// Set action_config to null since we expose its fields as top-level attributes
	// This prevents "unknown value" errors
	rule.ActionConfig = resource_rule_custom_response.NewActionConfigValueNull()

	// Set body and status_code to null if unknown
	if rule.Body.IsUnknown() {
		rule.Body = types.StringNull()
	}
	if rule.StatusCode.IsUnknown() {
		rule.StatusCode = types.Int64Null()
	}

	return
}

func callRuleCustomResponseReadAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	// Use shared retry logic for eventual consistency
	api, res, err := utils.RetryRuleRead(ctx, func() (*quantadmingo.V2RuleCustomResponse, *http.Response, error) {
		return r.client.Instance.RulesAPI.RulesCustomResponseRead(r.client.AuthContext, r.client.Organization, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()
	}, "rule_custom_response")

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
	rule.Action = types.StringValue("custom_response")
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

	// Handle custom response specific fields
	if api.ActionConfig.CustomResponseStatusCode != nil {
		rule.CustomResponseStatusCode = types.Int64Value(int64(*api.ActionConfig.CustomResponseStatusCode))
	} else {
		rule.CustomResponseStatusCode = types.Int64Null()
	}
	rule.CustomResponseBody = types.StringValue(api.ActionConfig.CustomResponseBody)

	// Set action_config to null since we expose its fields as top-level attributes
	// This prevents "unknown value" errors
	rule.ActionConfig = resource_rule_custom_response.NewActionConfigValueNull()

	// Set body and status_code to null (these are schema-level computed fields)
	rule.Body = types.StringNull()
	rule.StatusCode = types.Int64Null()

	return
}

func callRuleCustomResponseUpdateAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	req := *quantadmingo.NewV2RuleCustomResponseRequestWithDefaults()
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

	req.SetCustomResponseStatusCode(int32(rule.CustomResponseStatusCode.ValueInt64()))
	req.SetCustomResponseBody(rule.CustomResponseBody.ValueString())

	// Weight handling
	if !rule.Weight.IsNull() && !rule.Weight.IsUnknown() {
		weight := int32(rule.Weight.ValueInt64())
		req.SetWeight(weight)
	}

	api, res, err := r.client.Instance.RulesAPI.RulesCustomResponseUpdate(r.client.AuthContext, r.client.Organization, rule.Project.ValueString(), rule.Uuid.ValueString()).V2RuleCustomResponseRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	// CRITICAL: UUID changes after every update - must capture the new UUID from the response
	rule.Uuid = types.StringValue(api.GetUuid())
	rule.RuleId = types.StringValue(api.GetRuleId())

	return
}

func callRuleCustomResponseDeleteAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
	}

	org := r.client.Organization
	_, err := r.client.Instance.RulesAPI.RulesCustomResponseDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
