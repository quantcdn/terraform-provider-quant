package provider

import (
	"context"
	"strings"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

type ruleProxyResource struct {
	client *client.Client
}

type ruleProxyResourceModel struct {
	AuthPass                  types.String      `tfsdk:"auth_pass"`
	AuthUser                  types.String      `tfsdk:"auth_user"`
	CacheLifetime             types.String      `tfsdk:"cache_lifetime"`
	CookieName                types.String      `tfsdk:"cookie_name"`
	Country                   types.String      `tfsdk:"country"`
	CountryIs                 types.List        `tfsdk:"country_is"`
	CountryIsNot              types.List        `tfsdk:"country_is_not"`
	DisableSslVerify          types.String       `tfsdk:"disable_ssl_verify"`
	Disabled                  types.Bool        `tfsdk:"disabled"`
	Domain                    types.String      `tfsdk:"domain"`
	FailoverLifetime          types.String      `tfsdk:"failover_lifetime"`
	FailoverMode              types.String      `tfsdk:"failover_mode"`
	FailoverOriginStatusCodes types.List        `tfsdk:"failover_origin_status_codes"`
	FailoverOriginTtfb        types.String      `tfsdk:"failover_origin_ttfb"`
	Host                      types.String      `tfsdk:"host"`
	InjectHeaders             types.Map         `tfsdk:"inject_headers"`
	Ip                        types.String      `tfsdk:"ip"`
	IpIs                      types.List        `tfsdk:"ip_is"`
	IpIsNot                   types.List        `tfsdk:"ip_is_not"`
	Method                    types.String      `tfsdk:"method"`
	MethodIs                  types.List        `tfsdk:"method_is"`
	MethodIsNot               types.List        `tfsdk:"method_is_not"`
	Name                      types.String      `tfsdk:"name"`
	Notify                    types.String      `tfsdk:"notify"`
	NotifyConfig              NotifyConfigValue `tfsdk:"notify_config"`
	OnlyProxy404              types.Bool        `tfsdk:"only_proxy_404"`
	OnlyWithCookie            types.Bool        `tfsdk:"only_with_cookie"`
	Organization              types.String      `tfsdk:"organization"`
	Project                   types.String      `tfsdk:"project"`
	ProxyStripHeaders         types.List        `tfsdk:"proxy_strip_headers"`
	ProxyStripRequestHeaders  types.List        `tfsdk:"proxy_strip_request_headers"`
	Rule                      types.String      `tfsdk:"rule"`
	To                        types.String      `tfsdk:"to"`
	Url                       types.List        `tfsdk:"url"`
	Uuid                      types.String      `tfsdk:"uuid"`
	WafConfig                 WafConfigValue    `tfsdk:"waf_config"`
	WafEnabled                types.Bool        `tfsdk:"waf_enabled"`
}

type NotifyConfigValue struct {
	OriginStatusCodes []types.String `tfsdk:"origin_status_codes"`
	Period            types.String   `tfsdk:"period"`
	SlackWebhook      types.String   `tfsdk:"slack_webhook"`
}
type WafConfigValue struct {
	AllowIp                        []types.String  `tfsdk:"allow_ip"`
	AllowRules                     []types.String  `tfsdk:"allow_rules"`
	BlockIp                        []types.String  `tfsdk:"block_ip"`
	BlockLists                     BlockListsValue `tfsdk:"block_lists"`
	BlockReferer                   []types.String  `tfsdk:"block_referer"`
	BlockUa                        []types.String  `tfsdk:"block_ua"`
	Httpbl                         HttpblValue     `tfsdk:"httpbl"`
	IpRatelimitCooldown            types.Int64     `tfsdk:"ip_ratelimit_cooldown"`
	IpRatelimitMode                types.String    `tfsdk:"ip_ratelimit_mode"`
	IpRatelimitRps                 types.Int64     `tfsdk:"ip_ratelimit_rps"`
	Mode                           types.String    `tfsdk:"mode"`
	NotifyEmail                    []types.String  `tfsdk:"notify_email"`
	NotifySlack                    types.String    `tfsdk:"notify_slack"`
	NotifySlackHitsRpm             types.Int64     `tfsdk:"notify_slack_hits_rpm"`
	NotifySlackRpm                 types.Int64     `tfsdk:"notify_slack_rpm"`
	ParanoiaLevel                  types.Int64     `tfsdk:"paranoia_level"`
	RequestHeaderName              types.String    `tfsdk:"request_header_name"`
	RequestHeaderRatelimitCooldown types.Int64     `tfsdk:"request_header_ratelimit_cooldown"`
	RequestHeaderRatelimitMode     types.String    `tfsdk:"request_header_ratelimit_mode`
	RequestHeaderRatelimitRps      types.Int64     `tfsdk:"request_header_ratelimit_rps"`
	WafRatelimitCooldown           types.Int64     `tfsdk:"waf_ratelimit_cooldown"`
	WafRatelimitHits               types.Int64     `tfsdk:"waf_ratelimit_hits"`
	WafRatelimitMode               types.String    `tfsdk:"waf_ratelimit_mode"`
	WafRatelimitRps                types.Int64     `tfsdk:"waf_ratelimit_rps"`
}

type BlockListsValue struct {
	Referer   types.Bool `tfsdk:"referer"`
	Ip        types.Bool `tfsdk:"ip"`
	UserAgent types.Bool `tfsdk:"user_agent"`
	Ai        types.Bool `tfsdk:"ai"`
}

type HttpblValue struct {
	Enabled           types.Bool   `tfsdk:"httpbl_enabled"`
	ApiKey            types.String `tfsdk:"api_key"`
	BlockSuspicious   types.Bool   `tfsdk:"block_suspicious"`
	BlockHarvester    types.Bool   `tfsdk:"block_harvester"`
	BlockSpam         types.Bool   `tfsdk:"block_spam"`
	BlockSearchEngine types.Bool   `tfsdk:"block_search_engine"`
}

func (r *ruleProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_proxy"
}

func (r *ruleProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed: true,
			},
			"auth_pass": schema.StringAttribute{
				Optional: true,
			},
			"auth_user": schema.StringAttribute{
				Optional: true,
			},
			"cache_lifetime": schema.StringAttribute{
				Optional: true,
			},
			"cookie_name": schema.StringAttribute{
				Optional: true,
			},
			"country": schema.StringAttribute{
				Optional: true,
			},
			"country_is": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"country_is_not": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"disable_ssl_verify": schema.BoolAttribute{
				Optional: true,
			},
			"disabled": schema.BoolAttribute{
				Optional: true,
			},
			"domain": schema.StringAttribute{
				Optional: true,
			},
			"failover_lifetime": schema.StringAttribute{
				Optional: true,
			},
			"failover_mode": schema.StringAttribute{
				Optional: true,
			},
			"failover_origin_status_codes": schema.ListAttribute{
				ElementType: types.Int64Type,
				Optional:    true,
			},
			"failover_origin_ttfb": schema.StringAttribute{
				Optional: true,
			},
			"host": schema.StringAttribute{
				Optional: true,
			},
			"inject_headers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"ip": schema.StringAttribute{
				Optional: true,
			},
			"ip_is": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"ip_is_not": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"method": schema.StringAttribute{
				Optional: true,
			},
			"method_is": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"method_is_not": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"notify": schema.StringAttribute{
				Optional: true,
			},
			"notify_config": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"origin_status_codes": schema.ListAttribute{
						ElementType: types.Int64Type,
						Optional:    true,
					},
					"period": schema.StringAttribute{
						Optional: true,
					},
					"slack_webhook": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"only_proxy_404": schema.BoolAttribute{
				Optional: true,
			},
			"only_with_cookie": schema.BoolAttribute{
				Optional: true,
			},
			"organization": schema.StringAttribute{
				Optional: true,
			},
			"project": schema.StringAttribute{
				Optional: true,
			},
			"proxy_strip_headers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"proxy_strip_request_headers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"rule": schema.StringAttribute{
				Optional: true,
			},
			"to": schema.StringAttribute{
				Optional: true,
			},
			"url": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"waf_config": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"allow_ip": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"allow_rules": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"block_ip": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"block_lists": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"referer": schema.BoolAttribute{
								Optional: true,
							},
							"ip": schema.BoolAttribute{
								Optional: true,
							},
							"user_agent": schema.BoolAttribute{
								Optional: true,
							},
							"ai": schema.BoolAttribute{
								Optional: true,
							},
						},
					},
					"block_referer": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"block_ua": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"httpbl": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional: true,
							},
							"api_key": schema.StringAttribute{
								Optional: true,
							},
							"block_suspicious": schema.BoolAttribute{
								Optional: true,
							},
							"block_harvester": schema.BoolAttribute{
								Optional: true,
							},
							"block_spam": schema.BoolAttribute{
								Optional: true,
							},
							"block_search_engine": schema.BoolAttribute{
								Optional: true,
							},
						},
					},
					"ip_ratelimit_cooldown": schema.Int64Attribute{
						Optional: true,
					},
					"ip_ratelimit_mode": schema.StringAttribute{
						Optional: true,
					},
					"ip_ratelimit_rps": schema.Int64Attribute{
						Optional: true,
					},
					"mode": schema.StringAttribute{
						Optional: true,
					},
					"notify_email": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
					},
					"notify_slack": schema.StringAttribute{
						Optional: true,
					},
					"notify_slack_hits_rpm": schema.Int64Attribute{
						Optional: true,
					},
					"notify_slack_rpm": schema.Int64Attribute{
						Optional: true,
					},
					"paranoia_level": schema.Int64Attribute{
						Optional: true,
					},
					"request_header_name": schema.StringAttribute{
						Optional: true,
					},
					"request_header_ratelimit_cooldown": schema.Int64Attribute{
						Optional: true,
					},
					"request_header_ratelimit_mode": schema.StringAttribute{
						Optional: true,
					},
					"request_header_ratelimit_rps": schema.Int64Attribute{
						Optional: true,
					},
					"waf_ratelimit_cooldown": schema.Int64Attribute{
						Optional: true,
					},
					"waf_ratelimit_hits": schema.Int64Attribute{
						Optional: true,
					},
					"waf_ratelimit_mode": schema.StringAttribute{
						Optional: true,
					},
					"waf_ratelimit_rps": schema.Int64Attribute{
						Optional: true,
					},
				},
			},
			"waf_enabled": schema.BoolAttribute{
				Optional: true,
			},
		},
	}
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
	var data ruleProxyResourceModel

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
	var data ruleProxyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ruleProxyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	resp.Diagnostics.Append(callRuleProxyUpdateAPI(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ruleProxyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callRuleProxyDeleteAPI(ctx, r, &data)...)
}

func (r *ruleProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ruleProxyResourceModel

	parts := strings.Split(req.ID, "/")

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"The ID must follow the pattern project/uuid to import.",
		)
		return
	}

	data.Project = types.StringValue(parts[0])
	data.Uuid = types.StringValue(parts[1])

	// Read API call logic
	resp.Diagnostics.Append(callRuleProxyReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleProxyCreateAPI(ctx context.Context, r *ruleProxyResource, data *ruleProxyResourceModel) (diags diag.Diagnostics) {
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
	req.SetCacheLifetime(data.CacheLifetime.ValueString())

	if data.AuthPass.ValueString() != "" && data.AuthUser.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueString())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	req.SetFailoverMode(data.FailoverMode.ValueString())

	// Set strip headers.
	var stripHeaders []string
	data.ProxyStripHeaders.ElementsAs(ctx, stripHeaders, false)
	req.SetProxyStripHeaders(stripHeaders)

	req.SetWafEnabled(data.WafEnabled.ValueBool())

	req.WafConfig.SetMode(data.WafConfig.Mode.ValueString())
	req.WafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

	// Set WAF rules.
	var allowRules []string
	for _, v := range data.WafConfig.AllowIp {
		allowRules = append(allowRules, v.ValueString())
	}
	req.WafConfig.SetAllowRules(allowRules)

	var allowIp []string
	for _, v := range data.WafConfig.AllowIp {
		allowIp = append(allowIp, v.ValueString())
	}
	req.WafConfig.SetAllowIp(allowIp)

	var blockIp []string
	for _, v := range data.WafConfig.BlockIp {
		blockIp = append(blockIp, v.ValueString())
	}
	req.WafConfig.SetBlockIp(blockIp)

	var blockUserAgent []string
	for _, v := range data.WafConfig.BlockUa {
		blockUserAgent = append(blockUserAgent, v.ValueString())
	}
	req.WafConfig.SetBlockUa(blockUserAgent)

	var blockReferer []string
	for _, v := range data.WafConfig.BlockReferer {
		blockReferer = append(blockReferer, v.ValueString())
	}
	req.WafConfig.SetBlockReferer(blockReferer)

	// httpbl dictionary support.
	// @todo support httpbl.

	req.NotifyConfig.SetPeriod(data.NotifyConfig.Period.ValueString())
	req.NotifyConfig.SetSlackWebhook(data.NotifyConfig.SlackWebhook.ValueString())

	var originStatusCodes []string
	for _, v := range data.NotifyConfig.OriginStatusCodes {
		originStatusCodes = append(originStatusCodes, v.ValueString())
	}
	req.NotifyConfig.SetOriginStatusCodes(originStatusCodes)

	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyCreate(r.client.AuthContext, data.Organization.ValueString(), data.Project.ValueString()).RuleProxyRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to create rule proxy", err.Error())
		return
	}

	// API needs to return uuid.
	data.Uuid = types.StringValue(api.Uuid)

	return
}

func callRuleProxyUpdateAPI(ctx context.Context, r *ruleProxyResource, data *ruleProxyResourceModel) (diags diag.Diagnostics) {
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
	req.SetCacheLifetime(data.CacheLifetime.ValueString())

	if data.AuthPass.ValueString() != "" && data.AuthUser.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueString())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	req.SetFailoverMode(data.FailoverMode.ValueString())

	// Set strip headers.
	var stripHeaders []string
	data.ProxyStripHeaders.ElementsAs(ctx, stripHeaders, false)
	req.SetProxyStripHeaders(stripHeaders)

	req.SetWafEnabled(data.WafEnabled.ValueBool())

	req.WafConfig.SetMode(data.WafConfig.Mode.ValueString())
	req.WafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

	// Set WAF rules.
	var allowRules []string
	for _, v := range data.WafConfig.AllowIp {
		allowRules = append(allowRules, v.ValueString())
	}
	req.WafConfig.SetAllowRules(allowRules)

	var allowIp []string
	for _, v := range data.WafConfig.AllowIp {
		allowIp = append(allowIp, v.ValueString())
	}
	req.WafConfig.SetAllowIp(allowIp)

	var blockIp []string
	for _, v := range data.WafConfig.BlockIp {
		blockIp = append(blockIp, v.ValueString())
	}
	req.WafConfig.SetBlockIp(blockIp)

	var blockUserAgent []string
	for _, v := range data.WafConfig.BlockUa {
		blockUserAgent = append(blockUserAgent, v.ValueString())
	}
	req.WafConfig.SetBlockUa(blockUserAgent)

	var blockReferer []string
	for _, v := range data.WafConfig.BlockReferer {
		blockReferer = append(blockReferer, v.ValueString())
	}
	req.WafConfig.SetBlockReferer(blockReferer)

	// httpbl dictionary support.
	// @todo support httpbl.

	req.NotifyConfig.SetPeriod(data.NotifyConfig.Period.ValueString())
	req.NotifyConfig.SetSlackWebhook(data.NotifyConfig.SlackWebhook.ValueString())

	var originStatusCodes []string
	for _, v := range data.NotifyConfig.OriginStatusCodes {
		originStatusCodes = append(originStatusCodes, v.ValueString())
	}
	req.NotifyConfig.SetOriginStatusCodes(originStatusCodes)

	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyUpdate(r.client.AuthContext, org, data.Project.ValueString(), data.Uuid.ValueString()).RuleProxyRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule proxy", err.Error())
		return
	}

	return
}

func callRuleProxyDeleteAPI(ctx context.Context, r *ruleProxyResource, rule *ruleProxyResourceModel) (diags diag.Diagnostics) {
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

func callRuleProxyReadAPI(ctx context.Context, r *ruleProxyResource, rule *ruleProxyResourceModel) (diags diag.Diagnostics) {
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

	rule.AuthPass = types.StringValue(*api.ActionConfig.AuthPass)
	rule.AuthUser = types.StringValue(*api.ActionConfig.AuthUser)
	rule.CacheLifetime = types.StringValue(*api.ActionConfig.CacheLifetime)
	rule.DisableSslVerify = types.StringValue(*api.ActionConfig.DisableSslVerify)
	// rule.FailoverMode = types.StringValue(*api.ActionConfig.FailoverMode)
	rule.FailoverMode = types.StringValue("true")
	rule.FailoverLifetime = types.StringValue(*api.ActionConfig.FailoverLifetime)
	failoverCodes, d := types.ListValueFrom(ctx, types.StringType, api.ActionConfig.FailoverOriginStatusCodes)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.FailoverOriginStatusCodes = failoverCodes
	rule.FailoverOriginTtfb = types.StringValue(*api.ActionConfig.FailoverOriginTtfb)
	rule.Host = types.StringValue(*api.ActionConfig.Host)
	rule.Notify = types.StringValue(*api.ActionConfig.Notify)
	rule.OnlyProxy404 = types.BoolValue(*api.ActionConfig.OnlyProxy404)
	rule.To = types.StringValue(api.ActionConfig.To)

	var allowRules = []types.String{}
	for _, v := range(api.ActionConfig.WafConfig.AllowRules) {
		allowRules = append(allowRules, types.StringValue(v))
	}
	rule.WafConfig.AllowRules = allowRules
	var allowIp = []types.String{}
	for _, v := range(api.ActionConfig.WafConfig.AllowIp) {
		allowIp = append(allowIp, types.StringValue(v))
	}
	rule.WafConfig.AllowIp = allowIp
	var blockIp = []types.String{}
	for _, v := range(api.ActionConfig.WafConfig.BlockIp) {
		blockIp = append(blockIp, types.StringValue(v))
	}
	rule.WafConfig.BlockIp = blockIp
	var blockReferer = []types.String{}
	for _, v := range(api.ActionConfig.WafConfig.BlockReferer) {
		blockReferer = append(blockReferer, types.StringValue(v))
	}
	rule.WafConfig.BlockReferer = blockReferer
	var blockUa = []types.String{}
	for _, v := range(api.ActionConfig.WafConfig.BlockUa) {
		blockUa = append(blockUa, types.StringValue(v))
	}
	rule.WafConfig.BlockUa = blockUa
	rule.WafConfig.Httpbl.ApiKey = types.StringValue(*api.ActionConfig.WafConfig.Httpbl.ApiKey)
	rule.WafConfig.Httpbl.BlockHarvester = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockHarvester)
	rule.WafConfig.Httpbl.BlockSearchEngine = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockSearchEngine)
	rule.WafConfig.Httpbl.BlockSpam = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockSpam)
	rule.WafConfig.Httpbl.BlockSuspicious = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.BlockSuspicious)
	rule.WafConfig.Httpbl.Enabled = types.BoolValue(api.ActionConfig.WafConfig.Httpbl.HttpblEnabled)
	rule.WafConfig.IpRatelimitCooldown = types.Int64Value(int64(*api.ActionConfig.WafConfig.IpRatelimitCooldown))
	rule.WafConfig.IpRatelimitMode = types.StringValue(*api.ActionConfig.WafConfig.IpRatelimitMode)
	rule.WafConfig.IpRatelimitRps = types.Int64Value(int64(*api.ActionConfig.WafConfig.IpRatelimitRps))
	rule.WafConfig.Mode = types.StringValue(api.ActionConfig.WafConfig.Mode)
	var notifyEmail = []types.String{}
	for _, v := range(api.ActionConfig.WafConfig.NotifyEmail) {
		notifyEmail = append(notifyEmail, types.StringValue(v))
	}
	rule.WafConfig.NotifyEmail = notifyEmail
	rule.WafConfig.NotifySlack = types.StringValue(*api.ActionConfig.WafConfig.NotifySlack)
	rule.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(*api.ActionConfig.WafConfig.NotifySlackHitsRpm))
	rule.WafConfig.NotifySlackRpm = types.Int64Value(int64(*api.ActionConfig.WafConfig.NotifySlackRpm))
	rule.WafConfig.ParanoiaLevel = types.Int64Value(int64(*api.ActionConfig.WafConfig.ParanoiaLevel))
	rule.WafConfig.RequestHeaderName = types.StringValue(*api.ActionConfig.WafConfig.RequestHeaderName)
	rule.WafConfig.RequestHeaderRatelimitCooldown = types.Int64Value(int64(*api.ActionConfig.WafConfig.RequestHeaderRatelimitCooldown))
	rule.WafConfig.RequestHeaderRatelimitMode = types.StringValue(*api.ActionConfig.WafConfig.RequestHeaderRatelimitMode)
	rule.WafConfig.RequestHeaderRatelimitRps = types.Int64Value(int64(*api.ActionConfig.WafConfig.RequestHeaderRatelimitRps))
	rule.WafConfig.WafRatelimitCooldown = types.Int64Value(int64(*api.ActionConfig.WafConfig.WafRatelimitCooldown))
	rule.WafConfig.WafRatelimitMode = types.StringValue(*api.ActionConfig.WafConfig.WafRatelimitMode)
	rule.WafConfig.WafRatelimitRps = types.Int64Value(int64(*api.ActionConfig.WafConfig.WafRatelimitRps))

	return
}
