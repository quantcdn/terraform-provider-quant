package provider

import (
	"context"
	"fmt"
	"net/http"
	"github.com/quantcdn/terraform-provider-quant/internal/client"
	"github.com/quantcdn/terraform-provider-quant/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/internal/resource_rule_custom_response"
	"github.com/quantcdn/terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
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
	s := resource_rule_custom_response.RuleCustomResponseResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	resp.Schema = s
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
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}
	r.client = client
}

func (r *ruleCustomResponseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_custom_response.RuleCustomResponseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleCustomResponseCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleCustomResponseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_custom_response.RuleCustomResponseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleCustomResponseReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleCustomResponseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var plan resource_rule_custom_response.RuleCustomResponseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve UUID and RuleId from state (needed for update API call)
	var state resource_rule_custom_response.RuleCustomResponseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	resp.Diagnostics.Append(callRuleCustomResponseUpdateAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read after update to populate computed fields correctly
	resp.Diagnostics.Append(callRuleCustomResponseReadAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleCustomResponseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
	data.Project, data.Uuid, err = utils.GetRuleImportId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(callRuleCustomResponseReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// callRuleCustomResponseCreateAPI creates a custom response rule via the API.
func callRuleCustomResponseCreateAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	req := quantadmingo.NewV2RuleCustomResponseRequestWithDefaults()

	// Map simple fields: name, disabled, weight, domain, url
	diags.Append(mapper.ToSDK(ctx, rule, req)...)
	if diags.HasError() {
		return
	}

	// Custom response specific fields (map to ActionConfig in the response)
	req.SetCustomResponseBody(rule.CustomResponseBody.ValueString())
	req.SetCustomResponseStatusCode(int32(rule.CustomResponseStatusCode.ValueInt64()))

	// Conditional list fields: country, ip, method
	diags.Append(buildConditionalListsForRequest(ctx,
		rule.Country, rule.CountryIs, rule.CountryIsNot,
		rule.Ip, rule.IpIs, rule.IpIsNot,
		rule.Method, rule.MethodIs, rule.MethodIsNot,
		req.SetCountry, req.SetCountryIs, req.SetCountryIsNot,
		req.SetIp, req.SetIpIs, req.SetIpIsNot,
		req.SetMethod, req.SetMethodIs, req.SetMethodIsNot,
	)...)
	if diags.HasError() {
		return
	}

	res, httpResp, err := r.client.Instance.RulesAPI.RulesCustomResponseCreate(
		r.client.AuthContext, r.client.Organization, rule.Project.ValueString(),
	).V2RuleCustomResponseRequest(*req).Execute()

	if err != nil {
		if msg := parseAPIError(httpResp); msg != "" {
			diags.AddError("Failed to create rule", msg)
			return
		}
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	// Capture UUID and RuleId from create response
	rule.Uuid = types.StringValue(res.GetUuid())
	rule.RuleId = types.StringValue(res.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)

	// Set constants
	rule.Action = types.StringValue("custom_response")
	rule.Rule = types.StringValue("")

	// Set weight from response (may differ from request)
	if res.Weight != nil {
		rule.Weight = types.Int64Value(int64(*res.Weight))
	} else {
		rule.Weight = types.Int64Value(0)
	}

	// Set only_with_cookie from response
	if res.OnlyWithCookie != nil && *res.OnlyWithCookie != "" {
		rule.OnlyWithCookie = types.StringValue(*res.OnlyWithCookie)
	} else {
		rule.OnlyWithCookie = types.StringNull()
	}

	// Normalize domain list from response
	domainList, d := types.ListValueFrom(ctx, types.StringType, res.GetDomain())
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	rule.Domain = domainList

	// Ensure conditional list fields have proper values (empty lists for null selectors)
	diags.Append(setConditionalListsFromAPI(ctx,
		res.Country, res.CountryIs, res.CountryIsNot,
		res.Ip, res.IpIs, res.IpIsNot,
		res.Method, res.MethodIs, res.MethodIsNot,
		&rule.Country, &rule.CountryIs, &rule.CountryIsNot,
		&rule.Ip, &rule.IpIs, &rule.IpIsNot,
		&rule.Method, &rule.MethodIs, &rule.MethodIsNot,
	)...)

	// Set action_config to null since we expose its fields as top-level attributes
	rule.ActionConfig = resource_rule_custom_response.NewActionConfigValueNull()

	// Set legacy computed fields to null
	rule.Body = types.StringNull()
	rule.StatusCode = types.Int64Null()

	// Read back from API to ensure full state consistency
	readDiags := callRuleCustomResponseReadAPI(ctx, r, rule)
	diags.Append(readDiags...)

	return
}

// callRuleCustomResponseReadAPI reads a custom response rule from the API.
func callRuleCustomResponseReadAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule UUID",
			"Unable to read rule without a UUID. Please update terraform state.",
		)
		return
	}

	api, res, err := utils.RetryRuleRead(ctx, func() (*quantadmingo.V2RuleCustomResponse, *http.Response, error) {
		return r.client.Instance.RulesAPI.RulesCustomResponseRead(
			r.client.AuthContext, r.client.Organization,
			rule.Project.ValueString(), rule.Uuid.ValueString(),
		).Execute()
	}, "rule_custom_response")

	if err != nil {
		diags.AddError("Failed to read rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	// Map simple response fields: name, disabled, weight, domain, url, only_with_cookie
	diags.Append(mapper.FromSDK(ctx, api, rule)...)

	// Override fields that need special handling
	rule.Uuid = types.StringValue(api.Uuid)
	rule.RuleId = types.StringValue(api.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)

	// Constants
	rule.Action = types.StringValue("custom_response")
	rule.Rule = types.StringValue("")

	// Custom response specific fields from ActionConfig
	if api.ActionConfig != nil {
		if api.ActionConfig.CustomResponseStatusCode != nil {
			rule.CustomResponseStatusCode = types.Int64Value(int64(*api.ActionConfig.CustomResponseStatusCode))
		} else {
			rule.CustomResponseStatusCode = types.Int64Null()
		}
		rule.CustomResponseBody = types.StringValue(api.ActionConfig.CustomResponseBody)
	}

	// Conditional list fields
	diags.Append(setConditionalListsFromAPI(ctx,
		api.Country, api.CountryIs, api.CountryIsNot,
		api.Ip, api.IpIs, api.IpIsNot,
		api.Method, api.MethodIs, api.MethodIsNot,
		&rule.Country, &rule.CountryIs, &rule.CountryIsNot,
		&rule.Ip, &rule.IpIs, &rule.IpIsNot,
		&rule.Method, &rule.MethodIs, &rule.MethodIsNot,
	)...)

	// Set action_config to null since we expose its fields as top-level attributes
	rule.ActionConfig = resource_rule_custom_response.NewActionConfigValueNull()

	// Set legacy computed fields to null
	rule.Body = types.StringNull()
	rule.StatusCode = types.Int64Null()

	return
}

// callRuleCustomResponseUpdateAPI updates a custom response rule via the API.
func callRuleCustomResponseUpdateAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unknown rule, please update terraform state.",
		)
		return
	}

	req := quantadmingo.NewV2RuleCustomResponseRequestWithDefaults()

	// Map simple fields: name, disabled, weight, domain, url
	diags.Append(mapper.ToSDK(ctx, rule, req)...)
	if diags.HasError() {
		return
	}

	// Custom response specific fields
	req.SetCustomResponseBody(rule.CustomResponseBody.ValueString())
	req.SetCustomResponseStatusCode(int32(rule.CustomResponseStatusCode.ValueInt64()))

	// Conditional list fields: country, ip, method
	diags.Append(buildConditionalListsForRequest(ctx,
		rule.Country, rule.CountryIs, rule.CountryIsNot,
		rule.Ip, rule.IpIs, rule.IpIsNot,
		rule.Method, rule.MethodIs, rule.MethodIsNot,
		req.SetCountry, req.SetCountryIs, req.SetCountryIsNot,
		req.SetIp, req.SetIpIs, req.SetIpIsNot,
		req.SetMethod, req.SetMethodIs, req.SetMethodIsNot,
	)...)
	if diags.HasError() {
		return
	}

	api, res, err := r.client.Instance.RulesAPI.RulesCustomResponseUpdate(
		r.client.AuthContext, r.client.Organization,
		rule.Project.ValueString(), rule.Uuid.ValueString(),
	).V2RuleCustomResponseRequest(*req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	// Capture UUID and RuleId from response
	rule.Uuid = types.StringValue(api.GetUuid())
	rule.RuleId = types.StringValue(api.GetRuleId())

	return
}

// callRuleCustomResponseDeleteAPI deletes a custom response rule via the API.
func callRuleCustomResponseDeleteAPI(ctx context.Context, r *ruleCustomResponseResource, rule *resource_rule_custom_response.RuleCustomResponseModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
		return
	}

	_, err := r.client.Instance.RulesAPI.RulesCustomResponseDelete(
		r.client.AuthContext, r.client.Organization,
		rule.Project.ValueString(), rule.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
