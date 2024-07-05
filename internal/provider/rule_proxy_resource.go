package provider

import (
	"context"
	"strconv"
	"strings"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_proxy"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource = (*ruleProxyResource)(nil)
	_ resource.ResourceWithConfigure = (*ruleProxyResource)(nil)
	_ resource.ResourceWithImportState = (*ruleProxyResource)(nil)
)

func NewRuleProxyResource() resource.Resource {
	return &ruleProxyResource{}
}

type ruleProxyResource struct{
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
	var data resource_rule_proxy.RuleProxyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	resp.Diagnostics.Append(callRuleProxyUpdateAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

	id := strings.Split(req.ID, "/")

	if len(id) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected ID in the format 'project/UUID'")
		return
	}

	// Import API call logic
	data.Project = types.StringValue(id[0])
	data.Uuid = types.StringValue(id[1])

	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save imported data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleProxyCreateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
    req := *openapi.NewRuleProxyRequestWithDefaults()
    req.SetName(data.Name.ValueString())
    req.SetDomain(data.Domain.ValueString())
    // @todo: this should be a list of URLs
    var urls []string
    data.Url.ElementsAs(ctx, urls, false)
	req.SetUrl(urls)

	req.SetCountry(data.Country.ValueString())
	var countryList []string

	if data.Country.ValueString() == "country_is" {
		data.CountryIs.ElementsAs(ctx, countryList, false)
		req.SetCountryIs(countryList)
	} else if data.Country.ValueString() == "country_is_not" {
		data.CountryIsNot.ElementsAs(ctx, countryList, false)
		req.SetCountryIsNot(countryList)
	}

	req.SetIp(data.Ip.ValueString())
	var iplist []string

	if data.Ip.ValueString() == "ip_is" {
		data.IpIs.ElementsAs(ctx, iplist, false)
		req.SetIpIs(iplist)
	} else if data.Ip.ValueString() == "ip_is_not" {
		data.IpIsNot.ElementsAs(ctx, iplist, false)
		req.SetIpIsNot(iplist)
	}

	req.SetMethod(data.Method.ValueString())
	var methodList []string

	if data.Method.ValueString() == "method_is" {
		data.MethodIs.ElementsAs(ctx, methodList, false)
		req.SetMethodIs(methodList)
	} else if data.Method.ValueString() == "method_is_not" {
		data.MethodIsNot.ElementsAs(ctx, methodList, false)
		req.SetMethodIsNot(methodList)
	}

	// The proxy location.
	req.SetTo(data.To.ValueString())
	req.SetHost(data.Host.ValueString())

	// @todo: this should probably accept a string?
    cl := strconv.FormatInt(data.CacheLifetime.ValueInt64(), 10)
	req.SetCacheLifetime(cl)

	if data.AuthPass.ValueString() != "" && data.AuthUser.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	req.SetFailoverMode(data.FailoverMode.ValueString())

	// Set strip headers.
	var stripHeaders []string
	data.ProxyStripHeaders.ElementsAs(ctx, stripHeaders, false)
	req.SetProxyStripHeaders(stripHeaders)

	req.SetWafEnabled(data.WafEnabled.ValueBool())

	// Build WAF config.
	wafConfig := *openapi.NewWAFConfigWithDefaults()
	wafConfig.SetMode(data.WafConfig.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

	// Set WAF rules.
	var allowRules []string
	data.WafConfig.AllowRules.ElementsAs(ctx, allowRules, false)
	wafConfig.SetAllowRules(allowRules)

	var allowIp []string
	data.WafConfig.AllowIp.ElementsAs(ctx, allowIp, false)
	wafConfig.SetAllowIp(allowIp)

	var blockIp []string
	data.WafConfig.BlockIp.ElementsAs(ctx, blockIp, false)
	wafConfig.SetBlockIp(blockIp)

	var blockUserAgent []string
	data.WafConfig.BlockUa.ElementsAs(ctx, blockUserAgent, false)
	wafConfig.SetBlockUa(blockUserAgent)

	var blockReferer []string
	data.WafConfig.BlockReferer.ElementsAs(ctx, blockReferer, false)
	wafConfig.SetBlockReferer(blockReferer)

	// httpbl dictionary support.
	// @todo support httpbl.

	notify := *openapi.NewNotifyConfigWithDefaults()
	notify.SetPeriod(data.NotifyConfig.Period.ValueString())
	notify.SetSlackWebhook(data.NotifyConfig.SlackWebhook.ValueString())

	var originStatusCodes []string
	data.NotifyConfig.OriginStatusCodes.ElementsAs(ctx, originStatusCodes, true)
	notify.SetOriginStatusCodes(originStatusCodes)

	req.SetNotifyConfig(notify)
	req.SetWafConfig(wafConfig)

	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyCreate(r.client.AuthContext, data.Organization.ValueString(), data.Project.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to create rule proxy", err.Error())
		return
	}

	// API needs to return uuid.
    data.Uuid = types.StringValue(api.Uuid)

	return
}

func callRuleProxyUpdateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.Uuid.IsNull() || data.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	if !data.Organization.IsNull() {
		org = data.Organization.ValueString()
	}

	req := *openapi.NewRuleProxyRequestWithDefaults()
    req.SetName(data.Name.ValueString())
    req.SetDomain(data.Domain.ValueString())
    // @todo: this should be a list of URLs
    var urls []string
    data.Url.ElementsAs(ctx, urls, false)
	req.SetUrl(urls)

	req.SetCountry(data.Country.ValueString())
	var countryList []string

	if data.Country.ValueString() == "country_is" {
	data.CountryIs.ElementsAs(ctx, countryList, false)
	req.SetCountryIs(countryList)
	} else if data.Country.ValueString() == "country_is_not" {
	data.CountryIsNot.ElementsAs(ctx, countryList, false)
	req.SetCountryIsNot(countryList)
	}

	req.SetIp(data.Ip.ValueString())
	var iplist []string

	if data.Ip.ValueString() == "ip_is" {
	data.IpIs.ElementsAs(ctx, iplist, false)
	req.SetIpIs(iplist)
	} else if data.Ip.ValueString() == "ip_is_not" {
	data.IpIsNot.ElementsAs(ctx, iplist, false)
	req.SetIpIsNot(iplist)
	}

	req.SetMethod(data.Method.ValueString())
	var methodList []string

	if data.Method.ValueString() == "method_is" {
	data.MethodIs.ElementsAs(ctx, methodList, false)
	req.SetMethodIs(methodList)
	} else if data.Method.ValueString() == "method_is_not" {
	data.MethodIsNot.ElementsAs(ctx, methodList, false)
	req.SetMethodIsNot(methodList)
	}

	// The proxy location.
	req.SetTo(data.To.ValueString())
	req.SetHost(data.Host.ValueString())

	// @todo: this should probably accept a string?
	    cl := strconv.FormatInt(data.CacheLifetime.ValueInt64(), 10)
	req.SetCacheLifetime(cl)

	if data.AuthPass.ValueString() != "" && data.AuthUser.ValueString() != "" {
	// Only set basic auth details if we have both.
	req.SetAuthUser(data.AuthUser.ValueString())
	req.SetAuthPass(data.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	req.SetFailoverMode(data.FailoverMode.ValueString())

	// Set strip headers.
	var stripHeaders []string
	data.ProxyStripHeaders.ElementsAs(ctx, stripHeaders, false)
	req.SetProxyStripHeaders(stripHeaders)

	req.SetWafEnabled(data.WafEnabled.ValueBool())

	// Build WAF config.
	wafConfig := *openapi.NewWAFConfigWithDefaults()
	wafConfig.SetMode(data.WafConfig.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

	// Set WAF rules.
	var allowRules []string
	data.WafConfig.AllowRules.ElementsAs(ctx, allowRules, false)
	wafConfig.SetAllowRules(allowRules)

	var allowIp []string
	data.WafConfig.AllowIp.ElementsAs(ctx, allowIp, false)
	wafConfig.SetAllowIp(allowIp)

	var blockIp []string
	data.WafConfig.BlockIp.ElementsAs(ctx, blockIp, false)
	wafConfig.SetBlockIp(blockIp)

	var blockUserAgent []string
	data.WafConfig.BlockUa.ElementsAs(ctx, blockUserAgent, false)
	wafConfig.SetBlockUa(blockUserAgent)

	var blockReferer []string
	data.WafConfig.BlockReferer.ElementsAs(ctx, blockReferer, false)
	wafConfig.SetBlockReferer(blockReferer)

	// httpbl dictionary support.
	// @todo support httpbl.

	notify := *openapi.NewNotifyConfigWithDefaults()
	notify.SetPeriod(data.NotifyConfig.Period.ValueString())
	notify.SetSlackWebhook(data.NotifyConfig.SlackWebhook.ValueString())

	var originStatusCodes []string
	data.NotifyConfig.OriginStatusCodes.ElementsAs(ctx, originStatusCodes, true)
	notify.SetOriginStatusCodes(originStatusCodes)

	req.SetNotifyConfig(notify)
	req.SetWafConfig(wafConfig)

	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyUpdate(r.client.AuthContext, data.Organization.ValueString(), data.Project.ValueString(), data.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to update rule proxy", err.Error())
		return
	}

	return
}

func callRuleProxyDeleteAPI(ctx context.Context, r *ruleProxyResource, rule *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule proxy", err.Error())
		return
	}

	return
}

func callRuleProxyReadAPI(ctx context.Context, r *ruleProxyResource, rule *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyRead(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to read rule proxy", err.Error())
		return
	}

	rule.Uuid = types.StringValue(api.Uuid)
    rule.Action = types.StringValue(api.Action)

    var actionConfig = resource_rule_proxy.ActionConfigValue{}
    actionConfig.AuthPass = types.StringValue(*api.ActionConfig.AuthPass)
    actionConfig.AuthUser = types.StringValue(*api.ActionConfig.AuthUser)
    actionConfig.CacheLifetime = types.StringValue(*api.ActionConfig.CacheLifetime)
    actionConfig.DisableSslVerify = types.BoolValue(*api.ActionConfig.DisableSslVerify)
    actionConfig.FailoverMode = types.BoolValue(*api.ActionConfig.FailoverMode)
    actionConfig.FailoverLifetime = types.StringValue(*api.ActionConfig.FailoverLifetime)
    failoverCodes, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.FailoverOriginStatusCodes)
    if d.HasError() {
    	diags.Append(d...)
     	return
    }
	actionConfig.FailoverOriginStatusCodes = failoverCodes
	actionConfig.FailoverOriginTtfb = types.StringValue(*api.ActionConfig.FailoverOriginTtfb)
	actionConfig.Host = types.StringValue(*api.ActionConfig.Host)
	actionConfig.Notify = types.StringValue(*api.ActionConfig.Notify)
	actionConfig.NotifyEmail = types.StringValue(*api.ActionConfig.NotifyEmail)
	actionConfig.OnlyProxy404 = types.BoolValue(*api.ActionConfig.OnlyProxy404)
	actionConfig.OriginTimeout = types.StringValue(*api.ActionConfig.OriginTimeout)
	actionConfig.To = types.StringValue(api.ActionConfig.To)

	var wafConfig = resource_rule_proxy.WafConfigValue{}
	allowRules, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.WafConfig.AllowRules)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.AllowRules = allowRules
	allowIp, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.WafConfig.AllowIp)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.AllowIp = allowIp
	blockIp, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.WafConfig.BlockIp)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.BlockIp = blockIp
	blockReferer, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.WafConfig.BlockReferer)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.BlockReferer = blockReferer
	blockUa, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.WafConfig.BlockUa)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.BlockUa = blockUa

	var httpbl = resource_rule_proxy.HttpblValue{}
	httpbl.ApiKey = types.StringValue(*api.ActionConfig.WafConfig.Httpbl.ApiKey)
	httpbl.BlockHarvester = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockHarvester)
	httpbl.BlockSearchEngine = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockSearchEngine)
	httpbl.BlockSpam = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockSpam)
	httpbl.BlockSuspicious = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockSuspicious)
	httpbl.HttpblEnabled = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.HttpblEnabled)
	httpVal, d := httpbl.ToObjectValue(ctx)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.Httpbl = httpVal

	wafConfig.IpRatelimitCooldown = types.Int64Value(int64(*api.ActionConfig.WafConfig.IpRatelimitCooldown))
	wafConfig.IpRatelimitMode = types.StringValue(*api.ActionConfig.WafConfig.IpRatelimitMode)
	wafConfig.IpRatelimitRps = types.Int64Value(int64(*api.ActionConfig.WafConfig.IpRatelimitRps))
	wafConfig.Mode = types.StringValue(api.ActionConfig.WafConfig.Mode)
	notifyEmail, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.WafConfig.NotifyEmail)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	wafConfig.NotifyEmail = notifyEmail
	wafConfig.NotifySlack = types.StringValue(*api.ActionConfig.WafConfig.NotifySlack)
	wafConfig.NotifySlackHitsRpm = types.Int64Value(int64(*api.ActionConfig.WafConfig.NotifySlackHitsRpm))
	wafConfig.NotifySlackRpm = types.Int64Value(int64(*api.ActionConfig.WafConfig.NotifySlackRpm))
	wafConfig.ParanoiaLevel = types.Int64Value(int64(*api.ActionConfig.WafConfig.ParanoiaLevel))
    wafConfig.RequestHeaderName = types.StringValue(*api.ActionConfig.WafConfig.RequestHeaderName)
    wafConfig.RequestHeaderRatelimitCooldown = types.Int64Value(int64(*api.ActionConfig.WafConfig.RequestHeaderRatelimitCooldown))
    wafConfig.RequestHeaderRatelimitMode = types.StringValue(*api.ActionConfig.WafConfig.RequestHeaderRatelimitMode)
    wafConfig.RequestHeaderRatelimitRps = types.Int64Value(int64(*api.ActionConfig.WafConfig.RequestHeaderRatelimitRps))
    wafConfig.WafRatelimitCooldown = types.Int64Value(int64(*api.ActionConfig.WafConfig.WafRatelimitCooldown))
    wafConfig.WafRatelimitMode = types.StringValue(*api.ActionConfig.WafConfig.WafRatelimitMode)
    wafConfig.WafRatelimitRps = types.Int64Value(int64(*api.ActionConfig.WafConfig.WafRatelimitRps))

    cfg, d := wafConfig.ToObjectValue(ctx)
    if d.HasError() {
    	diags.Append(d...)
     	return
    }
    actionConfig.WafConfig = cfg
    actionConfig.WafEnabled = types.BoolValue(api.ActionConfig.WafEnabled)

    rule.ActionConfig = actionConfig

	return
}
