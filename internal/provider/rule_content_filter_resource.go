package provider

import (
	"context"
	"fmt"
	"net/http"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_content_filter"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*ruleContentFilterResource)(nil)
	_ resource.ResourceWithConfigure   = (*ruleContentFilterResource)(nil)
	_ resource.ResourceWithImportState = (*ruleContentFilterResource)(nil)
)

func NewRuleContentFilterResource() resource.Resource {
	return &ruleContentFilterResource{}
}

type ruleContentFilterResource struct {
	client *client.Client
}

func (r *ruleContentFilterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_content_filter"
}

func (r *ruleContentFilterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_rule_content_filter.RuleContentFilterResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	resp.Schema = s
}

func (r *ruleContentFilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleContentFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_content_filter.RuleContentFilterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleContentFilterCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleContentFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_content_filter.RuleContentFilterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleContentFilterReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleContentFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var plan resource_rule_content_filter.RuleContentFilterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve UUID and RuleId from state (needed for update API call)
	var state resource_rule_content_filter.RuleContentFilterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	resp.Diagnostics.Append(callRuleContentFilterUpdateAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read after update to populate computed fields correctly
	resp.Diagnostics.Append(callRuleContentFilterReadAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleContentFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_content_filter.RuleContentFilterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleContentFilterDeleteAPI(ctx, r, &data)...)
}

func (r *ruleContentFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_content_filter.RuleContentFilterModel
	var err error
	data.Project, data.Uuid, err = utils.GetRuleImportId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(callRuleContentFilterReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// callRuleContentFilterCreateAPI creates a content filter rule via the API.
func callRuleContentFilterCreateAPI(ctx context.Context, r *ruleContentFilterResource, rule *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	req := quantadmingo.NewV2RuleContentFilterRequestWithDefaults()

	// Map simple fields: name, disabled, weight, domain, url
	diags.Append(mapper.ToSDK(ctx, rule, req)...)
	if diags.HasError() {
		return
	}

	// Content filter specific field (maps to ActionConfig in response, not handled by mapper)
	req.SetFnUuid(rule.FnUuid.ValueString())

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

	res, httpResp, err := r.client.Instance.RulesAPI.RulesContentFilterCreate(
		r.client.AuthContext, r.client.Organization, rule.Project.ValueString(),
	).V2RuleContentFilterRequest(*req).Execute()

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
	rule.Action = types.StringValue("content_filter")
	rule.Rule = types.StringNull()

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
	rule.ActionConfig = resource_rule_content_filter.NewActionConfigValueNull()

	// Read back from API to ensure full state consistency
	readDiags := callRuleContentFilterReadAPI(ctx, r, rule)
	diags.Append(readDiags...)

	return
}

// callRuleContentFilterReadAPI reads a content filter rule from the API.
func callRuleContentFilterReadAPI(ctx context.Context, r *ruleContentFilterResource, rule *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule UUID",
			"Unable to read rule without a UUID. Please update terraform state.",
		)
		return
	}

	api, res, err := utils.RetryRuleRead(ctx, func() (*quantadmingo.V2RuleContentFilter, *http.Response, error) {
		return r.client.Instance.RulesAPI.RulesContentFilterRead(
			r.client.AuthContext, r.client.Organization,
			rule.Project.ValueString(), rule.Uuid.ValueString(),
		).Execute()
	}, "rule_content_filter")

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
	rule.Action = types.StringValue("content_filter")
	rule.Rule = types.StringNull()

	// Content filter specific field from ActionConfig
	rule.FnUuid = types.StringValue(api.ActionConfig.GetFnUuid())

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
	rule.ActionConfig = resource_rule_content_filter.NewActionConfigValueNull()

	return
}

// callRuleContentFilterUpdateAPI updates a content filter rule via the API.
func callRuleContentFilterUpdateAPI(ctx context.Context, r *ruleContentFilterResource, rule *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unknown rule, please update terraform state.",
		)
		return
	}

	req := quantadmingo.NewV2RuleContentFilterRequestWithDefaults()

	// Map simple fields: name, disabled, weight, domain, url
	diags.Append(mapper.ToSDK(ctx, rule, req)...)
	if diags.HasError() {
		return
	}

	// Content filter specific field
	req.SetFnUuid(rule.FnUuid.ValueString())

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

	api, res, err := r.client.Instance.RulesAPI.RulesContentFilterUpdate(
		r.client.AuthContext, r.client.Organization,
		rule.Project.ValueString(), rule.Uuid.ValueString(),
	).V2RuleContentFilterRequest(*req).Execute()

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

// callRuleContentFilterDeleteAPI deletes a content filter rule via the API.
func callRuleContentFilterDeleteAPI(ctx context.Context, r *ruleContentFilterResource, rule *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
		return
	}

	_, err := r.client.Instance.RulesAPI.RulesContentFilterDelete(
		r.client.AuthContext, r.client.Organization,
		rule.Project.ValueString(), rule.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
