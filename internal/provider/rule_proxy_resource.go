package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_proxy"
	"terraform-provider-quant/internal/utils"

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

	// The proxy location.
	req.SetTo(data.Proxy.To.ValueString())
	req.SetHost(data.Proxy.Host.ValueString())
	req.SetCacheLifetime(int32(data.Proxy.CacheLifetime.ValueInt64()))

	if data.Proxy.AuthUser.ValueString() != "" && data.Proxy.AuthPass.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.Proxy.AuthUser.ValueString())
		req.SetAuthPass(data.Proxy.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.Proxy.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.Proxy.OnlyProxy404.ValueBool())

	if data.Failover.FailoverMode.ValueString() == "true" {
		req.SetFailoverMode("true")
	} else {
		req.SetFailoverMode("false")
	}

	// Set strip headers.
	var stripHeaders []string
	for _, header := range data.Proxy.ProxyStripHeaders.Elements() {
		stripHeaders = append(stripHeaders, header.String())
	}
	req.SetProxyStripHeaders(stripHeaders)

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

	// httpbl dictionary support.
	// httpbl := openapi.NewWAFConfigHttpblWithDefaults()
	// httpbl.SetApiKey(data.WafConfig.Httpbl.ApiKey.ValueString())
	// httpbl.SetBlockHarvester(data.WafConfig.Httpbl.BlockHarvester.ValueBool())
	// httpbl.SetBlockSearchEngine(data.WafConfig.Httpbl.BlockSearchEngine.ValueBool())
	// httpbl.SetBlockSpam(data.WafConfig.Httpbl.BlockSpam.ValueBool())
	// httpbl.SetBlockSuspicious(data.WafConfig.Httpbl.BlockSuspicious.ValueBool())
	// httpbl.SetHttpblEnabled(data.WafConfig.Httpbl.Enabled.ValueBool())
	// wafConfig.SetHttpbl(*httpbl)

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

	// API needs to return uuid.
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())

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
	req.SetTo(data.Proxy.To.ValueString())
	req.SetHost(data.Proxy.Host.ValueString())
	req.SetCacheLifetime(int32(data.Proxy.CacheLifetime.ValueInt64()))

	if data.Proxy.AuthUser.ValueString() != "" && data.Proxy.AuthPass.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.Proxy.AuthUser.ValueString())
		req.SetAuthPass(data.Proxy.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.Proxy.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.Proxy.OnlyProxy404.ValueBool())

	if data.Failover.FailoverMode.ValueString() == "true" {
		req.SetFailoverMode("true")
	} else {
		req.SetFailoverMode("false")
	}

	// Set strip headers.
	var stripHeaders []string
	for _, header := range data.Proxy.ProxyStripHeaders.Elements() {
		stripHeaders = append(stripHeaders, header.String())
	}
	req.SetProxyStripHeaders(stripHeaders)

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

	// httpbl dictionary support.
	// httpbl := openapi.NewWAFConfigUpdateHttpblWithDefaults()
	// httpbl.SetApiKey(data.WafConfig.Httpbl.ApiKey.ValueString())
	// httpbl.SetBlockHarvester(data.WafConfig.Httpbl.BlockHarvester.ValueBool())
	// httpbl.SetBlockSearchEngine(data.WafConfig.Httpbl.BlockSearchEngine.ValueBool())
	// httpbl.SetBlockSpam(data.WafConfig.Httpbl.BlockSpam.ValueBool())
	// httpbl.SetBlockSuspicious(data.WafConfig.Httpbl.BlockSuspicious.ValueBool())
	// httpbl.SetHttpblEnabled(data.WafConfig.Httpbl.Enabled.ValueBool())
	// wafConfig.SetHttpbl(*httpbl)

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
	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyDelete(r.client.AuthContext, org, data.Project.ValueString(), data.RuleId.ValueString()).Execute()

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

	data.Name = types.StringValue(api.GetName())
	data.Uuid = types.StringValue(api.GetUuid())
	domains, d := types.ListValueFrom(ctx, types.StringType, api.GetDomain())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.Domain = domains
	urls, d := types.ListValueFrom(ctx, types.StringType, api.GetUrl())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.Url = urls
	data.Ip = types.StringValue(api.GetIp())
	ips, d := types.ListValueFrom(ctx, types.StringType, api.GetIpIs())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.IpIs = types.List(ips)
	ipIsNot, d := types.ListValueFrom(ctx, types.StringType, api.GetIpIsNot())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.IpIsNot = types.List(ipIsNot)
	data.Country = types.StringValue(api.GetCountry())
	countries, d := types.ListValueFrom(ctx, types.StringType, api.GetCountryIs())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.CountryIs = types.List(countries)
	countriesNot, d := types.ListValueFrom(ctx, types.StringType, api.GetCountryIsNot())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.CountryIsNot = types.List(countriesNot)

	data.Method = types.StringValue(api.GetMethod())
	methods, d := types.ListValueFrom(ctx, types.StringType, api.GetMethodIs())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.MethodIs = types.List(methods)
	methodIsNot, d := types.ListValueFrom(ctx, types.StringType, api.GetMethodIsNot())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	data.MethodIsNot = types.List(methodIsNot)

	// Rule specific fields.
	actionConfig, ok := api.GetActionConfigOk()

	if ok {
		data.Proxy.AuthPass = types.StringValue(actionConfig.GetAuthPass())
		data.Proxy.AuthUser = types.StringValue(actionConfig.GetAuthUser())
		if actionConfig.CacheLifetime != nil {
			data.Proxy.CacheLifetime = types.Int64Value(int64(actionConfig.GetCacheLifetime()))
		}
		data.Proxy.DisableSslVerify = types.BoolValue(actionConfig.GetDisableSslVerify())
		data.Failover.FailoverMode = types.StringValue(fmt.Sprintf("%v", actionConfig.GetFailoverMode()))
		if actionConfig.FailoverLifetime != nil {
			data.Failover.FailoverLifetime = types.StringValue(actionConfig.GetFailoverLifetime())
		}
		failoverCodes, d := types.ListValueFrom(ctx, types.StringType, actionConfig.GetFailoverOriginStatusCodes())
		if d.HasError() {
			diags.Append(d...)
			return
		}
		data.Failover.FailoverOriginStatusCodes = failoverCodes
		data.Failover.FailoverOriginTtfb = types.StringValue(actionConfig.GetFailoverOriginTtfb())
		data.Proxy.Host = types.StringValue(actionConfig.GetHost())
		data.Proxy.OnlyProxy404 = types.BoolValue(actionConfig.GetOnlyProxy404())
		data.Proxy.To = types.StringValue(actionConfig.GetTo())

		proxyStripHeaders, d := types.ListValueFrom(ctx, types.StringType, actionConfig.GetProxyStripHeaders())
		data.Proxy.ProxyStripHeaders = proxyStripHeaders
		injectHeaders, d := types.MapValueFrom(ctx, types.StringType, actionConfig.GetInjectHeaders())
		data.Proxy.InjectHeaders = injectHeaders
		proxyStripRequestHeaders, d := types.ListValueFrom(ctx, types.StringType, actionConfig.GetProxyStripRequestHeaders())
		data.Proxy.ProxyStripRequestHeaders = proxyStripRequestHeaders

		// WafConfig specific read.
		wafconfig, ok := actionConfig.GetWafConfigOk()
		if !ok {
			diags.AddError("Failed to read WafConfig", "WafConfig is missing")
			return
		}
		if ok {
			if len(wafconfig.AllowRules) > 0 {
				allowRules, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetAllowRules())
				data.WafConfig.AllowRules = allowRules
			} else {
				data.WafConfig.AllowRules = types.ListNull(types.StringType)
			}
			if len(wafconfig.AllowIp) > 0 {
				allowIp, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetAllowIp())
				data.WafConfig.AllowIp = allowIp
			} else {
				data.WafConfig.AllowIp = types.ListNull(types.StringType)
			}
			if len(wafconfig.BlockIp) > 0 {
				blockIp, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetBlockIp())
				data.WafConfig.BlockIp = blockIp
			} else {
				data.WafConfig.BlockIp = types.ListNull(types.StringType)
			}
			if len(wafconfig.BlockReferer) > 0 {
				blockReferer, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetBlockReferer())
				data.WafConfig.BlockReferer = blockReferer
			} else {
				data.WafConfig.BlockReferer = types.ListNull(types.StringType)
			}
			if len(wafconfig.BlockUa) > 0 {
				blockUa, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetBlockUa())
				data.WafConfig.BlockUa = blockUa
			} else {
				data.WafConfig.BlockUa = types.ListNull(types.StringType)
			}

			// httpbl, ok := wafconfig.GetHttpblOk()
			// if ok {
			// 	rule.WafConfig.Httpbl.ApiKey = types.StringValue(httpbl.GetApiKey())
			// 	rule.WafConfig.Httpbl.BlockHarvester = types.BoolValue(httpbl.GetBlockHarvester())
			// 	rule.WafConfig.Httpbl.BlockSearchEngine = types.BoolValue(httpbl.GetBlockSearchEngine())
			// 	rule.WafConfig.Httpbl.BlockSpam = types.BoolValue(httpbl.GetBlockSpam())
			// 	rule.WafConfig.Httpbl.BlockSuspicious = types.BoolValue(httpbl.GetBlockSuspicious())
			// 	rule.WafConfig.Httpbl.Enabled = types.BoolValue(httpbl.GetHttpblEnabled())
			// } else {
			// 	rule.WafConfig.Httpbl.Enabled = types.BoolValue(false)
			// 	rule.WafConfig.Httpbl.ApiKey = types.StringValue("")
			// 	rule.WafConfig.Httpbl.BlockHarvester = types.BoolValue(false)
			// 	rule.WafConfig.Httpbl.BlockSearchEngine = types.BoolValue(false)
			// 	rule.WafConfig.Httpbl.BlockSpam = types.BoolValue(false)
			// 	rule.WafConfig.Httpbl.BlockSuspicious = types.BoolValue(false)
			// }
			data.WafConfig.IpRatelimitMode = types.StringValue(wafconfig.GetIpRatelimitMode())
			if wafconfig.IpRatelimitCooldown != nil {
				data.WafConfig.IpRatelimitCooldown = types.Int64Value(int64(wafconfig.GetIpRatelimitCooldown()))
			}
			if wafconfig.IpRatelimitRps != nil {
				data.WafConfig.IpRatelimitRps = types.Int64Value(int64(wafconfig.GetIpRatelimitRps()))
			}
			data.WafConfig.Mode = types.StringValue(wafconfig.GetMode())

			// NotifyEmail is a list of strings, so we need to convert it.
			if len(wafconfig.NotifyEmail) > 0 {
				notifyEmail, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetNotifyEmail())
				data.WafConfig.NotifyEmail = notifyEmail
			} else {
				data.WafConfig.NotifyEmail = types.ListNull(types.StringType)
			}

			data.WafConfig.NotifySlack = types.StringValue(wafconfig.GetNotifySlack())
			if wafconfig.NotifySlackHitsRpm != nil {
				data.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(wafconfig.GetNotifySlackHitsRpm()))
			}
			if wafconfig.NotifySlackRpm != nil {
				data.WafConfig.NotifySlackRpm = types.Int64Value(int64(wafconfig.GetNotifySlackRpm()))
			}
			if wafconfig.ParanoiaLevel != nil {
				data.WafConfig.ParanoiaLevel = types.Int64Value(int64(wafconfig.GetParanoiaLevel()))
			}

			if wafconfig.RequestHeaderRatelimitCooldown != nil {
				data.WafConfig.RequestHeaderRatelimitCooldown = types.Int64Value(int64(wafconfig.GetRequestHeaderRatelimitCooldown()))
			}
			if wafconfig.RequestHeaderRatelimitRps != nil {
				data.WafConfig.RequestHeaderRatelimitRps = types.Int64Value(int64(wafconfig.GetRequestHeaderRatelimitRps()))
			}
			if wafconfig.WafRatelimitCooldown != nil {
				data.WafConfig.WafRatelimitCooldown = types.Int64Value(int64(wafconfig.GetWafRatelimitCooldown()))
			}
			if wafconfig.WafRatelimitRps != nil {
				data.WafConfig.WafRatelimitRps = types.Int64Value(int64(wafconfig.GetWafRatelimitRps()))
			}
			data.WafConfig.RequestHeaderName = types.StringValue(wafconfig.GetRequestHeaderName())
			data.WafConfig.RequestHeaderRatelimitMode = types.StringValue(wafconfig.GetRequestHeaderRatelimitMode())
			data.WafConfig.WafRatelimitMode = types.StringValue(wafconfig.GetWafRatelimitMode())
		}
	}
	return
}
