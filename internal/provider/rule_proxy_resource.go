package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/mapper"
	"terraform-provider-quant/internal/resource_rule_proxy"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
			"Expected *client.Client, got: %T. Please report this issue to the provider developers",
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
	diags.Append(mapper.ToSDK(ctx, data, req)...)
	if diags.HasError() {
		return nil, diags
	}

	// Application proxy: when true, clear 'to' (backend computes it)
	appProxy := !data.ApplicationProxy.IsNull() && !data.ApplicationProxy.IsUnknown() && data.ApplicationProxy.ValueBool()
	if !appProxy {
		if !data.To.IsNull() && !data.To.IsUnknown() && data.To.ValueString() != "" {
			req.SetTo(data.To.ValueString())
		}
	}
	// Note: when appProxy is true, mapper already set ApplicationProxy/Name/Environment/Container/Port

	// Cache lifetime: string in TF schema but needs validation
	if !data.CacheLifetime.IsNull() && !data.CacheLifetime.IsUnknown() {
		cacheLifetime, err := parseCacheLifetime(data.CacheLifetime)
		if err != nil {
			diags.AddError("Invalid cache_lifetime value",
				fmt.Sprintf("Could not parse cache_lifetime: %s", err.Error()))
			return nil, diags
		}
		req.SetCacheLifetime(strconv.FormatInt(cacheLifetime, 10))
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
	if diags.HasError() {
		return nil, diags
	}

	// Inject headers (map field — not handled by mapper)
	if !data.InjectHeaders.IsNull() && !data.InjectHeaders.IsUnknown() {
		hdrs := make(map[string]string)
		diags.Append(data.InjectHeaders.ElementsAs(ctx, &hdrs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		req.SetInjectHeaders(hdrs)
	}

	// WAF configuration (nested object — not handled by mapper)
	if data.WafEnabled.ValueBool() && !data.WafConfig.IsNull() && !data.WafConfig.IsUnknown() {
		wafConfig := quantadmingo.NewWafConfig()
		wafConfig.SetMode(data.WafConfig.Mode.ValueString())
		wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

		// WAF string lists
		setWafStringList(ctx, data.WafConfig.AllowRules, wafConfig.SetAllowRules, &diags)
		setWafStringList(ctx, data.WafConfig.AllowIp, wafConfig.SetAllowIp, &diags)
		setWafStringList(ctx, data.WafConfig.BlockIp, wafConfig.SetBlockIp, &diags)
		setWafStringList(ctx, data.WafConfig.BlockUa, wafConfig.SetBlockUa, &diags)
		setWafStringList(ctx, data.WafConfig.BlockReferer, wafConfig.SetBlockReferer, &diags)
		setWafStringList(ctx, data.WafConfig.NotifyEmail, wafConfig.SetNotifyEmail, &diags)
		setWafStringList(ctx, data.WafConfig.BlockAsn, wafConfig.SetBlockAsn, &diags)
		if diags.HasError() {
			return nil, diags
		}

		if !data.WafConfig.NotifySlack.IsNull() && !data.WafConfig.NotifySlack.IsUnknown() {
			wafConfig.SetNotifySlack(data.WafConfig.NotifySlack.ValueString())
		}
		if !data.WafConfig.NotifySlackHitsRpm.IsNull() && !data.WafConfig.NotifySlackHitsRpm.IsUnknown() {
			wafConfig.SetNotifySlackHitsRpm(int32(data.WafConfig.NotifySlackHitsRpm.ValueInt64()))
		}

		// Block lists
		if !data.WafConfig.BlockLists.IsNull() && !data.WafConfig.BlockLists.IsUnknown() {
			var blockListsObj resource_rule_proxy.BlockListsValue
			diags.Append(data.WafConfig.BlockLists.As(ctx, &blockListsObj, basetypes.ObjectAsOptions{})...)
			if !diags.HasError() {
				blockLists := quantadmingo.NewWafConfigBlockLists()
				blockLists.SetAi(blockListsObj.Ai.ValueBool())
				blockLists.SetIp(blockListsObj.Ip.ValueBool())
				blockLists.SetReferer(blockListsObj.Referer.ValueBool())
				blockLists.SetUserAgent(blockListsObj.UserAgent.ValueBool())
				wafConfig.SetBlockLists(*blockLists)
			}
		}

		// Thresholds
		if !data.WafConfig.Thresholds.IsNull() && !data.WafConfig.Thresholds.IsUnknown() {
			var thresholdObjs []resource_rule_proxy.ThresholdsValue
			diags.Append(data.WafConfig.Thresholds.ElementsAs(ctx, &thresholdObjs, false)...)
			if !diags.HasError() {
				sdkThresholds := make([]quantadmingo.WafConfigThresholdsInner, len(thresholdObjs))
				for i, t := range thresholdObjs {
					th := quantadmingo.NewWafConfigThresholdsInner()
					if !t.ThresholdsType.IsNull() && !t.ThresholdsType.IsUnknown() {
						th.SetType(t.ThresholdsType.ValueString())
					}
					if !t.Mode.IsNull() && !t.Mode.IsUnknown() {
						th.SetMode(t.Mode.ValueString())
					}
					if !t.Rps.IsNull() && !t.Rps.IsUnknown() {
						th.SetRps(int32(t.Rps.ValueInt64()))
					}
					if !t.Cooldown.IsNull() && !t.Cooldown.IsUnknown() {
						th.SetCooldown(int32(t.Cooldown.ValueInt64()))
					}
					if !t.Hits.IsNull() && !t.Hits.IsUnknown() {
						th.SetHits(int32(t.Hits.ValueInt64()))
					}
					if !t.Minutes.IsNull() && !t.Minutes.IsUnknown() {
						th.SetMinutes(int32(t.Minutes.ValueInt64()))
					}
					if !t.Value.IsNull() && !t.Value.IsUnknown() {
						th.SetValue(t.Value.ValueString())
					}
					if !t.NotifySlack.IsNull() && !t.NotifySlack.IsUnknown() {
						th.SetNotifySlack(t.NotifySlack.ValueString())
					}
					sdkThresholds[i] = *th
				}
				wafConfig.SetThresholds(sdkThresholds)
			}
		}

		// HTTPBL
		if !data.WafConfig.Httpbl.IsNull() && !data.WafConfig.Httpbl.IsUnknown() {
			var httpblObj resource_rule_proxy.HttpblValue
			diags.Append(data.WafConfig.Httpbl.As(ctx, &httpblObj, basetypes.ObjectAsOptions{})...)
			if !diags.HasError() {
				httpbl := quantadmingo.NewWafConfigHttpbl()
				httpbl.SetHttpblEnabled(httpblObj.HttpblEnabled.ValueBool())
				httpbl.SetHttpblKey(httpblObj.HttpblKey.ValueString())
				httpbl.SetBlockHarvester(httpblObj.BlockHarvester.ValueBool())
				httpbl.SetBlockSearchEngine(httpblObj.BlockSearchEngine.ValueBool())
				httpbl.SetBlockSpam(httpblObj.BlockSpam.ValueBool())
				httpbl.SetBlockSuspicious(httpblObj.BlockSuspicious.ValueBool())
				wafConfig.SetHttpbl(*httpbl)
			}
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
	prevApplicationProxy := data.ApplicationProxy
	prevApplicationName := data.ApplicationName
	prevApplicationEnvironment := data.ApplicationEnvironment
	prevApplicationContainer := data.ApplicationContainer
	prevApplicationPort := data.ApplicationPort
	prevTo := data.To

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
		if msg := parseAPIError(httpResp); msg != "" {
			diags.AddError("Failed to read rule proxy", msg)
			return
		}
		if httpResp != nil && httpResp.StatusCode == 404 {
			diags.AddError("Rule proxy not found",
				fmt.Sprintf("The rule proxy with UUID %s no longer exists. It may have been deleted outside of Terraform.",
					data.Uuid.ValueString()))
			return
		}
		diags.AddError("Error reading rule proxy", err.Error())
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

	// InjectHeaders: read from action config
	if actionConfig.HasInjectHeaders() {
		hdrs := actionConfig.GetInjectHeaders()
		if len(hdrs) > 0 {
			hdrMap := make(map[string]attr.Value, len(hdrs))
			for k, v := range hdrs {
				hdrMap[k] = types.StringValue(v)
			}
			data.InjectHeaders, _ = types.MapValue(types.StringType, hdrMap)
		} else {
			data.InjectHeaders = types.MapNull(types.StringType)
		}
	} else {
		data.InjectHeaders = types.MapNull(types.StringType)
	}

	// OnlyWithCookie: mapper sets it from API, but we need null for empty string
	if api.OnlyWithCookie == nil || *api.OnlyWithCookie == "" {
		data.OnlyWithCookie = types.StringNull()
	}

	// Handle proxy configuration from ActionConfig
	// When application_proxy is true, API auto-generates the 'to' field — keep the original value
	if !prevApplicationProxy.IsNull() && prevApplicationProxy.ValueBool() {
		data.To = prevTo
	} else {
		data.To = types.StringValue(actionConfig.GetTo())
	}
	if h := actionConfig.GetHost(); h != "" {
		data.Host = types.StringValue(h)
	} else {
		data.Host = types.StringNull()
	}

	// Origin timeout
	if v, ok := actionConfig.GetOriginTimeoutOk(); ok {
		formatted := fmt.Sprintf("%v", *v)
		if formatted == "" || formatted == "0" {
			data.OriginTimeout = types.StringNull()
		} else {
			data.OriginTimeout = types.StringValue(formatted)
		}
	} else {
		data.OriginTimeout = types.StringNull()
	}

	// Application proxy fields are request-only; preserve from prior state
	if !prevApplicationProxy.IsNull() && !prevApplicationProxy.IsUnknown() {
		data.ApplicationProxy = prevApplicationProxy
	} else {
		data.ApplicationProxy = types.BoolValue(false)
	}
	if !prevApplicationName.IsNull() && !prevApplicationName.IsUnknown() {
		data.ApplicationName = prevApplicationName
	} else {
		data.ApplicationName = types.StringNull()
	}
	if !prevApplicationEnvironment.IsNull() && !prevApplicationEnvironment.IsUnknown() {
		data.ApplicationEnvironment = prevApplicationEnvironment
	} else {
		data.ApplicationEnvironment = types.StringNull()
	}
	if !prevApplicationContainer.IsNull() && !prevApplicationContainer.IsUnknown() {
		data.ApplicationContainer = prevApplicationContainer
	} else {
		data.ApplicationContainer = types.StringNull()
	}
	if !prevApplicationPort.IsNull() && !prevApplicationPort.IsUnknown() {
		data.ApplicationPort = prevApplicationPort
	} else {
		data.ApplicationPort = types.Int64Null()
	}

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

	// ProxyStripHeaders / ProxyStripRequestHeaders from ActionConfig
	diags.Append(setStringListFromAPI(ctx, actionConfig.GetProxyStripHeaders(), &data.ProxyStripHeaders)...)
	diags.Append(setStringListFromAPI(ctx, actionConfig.GetProxyStripRequestHeaders(), &data.ProxyStripRequestHeaders)...)

	// Failover configuration from ActionConfig
	data.FailoverMode = types.BoolValue(actionConfig.GetFailoverMode())
	if v := actionConfig.GetFailoverLifetime(); v != "" && v != "0" {
		data.FailoverLifetime = types.StringValue(v)
	} else if data.FailoverLifetime.IsUnknown() {
		data.FailoverLifetime = types.StringNull()
	}
	if v := actionConfig.GetFailoverOriginTtfb(); v != "" && v != "0" {
		data.FailoverOriginTtfb = types.StringValue(v)
	} else if data.FailoverOriginTtfb.IsUnknown() {
		data.FailoverOriginTtfb = types.StringNull()
	}
	failoverStatusCodes := actionConfig.GetFailoverOriginStatusCodes()
	if failoverStatusCodes == nil {
		data.FailoverOriginStatusCodes = types.ListNull(types.StringType)
	} else {
		codesList, d := types.ListValueFrom(ctx, types.StringType, failoverStatusCodes)
		diags.Append(d...)
		data.FailoverOriginStatusCodes = codesList
	}

	// WAF configuration (nested object)
	data.WafEnabled = types.BoolValue(actionConfig.GetWafEnabled())
	if data.WafEnabled.ValueBool() {
		wafConfig := actionConfig.GetWafConfig()
		data.WafConfig.Mode = types.StringValue(wafConfig.GetMode())
		data.WafConfig.ParanoiaLevel = types.Int64Value(int64(wafConfig.GetParanoiaLevel()))

		// WAF string lists
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetAllowRules(), &data.WafConfig.AllowRules)...)
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetAllowIp(), &data.WafConfig.AllowIp)...)
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockIp(), &data.WafConfig.BlockIp)...)
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockUa(), &data.WafConfig.BlockUa)...)
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockReferer(), &data.WafConfig.BlockReferer)...)
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetBlockAsn(), &data.WafConfig.BlockAsn)...)
		diags.Append(setStringListFromAPI(ctx, wafConfig.GetNotifyEmail(), &data.WafConfig.NotifyEmail)...)

		data.WafConfig.NotifySlack = types.StringValue(wafConfig.GetNotifySlack())
		data.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(wafConfig.GetNotifySlackHitsRpm()))

		// Block lists: preserve from prior state to avoid custom type conversion issues.
		// User-configured block_lists will round-trip via the Plan, not the Read.
		if data.WafConfig.BlockLists.IsUnknown() {
			data.WafConfig.BlockLists = types.ObjectNull(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx))
		}

		// HTTPBL: preserve from prior state
		if data.WafConfig.Httpbl.IsUnknown() {
			data.WafConfig.Httpbl = types.ObjectNull(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx))
		}

		// Thresholds: preserve from prior state
		if data.WafConfig.Thresholds.IsUnknown() {
			thObjType := types.ObjectType{AttrTypes: resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx)}
			data.WafConfig.Thresholds, _ = types.ListValue(thObjType, []attr.Value{})
		}
	} else {
		// WAF disabled: if waf_config block exists in config, populate with defaults
		if !data.WafConfig.Mode.IsNull() && !data.WafConfig.Mode.IsUnknown() {
			emptyStringList, _ := types.ListValue(types.StringType, []attr.Value{})
			data.WafConfig.AllowRules = emptyStringList
			data.WafConfig.AllowIp = emptyStringList
			data.WafConfig.BlockIp = emptyStringList
			data.WafConfig.BlockUa = emptyStringList
			data.WafConfig.BlockReferer = emptyStringList
			data.WafConfig.NotifyEmail = emptyStringList
			data.WafConfig.NotifySlack = types.StringValue("")
			data.WafConfig.NotifySlackHitsRpm = types.Int64Null()
			data.WafConfig.BlockAsn = emptyStringList
			data.WafConfig.Httpbl = types.ObjectNull(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx))
			data.WafConfig.BlockLists = types.ObjectNull(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx))
			thObjTypeDisabled := types.ObjectType{AttrTypes: resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx)}
			data.WafConfig.Thresholds, _ = types.ListValue(thObjTypeDisabled, []attr.Value{})
			if data.WafConfig.ParanoiaLevel.IsNull() || data.WafConfig.ParanoiaLevel.IsUnknown() {
				data.WafConfig.ParanoiaLevel = types.Int64Value(1)
			}
		}
	}

	// Static error page fields from ActionConfig
	if v := actionConfig.GetStaticErrorPage(); v != "" {
		data.StaticErrorPage = types.StringValue(v)
	} else if data.StaticErrorPage.IsUnknown() {
		data.StaticErrorPage = types.StringNull()
	}
	if codes := actionConfig.GetStaticErrorPageStatusCodes(); codes != nil && len(codes) > 0 {
		diags.Append(setStringListFromAPI(ctx, codes, &data.StaticErrorPageStatusCodes)...)
	} else if data.StaticErrorPageStatusCodes.IsUnknown() {
		data.StaticErrorPageStatusCodes = types.ListNull(types.StringType)
	}

	// Set action_config to null since we expose its fields as top-level attributes
	data.ActionConfig = resource_rule_proxy.NewActionConfigValueNull()

	// If waf_config wasn't properly populated, set to null
	if data.WafConfig.IsUnknown() || data.WafConfig.Mode.IsUnknown() {
		data.WafConfig = resource_rule_proxy.NewWafConfigValueNull()
	}

	return
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

	// CRITICAL: UUID changes after every update — must capture the new UUID
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
