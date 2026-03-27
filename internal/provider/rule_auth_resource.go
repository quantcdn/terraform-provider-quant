package provider

import (
	"context"
	"fmt"
	"net/http"
	"github.com/quantcdn/terraform-provider-quant/internal/client"
	"github.com/quantcdn/terraform-provider-quant/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/internal/resource_rule_auth"
	"github.com/quantcdn/terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*ruleAuthResource)(nil)
	_ resource.ResourceWithConfigure   = (*ruleAuthResource)(nil)
	_ resource.ResourceWithImportState = (*ruleAuthResource)(nil)
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
	s := resource_rule_auth.RuleAuthResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	resp.Schema = s
}

func (r *ruleAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_auth.RuleAuthModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleAuthCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_auth.RuleAuthModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleAuthReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var plan resource_rule_auth.RuleAuthModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state resource_rule_auth.RuleAuthModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	resp.Diagnostics.Append(callRuleAuthUpdateAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleAuthReadAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_auth.RuleAuthModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleAuthDeleteAPI(ctx, r, &data)...)
}

func (r *ruleAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_auth.RuleAuthModel
	var err error
	data.Project, data.Uuid, err = utils.GetRuleImportId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(callRuleAuthReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleAuthCreateAPI(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
	req := quantadmingo.NewV2RuleAuthRequestWithDefaults()

	diags.Append(mapper.ToSDK(ctx, rule, req)...)
	if diags.HasError() {
		return
	}

	// Auth-specific fields
	req.SetAuthUser(rule.AuthUser.ValueString())
	req.SetAuthPass(rule.AuthPass.ValueString())

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

	res, httpResp, err := r.client.Instance.RulesAPI.RulesAuthCreate(
		r.client.AuthContext, r.client.Organization, rule.Project.ValueString(),
	).V2RuleAuthRequest(*req).Execute()

	if err != nil {
		if msg := parseAPIError(httpResp); msg != "" {
			diags.AddError("Failed to create rule", msg)
			return
		}
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	rule.Uuid = types.StringValue(res.GetUuid())
	rule.RuleId = types.StringValue(res.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)
	rule.Action = types.StringValue("auth")
	rule.Rule = types.StringNull()

	if res.Weight != nil {
		rule.Weight = types.Int64Value(int64(*res.Weight))
	} else {
		rule.Weight = types.Int64Value(0)
	}

	if res.OnlyWithCookie != nil && *res.OnlyWithCookie != "" {
		rule.OnlyWithCookie = types.StringValue(*res.OnlyWithCookie)
	} else {
		rule.OnlyWithCookie = types.StringNull()
	}

	domainList, d := types.ListValueFrom(ctx, types.StringType, res.GetDomain())
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	rule.Domain = domainList

	diags.Append(setConditionalListsFromAPI(ctx,
		res.Country, res.CountryIs, res.CountryIsNot,
		res.Ip, res.IpIs, res.IpIsNot,
		res.Method, res.MethodIs, res.MethodIsNot,
		&rule.Country, &rule.CountryIs, &rule.CountryIsNot,
		&rule.Ip, &rule.IpIs, &rule.IpIsNot,
		&rule.Method, &rule.MethodIs, &rule.MethodIsNot,
	)...)

	rule.ActionConfig = resource_rule_auth.NewActionConfigValueNull()

	readDiags := callRuleAuthReadAPI(ctx, r, rule)
	diags.Append(readDiags...)

	return
}

func callRuleAuthReadAPI(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule UUID",
			"Unable to read rule without a UUID. Please update terraform state.",
		)
		return
	}

	api, res, err := utils.RetryRuleRead(ctx, func() (*quantadmingo.V2RuleAuth, *http.Response, error) {
		return r.client.Instance.RulesAPI.RulesAuthRead(
			r.client.AuthContext, r.client.Organization,
			rule.Project.ValueString(), rule.Uuid.ValueString(),
		).Execute()
	}, "rule_auth")

	if err != nil {
		diags.AddError("Failed to read rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	diags.Append(mapper.FromSDK(ctx, api, rule)...)

	rule.Uuid = types.StringValue(api.Uuid)
	rule.RuleId = types.StringValue(api.GetRuleId())
	rule.Organization = types.StringValue(r.client.Organization)
	rule.Action = types.StringValue("auth")
	rule.Rule = types.StringNull()

	// Auth-specific fields from ActionConfig
	rule.AuthUser = types.StringValue(api.ActionConfig.GetAuthUser())
	rule.AuthPass = types.StringValue(api.ActionConfig.GetAuthPass())

	diags.Append(setConditionalListsFromAPI(ctx,
		api.Country, api.CountryIs, api.CountryIsNot,
		api.Ip, api.IpIs, api.IpIsNot,
		api.Method, api.MethodIs, api.MethodIsNot,
		&rule.Country, &rule.CountryIs, &rule.CountryIsNot,
		&rule.Ip, &rule.IpIs, &rule.IpIsNot,
		&rule.Method, &rule.MethodIs, &rule.MethodIsNot,
	)...)

	rule.ActionConfig = resource_rule_auth.NewActionConfigValueNull()

	return
}

func callRuleAuthUpdateAPI(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unknown rule, please update terraform state.",
		)
		return
	}

	req := quantadmingo.NewV2RuleAuthRequestWithDefaults()

	diags.Append(mapper.ToSDK(ctx, rule, req)...)
	if diags.HasError() {
		return
	}

	req.SetAuthUser(rule.AuthUser.ValueString())
	req.SetAuthPass(rule.AuthPass.ValueString())

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

	api, res, err := r.client.Instance.RulesAPI.RulesAuthUpdate(
		r.client.AuthContext, r.client.Organization,
		rule.Project.ValueString(), rule.Uuid.ValueString(),
	).V2RuleAuthRequest(*req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	rule.Uuid = types.StringValue(api.GetUuid())
	rule.RuleId = types.StringValue(api.GetRuleId())

	return
}

func callRuleAuthDeleteAPI(ctx context.Context, r *ruleAuthResource, rule *resource_rule_auth.RuleAuthModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
		return
	}

	_, err := r.client.Instance.RulesAPI.RulesAuthDelete(
		r.client.AuthContext, r.client.Organization,
		rule.Project.ValueString(), rule.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
