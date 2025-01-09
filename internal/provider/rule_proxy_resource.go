package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_proxy"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
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

func (r *ruleProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_proxy"
}

func (r *ruleProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_rule_proxy.RuleProxyResourceSchema(ctx)
}

func (r *ruleProxyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ruleProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_rule_proxy.RuleProxyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	resp.Diagnostics.Append(callRuleProxyCreateAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_proxy.RuleProxyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resource_rule_proxy.RuleProxyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	// Update API call logic
	resp.Diagnostics.Append(callRuleProxyUpdateAPI(ctx, r, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_rule_proxy.RuleProxyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callRuleProxyDeleteAPI(ctx, r, &data)...)
}

func (r *ruleProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_proxy.RuleProxyModel
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
	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleProxyCreateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	req := *openapi.NewRuleProxyRequestWithDefaults()
	req.SetName(data.Name.ValueString())

	var domains []string
	for _, domain := range data.Domain.Elements() {
		if strVal, ok := domain.(types.String); ok {
			domains = append(domains, strVal.ValueString())
		}
	}
	req.SetDomain(domains)

	var urls []string
	for _, url := range data.Url.Elements() {
		if strVal, ok := url.(types.String); ok {
			urls = append(urls, strVal.ValueString())
		}
	}
	req.SetUrl(urls)

	req.SetCountry(data.Country.ValueString())
	var countryList []string
	if data.Country.ValueString() == "country_is" {
		for _, country := range data.CountryIs.Elements() {
			if strVal, ok := country.(types.String); ok {
				countryList = append(countryList, strVal.ValueString())
			}
		}
		req.SetCountryIs(countryList)
	} else if data.Country.ValueString() == "country_is_not" {
		for _, country := range data.CountryIsNot.Elements() {
			if strVal, ok := country.(types.String); ok {
				countryList = append(countryList, strVal.ValueString())
			}
		}
		req.SetCountryIsNot(countryList)
	}

	req.SetIp(data.Ip.ValueString())
	var iplist []string

	if data.Ip.ValueString() == "ip_is" {
		for _, ip := range data.IpIs.Elements() {
			if strVal, ok := ip.(types.String); ok {
				iplist = append(iplist, strVal.ValueString())
			}
		}
		req.SetIpIs(iplist)
	} else if data.Ip.ValueString() == "ip_is_not" {
		for _, ip := range data.IpIsNot.Elements() {
			if strVal, ok := ip.(types.String); ok {
				iplist = append(iplist, strVal.ValueString())
			}
		}
		req.SetIpIsNot(iplist)
	}

	req.SetMethod(data.Method.ValueString())
	var methodList []string

	if data.Method.ValueString() == "method_is" {
		for _, method := range data.MethodIs.Elements() {
			if strVal, ok := method.(types.String); ok {
				methodList = append(methodList, strVal.ValueString())
			}
		}
		req.SetMethodIs(methodList)
	} else if data.Method.ValueString() == "method_is_not" {
		for _, method := range data.MethodIsNot.Elements() {
			if strVal, ok := method.(types.String); ok {
				methodList = append(methodList, strVal.ValueString())
			}
		}
		req.SetMethodIsNot(methodList)
	}

	proxy := openapi.NewProxyConfigWithDefaults()
	proxy.SetTo(data.Proxy.To.ValueString())
	proxy.SetHost(data.Proxy.Host.ValueString())
	proxy.SetCacheLifetime(int32(data.Proxy.CacheLifetime.ValueInt64()))

	if data.Proxy.AuthUser.ValueString() != "" && data.Proxy.AuthPass.ValueString() != "" {
		// Only set basic auth details if we have both.
		proxy.SetAuthUser(data.Proxy.AuthUser.ValueString())
		proxy.SetAuthPass(data.Proxy.AuthPass.ValueString())
	}

	// Set strip headers.
	var stripHeaders []string
	for _, header := range data.Proxy.ProxyStripHeaders.Elements() {
		stripHeaders = append(stripHeaders, header.String())
	}
	proxy.SetProxyStripHeaders(stripHeaders)

	proxy.SetDisableSslVerify(data.Proxy.DisableSslVerify.ValueBool())
	proxy.SetOnlyProxy404(data.Proxy.OnlyProxy404.ValueBool())

	req.SetProxy(*proxy)

	failover := openapi.NewFailoverConfigWithDefaults()
	failover.SetFailoverMode(data.Failover.FailoverMode.ValueString())
	failover.SetFailoverOriginTtfb(data.Failover.FailoverOriginTtfb.ValueString())

	var statusCodes []string
	for _, code := range data.Failover.FailoverOriginStatusCodes.Elements() {
		if strVal, ok := code.(types.String); ok {
			statusCodes = append(statusCodes, strVal.ValueString())
		}
	}
	failover.SetFailoverOriginStatusCodes(statusCodes)

	req.SetFailover(*failover)

	req.SetWafEnabled(data.WafEnabled.ValueBool())

	wafConfig := openapi.NewWAFConfigWithDefaults()

	wafConfig.SetMode(data.WafConfig.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

	// Set WAF rules.
	var allowRules []string
	for _, r := range data.WafConfig.AllowRules.Elements() {
		if strVal, ok := r.(types.String); ok {
			allowRules = append(allowRules, strVal.ValueString())
		}
	}
	wafConfig.SetAllowRules(allowRules)

	var allowIp []string
	for _, ip := range data.WafConfig.AllowIp.Elements() {
		if strVal, ok := ip.(types.String); ok {
			allowIp = append(allowIp, strVal.ValueString())
		}
	}
	wafConfig.SetAllowIp(allowIp)

	var blockIp []string
	for _, ip := range data.WafConfig.BlockIp.Elements() {
		if strVal, ok := ip.(types.String); ok {
			blockIp = append(blockIp, strVal.ValueString())
		}
	}
	wafConfig.SetBlockIp(blockIp)

	var blockUserAgent []string
	for _, ua := range data.WafConfig.BlockUa.Elements() {
		if strVal, ok := ua.(types.String); ok {
			blockUserAgent = append(blockUserAgent, strVal.ValueString())
		}
	}
	wafConfig.SetBlockUa(blockUserAgent)

	var blockReferer []string
	for _, referer := range data.WafConfig.BlockReferer.Elements() {
		if strVal, ok := referer.(types.String); ok {
			blockReferer = append(blockReferer, strVal.ValueString())
		}
	}
	wafConfig.SetBlockReferer(blockReferer)

	var emails []string
	for _, email := range data.WafConfig.NotifyEmail.Elements() {
		if e, ok := email.(types.String); ok {
			emails = append(emails, e.ValueString())
		}
	}
	wafConfig.SetNotifyEmail(emails)

	req.SetWafConfig(*wafConfig)

	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyCreate(r.client.AuthContext, r.client.Organization, data.Project.ValueString()).RuleProxyRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to create rule proxy", err.Error())
		return
	}

	data.Uuid = types.StringValue(api.GetUuid())

	readDiags := callRuleProxyReadAPI(ctx, r, data)
	if readDiags.HasError() {
		diags.Append(readDiags...)
		return
	}

	return
}

func callRuleProxyUpdateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	req := *openapi.NewRuleProxyRequestUpdateWithDefaults()

	var domains []string
	for _, domain := range data.Domain.Elements() {
		if strVal, ok := domain.(types.String); ok {
			domains = append(domains, strVal.ValueString())
		}
	}
	req.SetDomain(domains)

	var urls []string
	for _, url := range data.Url.Elements() {
		if strVal, ok := url.(types.String); ok {
			urls = append(urls, strVal.ValueString())
		}
	}
	req.SetUrl(urls)

	req.SetCountry(data.Country.ValueString())
	var countryList []string
	if data.Country.ValueString() == "country_is" {
		for _, country := range data.CountryIs.Elements() {
			if strVal, ok := country.(types.String); ok {
				countryList = append(countryList, strVal.ValueString())
			}
		}
		req.SetCountryIs(countryList)
	} else if data.Country.ValueString() == "country_is_not" {
		for _, country := range data.CountryIsNot.Elements() {
			if strVal, ok := country.(types.String); ok {
				countryList = append(countryList, strVal.ValueString())
			}
		}
		req.SetCountryIsNot(countryList)
	}

	req.SetIp(data.Ip.ValueString())
	var iplist []string

	if data.Ip.ValueString() == "ip_is" {
		for _, ip := range data.IpIs.Elements() {
			if strVal, ok := ip.(types.String); ok {
				iplist = append(iplist, strVal.ValueString())
			}
		}
		req.SetIpIs(iplist)
	} else if data.Ip.ValueString() == "ip_is_not" {
		for _, ip := range data.IpIsNot.Elements() {
			if strVal, ok := ip.(types.String); ok {
				iplist = append(iplist, strVal.ValueString())
			}
		}
		req.SetIpIsNot(iplist)
	}

	req.SetMethod(data.Method.ValueString())
	var methodList []string

	if data.Method.ValueString() == "method_is" {
		for _, method := range data.MethodIs.Elements() {
			if strVal, ok := method.(types.String); ok {
				methodList = append(methodList, strVal.ValueString())
			}
		}
		req.SetMethodIs(methodList)
	} else if data.Method.ValueString() == "method_is_not" {
		for _, method := range data.MethodIsNot.Elements() {
			if strVal, ok := method.(types.String); ok {
				methodList = append(methodList, strVal.ValueString())
			}
		}
		req.SetMethodIsNot(methodList)
	}

	// The proxy location.
	proxy := openapi.NewProxyConfigUpdateWithDefaults()
	proxy.SetTo(data.Proxy.To.ValueString())
	proxy.SetHost(data.Proxy.Host.ValueString())
	proxy.SetCacheLifetime(int32(data.Proxy.CacheLifetime.ValueInt64()))

	if data.Proxy.AuthUser.ValueString() != "" && data.Proxy.AuthPass.ValueString() != "" {
		// Only set basic auth details if we have both.
		proxy.SetAuthUser(data.Proxy.AuthUser.ValueString())
		proxy.SetAuthPass(data.Proxy.AuthPass.ValueString())
	}

	proxy.SetDisableSslVerify(data.Proxy.DisableSslVerify.ValueBool())
	proxy.SetOnlyProxy404(data.Proxy.OnlyProxy404.ValueBool())

	req.SetProxy(*proxy)

	failover := openapi.NewFailoverConfigWithDefaults()
	failover.SetFailoverMode(data.Failover.FailoverMode.ValueString())
	failover.SetFailoverOriginTtfb(data.Failover.FailoverOriginTtfb.ValueString())
	// Set strip headers.
	var stripHeaders []string
	for _, header := range data.Proxy.ProxyStripHeaders.Elements() {
		stripHeaders = append(stripHeaders, header.String())
	}
	proxy.SetProxyStripHeaders(stripHeaders)
	req.SetFailover(*failover)

	req.SetWafEnabled(data.WafEnabled.ValueBool())

	wafConfig := openapi.NewWAFConfigUpdateWithDefaults()

	wafConfig.SetMode(data.WafConfig.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

	// Set WAF rules.
	var allowRules []string
	for _, r := range data.WafConfig.AllowRules.Elements() {
		if strVal, ok := r.(types.String); ok {
			allowRules = append(allowRules, strVal.ValueString())
		}
	}
	wafConfig.SetAllowRules(allowRules)

	var allowIp []string
	for _, ip := range data.WafConfig.AllowIp.Elements() {
		if strVal, ok := ip.(types.String); ok {
			allowIp = append(allowIp, strVal.ValueString())
		}
	}
	wafConfig.SetAllowIp(allowIp)

	var blockIp []string
	for _, ip := range data.WafConfig.BlockIp.Elements() {
		if strVal, ok := ip.(types.String); ok {
			blockIp = append(blockIp, strVal.ValueString())
		}
	}
	wafConfig.SetBlockIp(blockIp)

	var blockUserAgent []string
	for _, ua := range data.WafConfig.BlockUa.Elements() {
		if strVal, ok := ua.(types.String); ok {
			blockUserAgent = append(blockUserAgent, strVal.ValueString())
		}
	}
	wafConfig.SetBlockUa(blockUserAgent)

	var blockReferer []string
	for _, referer := range data.WafConfig.BlockReferer.Elements() {
		if strVal, ok := referer.(types.String); ok {
			blockReferer = append(blockReferer, strVal.ValueString())
		}
	}
	wafConfig.SetBlockReferer(blockReferer)

	var emails []string
	for _, email := range data.WafConfig.NotifyEmail.Elements() {
		if e, ok := email.(types.String); ok {
			emails = append(emails, e.ValueString())
		}
	}
	wafConfig.SetNotifyEmail(emails)
	req.SetWafConfig(*wafConfig)

	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyUpdate(r.client.AuthContext, org, data.Project.ValueString(), data.RuleId.ValueString()).RuleProxyRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule proxy", err.Error())
		return
	}

	// After successful update, read the resource to ensure state is consistent
	readDiags := callRuleProxyReadAPI(ctx, r, data)
	if readDiags.HasError() {
		diags.Append(readDiags...)
		return
	}

	return
}

func callRuleProxyDeleteAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	_, err := r.client.Instance.RulesProxyAPI.RulesProxyDelete(r.client.AuthContext, org, data.Project.ValueString(), data.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule proxy", err.Error())
		return
	}

	return
}

func callRuleProxyReadAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyRead(r.client.AuthContext, org, data.Project.ValueString(), data.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddWarning("Details: ", fmt.Sprintf("Org: %v, Project: %v, Uuid: %v", org, data.Project.ValueString(), data.RuleId.ValueString()))
		diags.AddError("Failed to read rule", err.Error())
		return
	}

	// Set required fields from API response
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())
	data.Organization = types.StringValue(r.client.Organization)
	data.Action = types.StringValue("proxy")
	data.Weight = types.Int64Value(0)
	data.Name = types.StringValue(api.GetName())
	data.CookieName = types.StringValue(api.GetOnlyWithCookie())
	data.Rule = types.StringNull()

	// Convert API domain list to types.List
	domainList, diag := types.ListValueFrom(ctx, types.StringType, api.Domain)
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	data.Domain = domainList

	// Convert API URL list to types.List
	urlList, diag := types.ListValueFrom(ctx, types.StringType, api.Url)
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	data.Url = urlList

	// Initialize empty lists for optional fields
	emptyStringList, _ := types.ListValueFrom(ctx, types.StringType, []string{})

	// Handle Method fields
	if api.Method != nil && *api.Method != "" {
		data.Method = types.StringValue(*api.Method)
		if len(api.MethodIs) > 0 {
			data.MethodIs, _ = types.ListValueFrom(ctx, types.StringType, api.MethodIs)
		} else {
			data.MethodIs = emptyStringList
		}
		if len(api.MethodIsNot) > 0 {
			data.MethodIsNot, _ = types.ListValueFrom(ctx, types.StringType, api.MethodIsNot)
		} else {
			data.MethodIsNot = emptyStringList
		}
	} else {
		data.Method = types.StringNull()
		data.MethodIs = emptyStringList
		data.MethodIsNot = emptyStringList
	}

	// Handle Country fields
	if api.Country != nil && *api.Country != "" {
		data.Country = types.StringValue(*api.Country)
		if len(api.CountryIs) > 0 {
			data.CountryIs, _ = types.ListValueFrom(ctx, types.StringType, api.CountryIs)
		} else {
			data.CountryIs = emptyStringList
		}
		if len(api.CountryIsNot) > 0 {
			data.CountryIsNot, _ = types.ListValueFrom(ctx, types.StringType, api.CountryIsNot)
		} else {
			data.CountryIsNot = emptyStringList
		}
	} else {
		data.Country = types.StringNull()
		data.CountryIs = emptyStringList
		data.CountryIsNot = emptyStringList
	}

	// Handle IP fields
	if api.Ip != nil && *api.Ip != "" {
		data.Ip = types.StringValue(*api.Ip)
		if len(api.IpIs) > 0 {
			data.IpIs, _ = types.ListValueFrom(ctx, types.StringType, api.IpIs)
		} else {
			data.IpIs = emptyStringList
		}
		if len(api.IpIsNot) > 0 {
			data.IpIsNot, _ = types.ListValueFrom(ctx, types.StringType, api.IpIsNot)
		} else {
			data.IpIsNot = emptyStringList
		}
	} else {
		data.Ip = types.StringNull()
		data.IpIs = emptyStringList
		data.IpIsNot = emptyStringList
	}

	// Set proxy configuration
	actionConfig := api.GetActionConfig()
	proxy := actionConfig.Proxy
	data.Proxy.AuthUser = types.StringValue(proxy.GetAuthUser())
	data.Proxy.AuthPass = types.StringValue(proxy.GetAuthPass())
	data.Proxy.CacheLifetime = types.Int64Value(int64(proxy.GetCacheLifetime()))
	data.Proxy.DisableSslVerify = types.BoolValue(proxy.GetDisableSslVerify())
	data.Proxy.OnlyProxy404 = types.BoolValue(proxy.GetOnlyProxy404())

	// Set WAF configuration
	data.WafConfig.NotifySlack = types.StringValue(api.GetActionConfig().WafConfig.GetNotifySlack())
	data.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(api.GetActionConfig().WafConfig.GetNotifySlackHitsRpm()))
	data.WafConfig.NotifySlackRpm = types.Int64Value(int64(api.GetActionConfig().WafConfig.GetNotifySlackRpm()))
	data.WafConfig.RequestHeaderName = types.StringValue(api.GetActionConfig().WafConfig.GetRequestHeaderName())

	if len(api.GetActionConfig().WafConfig.GetThresholds()) > 0 {
		thresholdObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cooldown":     types.Int64Type,
				"hits":         types.Int64Type,
				"minutes":      types.Int64Type,
				"mode":         types.StringType,
				"notify_slack": types.StringType,
				"rps":          types.Int64Type,
				"type":         types.StringType,
				"value":        types.StringType,
			},
		}
		data.WafConfig.Thresholds, _ = types.ListValueFrom(ctx, thresholdObjType, api.GetActionConfig().WafConfig.GetThresholds())
	} else {
		data.WafConfig.Thresholds = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"cooldown":     types.Int64Type,
				"hits":         types.Int64Type,
				"minutes":      types.Int64Type,
				"mode":         types.StringType,
				"notify_slack": types.StringType,
				"rps":          types.Int64Type,
				"type":         types.StringType,
				"value":        types.StringType,
			},
		})
	}

	if len(actionConfig.WafConfig.GetBlockIp()) > 0 {
		data.WafConfig.BlockIp, _ = types.ListValueFrom(ctx, types.StringType, actionConfig.WafConfig.GetBlockIp())
	} else {
		data.WafConfig.BlockIp = emptyStringList
	}

	if len(actionConfig.WafConfig.GetBlockUa()) > 0 {
		data.WafConfig.BlockUa, _ = types.ListValueFrom(ctx, types.StringType, actionConfig.WafConfig.GetBlockUa())
	} else {
		data.WafConfig.BlockUa = emptyStringList
	}

	if len(actionConfig.WafConfig.GetBlockReferer()) > 0 {
		data.WafConfig.BlockReferer, _ = types.ListValueFrom(ctx, types.StringType, actionConfig.WafConfig.GetBlockReferer())
	} else {
		data.WafConfig.BlockReferer = emptyStringList
	}

	if len(actionConfig.WafConfig.GetNotifyEmail()) > 0 {
		data.WafConfig.NotifyEmail, _ = types.ListValueFrom(ctx, types.StringType, actionConfig.WafConfig.GetNotifyEmail())
	} else {
		data.WafConfig.NotifyEmail = emptyStringList
	}

	// Set notification config
	if actionConfig.NotifyConfig != nil {
		if len(actionConfig.NotifyConfig.GetOriginStatusCodes()) > 0 {
			data.NotifyConfig.OriginStatusCodes, _ = types.ListValueFrom(ctx, types.StringType, actionConfig.NotifyConfig.GetOriginStatusCodes())
		} else {
			data.NotifyConfig.OriginStatusCodes = emptyStringList
		}
		data.NotifyConfig.Period = types.StringValue(actionConfig.NotifyConfig.GetPeriod())
		data.NotifyConfig.SlackWebhook = types.StringValue(actionConfig.NotifyConfig.GetSlackWebhook())
	} else {
		data.NotifyConfig = resource_rule_proxy.NotifyConfigValue{}
		data.NotifyConfig.OriginStatusCodes = emptyStringList
		data.NotifyConfig.Period = types.StringNull()
		data.NotifyConfig.SlackWebhook = types.StringNull()
	}

	// Set failover config
	failover := actionConfig.GetFailover()

	data.Failover.FailoverMode = types.StringValue(fmt.Sprintf("%v", *failover.FailoverMode))
	data.Failover.FailoverLifetime = types.StringValue(*failover.FailoverLifetime)
	if len(failover.FailoverOriginStatusCodes) > 0 {
		data.Failover.FailoverOriginStatusCodes, _ = types.ListValueFrom(ctx, types.StringType, failover.FailoverOriginStatusCodes)
	} else {
		data.Failover.FailoverOriginStatusCodes = emptyStringList
	}

	// Initialize empty lists for optional fields
	data.Failover = resource_rule_proxy.FailoverValue{
		FailoverMode:              types.StringValue("false"),
		FailoverLifetime:          types.StringValue(""),
		FailoverOriginStatusCodes: types.ListValueMust(types.StringType, []attr.Value{}),
	}

	data.NotifyConfig = resource_rule_proxy.NotifyConfigValue{
		OriginStatusCodes: types.ListValueMust(types.StringType, []attr.Value{}),
		Period:            types.StringValue(""),
		SlackWebhook:      types.StringValue(""),
	}

	thresholdObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"cooldown":     types.Int64Type,
			"hits":         types.Int64Type,
			"minutes":      types.Int64Type,
			"mode":         types.StringType,
			"notify_slack": types.StringType,
			"rps":          types.Int64Type,
			"type":         types.StringType,
			"value":        types.StringType,
		},
	}
	data.WafConfig.Thresholds = types.ListValueMust(thresholdObjType, []attr.Value{})
	return
}
