package provider

import (
	"context"
	"fmt"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_proxy"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/utils"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*ruleProxyResource)(nil)
	_ resource.ResourceWithConfigure   = (*ruleProxyResource)(nil)
	_ resource.ResourceWithImportState = (*ruleProxyResource)(nil)
)

func NewRuleProxyResource() resource.Resource {
	return &ruleProxyResource{}
}

type ruleProxyResource struct {
	client *client.Client
}

// parseCacheLifetime converts a string cache_lifetime value to int64 for API calls.
// Supports backwards compatibility with existing integer configurations.
func parseCacheLifetime(cacheLifetimeStr types.String) (int64, error) {
	if cacheLifetimeStr.IsNull() || cacheLifetimeStr.IsUnknown() {
		return 0, fmt.Errorf("cache_lifetime is null or unknown")
	}
	value := cacheLifetimeStr.ValueString()
	if value == "" {
		return 0, fmt.Errorf("cache_lifetime is empty")
	}
	return strconv.ParseInt(value, 10, 64)
}

// formatCacheLifetime converts a string cache_lifetime value to types.String for Terraform state.
func formatCacheLifetime(cacheLifetime string) types.String {
	return types.StringValue(cacheLifetime)
}

func (r *ruleProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_proxy"
}

func (r *ruleProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_rule_proxy.RuleProxyResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	dropThresholdsModeDefault(s.Attributes)
	resp.Schema = s
}

func (r *ruleProxyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleProxyCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var plan resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve UUID and RuleId from state (needed for update API call)
	var state resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	resp.Diagnostics.Append(callRuleProxyUpdateAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read after update to populate computed fields correctly
	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client != nil && r.client.RulesMutex != nil {
		r.client.RulesMutex.Lock()
		defer r.client.RulesMutex.Unlock()
	}

	var data resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleProxyDeleteAPI(ctx, r, &data)...)
}

func (r *ruleProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_proxy.RuleProxyModel
	var err error
	data.Project, data.Uuid, err = utils.GetRuleImportId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildProxyRequest creates and populates a V2RuleProxyRequest from the TF model.
// Used by both Create and Update to avoid code duplication.
func buildProxyRequest(ctx context.Context, data *resource_rule_proxy.RuleProxyModel) (*quantadmingo.V2RuleProxyRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := quantadmingo.NewV2RuleProxyRequestWithDefaults()

	// mapper.ToSDK handles: Name, Disabled, Weight, Domain, Url, Host,
	// AuthUser, AuthPass, DisableSslVerify, OnlyProxy404, ApplicationProxy,
	// ApplicationName, ApplicationEnvironment, ApplicationContainer,
	// ApplicationPort, ProxyStripHeaders, ProxyStripRequestHeaders,
	// FailoverMode, FailoverOriginTtfb, FailoverOriginStatusCodes,
	// FailoverLifetime, ProxyAlertEnabled, WafEnabled, OriginTimeout,
	// StaticErrorPage, StaticErrorPageStatusCodes
	// waf_config is caller-handled: it is only sent when waf_enabled is true.
	diags.Append(mapper.ToSDK(ctx, data, req, "waf_config")...)
	if diags.HasError() {
		return nil, diags
	}

	setProxyTarget(data, req)
	diags.Append(setProxyCacheLifetime(data, req)...)
	if diags.HasError() {
		return nil, diags
	}

	// Conditional list fields: country, ip, method
	diags.Append(buildConditionalListsForRequest(ctx,
		data.Country, data.CountryIs, data.CountryIsNot,
		data.Ip, data.IpIs, data.IpIsNot,
		data.Method, data.MethodIs, data.MethodIsNot,
		req.SetCountry, req.SetCountryIs, req.SetCountryIsNot,
		req.SetIp, req.SetIpIs, req.SetIpIsNot,
		req.SetMethod, req.SetMethodIs, req.SetMethodIsNot,
	)...)
	diags.Append(buildRuleExtraConditions(ctx, proxyExtraConditions(data), req)...)
	if diags.HasError() {
		return nil, diags
	}

	// Inject headers (map field — not handled by mapper)
	diags.Append(setProxyInjectHeaders(ctx, data, req)...)
	if diags.HasError() {
		return nil, diags
	}

	// WAF configuration (nested object — not handled by mapper)
	if data.WafEnabled.ValueBool() && !data.WafConfig.IsNull() && !data.WafConfig.IsUnknown() {
		wafConfig, d := buildWafConfig(ctx, data.WafConfig)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		req.SetWafConfig(*wafConfig)
	}

	return req, diags
}

// setWafStringList is a helper to extract a types.List and call a WAF setter.
func setWafStringList(ctx context.Context, list types.List, setter func([]string), diags *diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return
	}
	vals, d := extractStringList(ctx, list)
	diags.Append(d...)
	if vals != nil {
		setter(vals)
	}
}

// callRuleProxyCreateAPI creates a proxy rule via the API.
func callRuleProxyCreateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	req, d := buildProxyRequest(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	api, httpResp, err := r.client.Instance.RulesAPI.RulesProxyCreate(
		r.client.AuthContext, r.client.Organization, data.Project.ValueString(),
	).V2RuleProxyRequest(*req).Execute()

	if err != nil {
		if msg := parseAPIError(httpResp); msg != "" {
			diags.AddError("Failed to create rule", msg)
			return
		}
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	// Capture UUID and RuleId from create response
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())
	data.Organization = types.StringValue(r.client.Organization)

	// Set constants
	data.Action = types.StringValue("proxy")
	data.Rule = types.StringValue("")

	// Read back from API to get computed fields and ensure state consistency
	diags.Append(callRuleProxyReadAPI(ctx, r, data)...)
	return
}

// callRuleProxyReadAPI reads a proxy rule from the API.
func callRuleProxyReadAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.Uuid.IsNull() || data.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule UUID",
			"Unable to read rule without a UUID. Please update terraform state.",
		)
		return
	}

	// Preserve request-only application fields from prior state
	prior := *data

	tflog.Debug(ctx, "=== RULE PROXY READ REQUEST ===", map[string]interface{}{
		"organization": r.client.Organization,
		"project":      data.Project.ValueString(),
		"uuid":         data.Uuid.ValueString(),
	})

	api, httpResp, err := utils.RetryRuleRead(ctx, func() (*quantadmingo.V2RuleProxy, *http.Response, error) {
		return r.client.Instance.RulesAPI.RulesProxyRead(
			r.client.AuthContext, r.client.Organization,
			data.Project.ValueString(), data.Uuid.ValueString(),
		).Execute()
	}, "rule_proxy")

	if err != nil {
		diags.Append(ruleProxyReadError(httpResp, err, data.Uuid.ValueString())...)
		return
	}

	actionConfig := api.GetActionConfig()

	// mapper.FromSDK handles simple response fields: Name, Disabled, Weight,
	// Domain, Url, OnlyWithCookie, and all top-level string/bool/int/list fields
	// that have matching getters on V2RuleProxy.
	diags.Append(mapper.FromSDK(ctx, api, data)...)

	// Override fields that need special handling
	data.Uuid = types.StringValue(api.Uuid)
	data.RuleId = types.StringValue(api.GetRuleId())
	data.Organization = types.StringValue(r.client.Organization)
	data.Action = types.StringValue(api.GetAction())
	data.Rule = types.StringNull()

	data.InjectHeaders = injectHeadersFromAPI(actionConfig)

	// OnlyWithCookie: mapper sets it from API, but we need null for empty string
	if api.OnlyWithCookie == nil || *api.OnlyWithCookie == "" {
		data.OnlyWithCookie = types.StringNull()
	}

	setProxyTargetFromAPI(actionConfig, &prior, data)
	data.OriginTimeout = originTimeoutFromAPI(actionConfig)
	restoreProxyApplicationFields(&prior, data)

	// Cache lifetime: use API value converted to string, preserve null if not configured
	if !data.CacheLifetime.IsNull() {
		data.CacheLifetime = formatCacheLifetime(actionConfig.GetCacheLifetime())
	}

	// ActionConfig scalar booleans
	data.DisableSslVerify = types.BoolValue(actionConfig.GetDisableSslVerify())
	data.OnlyProxy404 = types.BoolValue(actionConfig.GetOnlyProxy404())
	data.ProxyAlertEnabled = types.BoolValue(actionConfig.GetProxyAlertEnabled())

	// Conditional list fields: country, ip, method
	diags.Append(setConditionalListsFromAPI(ctx,
		api.Country, api.CountryIs, api.CountryIsNot,
		api.Ip, api.IpIs, api.IpIsNot,
		api.Method, api.MethodIs, api.MethodIsNot,
		&data.Country, &data.CountryIs, &data.CountryIsNot,
		&data.Ip, &data.IpIs, &data.IpIsNot,
		&data.Method, &data.MethodIs, &data.MethodIsNot,
	)...)
	diags.Append(setRuleExtraConditionsFromAPI(ctx, api, proxyExtraConditions(data))...)

	// ProxyStripHeaders / ProxyStripRequestHeaders from ActionConfig
	diags.Append(setStringListFromAPI(ctx, actionConfig.GetProxyStripHeaders(), &data.ProxyStripHeaders)...)
	diags.Append(setStringListFromAPI(ctx, actionConfig.GetProxyStripRequestHeaders(), &data.ProxyStripRequestHeaders)...)

	diags.Append(setProxyFailoverFromAPI(ctx, actionConfig, data)...)

	// WAF configuration (nested object)
	data.WafEnabled = types.BoolValue(actionConfig.GetWafEnabled())
	if data.WafEnabled.ValueBool() {
		diags.Append(setWafConfigFromAPI(ctx, actionConfig.GetWafConfig(), data)...)
	} else {
		setWafDisabledDefaults(ctx, data)
	}

	diags.Append(setProxyStaticErrorPageFromAPI(ctx, actionConfig, data)...)

	// Set action_config to null since we expose its fields as top-level attributes
	data.ActionConfig = resource_rule_proxy.NewActionConfigValueNull()

	// If waf_config wasn't properly populated, set to null
	if data.WafConfig.IsUnknown() || data.WafConfig.Mode.IsUnknown() {
		data.WafConfig = resource_rule_proxy.NewWafConfigValueNull()
	}

	return
}

// ruleProxyReadError turns a failed proxy rule read into diagnostics.
func ruleProxyReadError(httpResp *http.Response, err error, uuid string) diag.Diagnostics {
	var diags diag.Diagnostics
	if msg := parseAPIError(httpResp); msg != "" {
		diags.AddError("Failed to read rule proxy", msg)
		return diags
	}
	if httpResp != nil && httpResp.StatusCode == 404 {
		diags.AddError("Rule proxy not found",
			fmt.Sprintf("The rule proxy with UUID %s no longer exists. It may have been deleted outside of Terraform.", uuid))
		return diags
	}
	diags.AddError("Error reading rule proxy", err.Error())
	return diags
}

// setStringListFromAPI converts a []string from the API response to a types.List.
// nil slices become null lists; empty/non-empty slices become proper list values.
func setStringListFromAPI(ctx context.Context, apiVal []string, target *types.List) diag.Diagnostics {
	if apiVal == nil {
		*target = types.ListNull(types.StringType)
		return nil
	}
	listVal, diags := types.ListValueFrom(ctx, types.StringType, apiVal)
	*target = listVal
	return diags
}

// callRuleProxyUpdateAPI updates a proxy rule via the API.
func callRuleProxyUpdateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unknown rule, please update terraform state.",
		)
		return
	}

	req, d := buildProxyRequest(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	api, httpResp, err := r.client.Instance.RulesAPI.RulesProxyUpdate(
		r.client.AuthContext, r.client.Organization,
		data.Project.ValueString(), data.Uuid.ValueString(),
	).V2RuleProxyRequest(*req).Execute()

	if err != nil {
		if msg := parseAPIError(httpResp); msg != "" {
			diags.AddError("Failed to update rule", msg)
			return
		}
		diags.AddError("Failed to update rule", err.Error())
		return
	}

	// Capture UUID and RuleId from response
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())

	return
}

// callRuleProxyDeleteAPI deletes a proxy rule via the API.
func callRuleProxyDeleteAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
		return
	}

	_, err := r.client.Instance.RulesAPI.RulesProxyDelete(
		r.client.AuthContext, r.client.Organization,
		data.Project.ValueString(), data.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
