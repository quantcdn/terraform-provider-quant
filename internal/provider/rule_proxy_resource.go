package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

type ruleProxyResourceModel struct {
	AuthPass                  types.String       `tfsdk:"auth_pass"`
	AuthUser                  types.String       `tfsdk:"auth_user"`
	CacheLifetime             types.Int64        `tfsdk:"cache_lifetime"`
	CookieName                types.String       `tfsdk:"cookie_name"`
	Country                   types.String       `tfsdk:"country"`
	CountryIs                 types.List         `tfsdk:"country_is"`
	CountryIsNot              types.List         `tfsdk:"country_is_not"`
	DisableSslVerify          types.Bool         `tfsdk:"disable_ssl_verify"`
	Disabled                  types.Bool         `tfsdk:"disabled"`
	Domain                    types.List         `tfsdk:"domain"`
	FailoverLifetime          types.String       `tfsdk:"failover_lifetime"`
	FailoverMode              types.Bool         `tfsdk:"failover_mode"`
	FailoverOriginStatusCodes types.List         `tfsdk:"failover_origin_status_codes"`
	FailoverOriginTtfb        types.String       `tfsdk:"failover_origin_ttfb"`
	Host                      types.String       `tfsdk:"host"`
	InjectHeaders             types.Map          `tfsdk:"inject_headers"`
	Ip                        types.String       `tfsdk:"ip"`
	IpIs                      types.List         `tfsdk:"ip_is"`
	IpIsNot                   types.List         `tfsdk:"ip_is_not"`
	Method                    types.String       `tfsdk:"method"`
	MethodIs                  types.List         `tfsdk:"method_is"`
	MethodIsNot               types.List         `tfsdk:"method_is_not"`
	Name                      types.String       `tfsdk:"name"`
	Notify                    types.String       `tfsdk:"notify"`
	NotifyConfig              *NotifyConfigValue `tfsdk:"notify_config"`
	OnlyProxy404              types.Bool         `tfsdk:"only_proxy_404"`
	OnlyWithCookie            types.Bool         `tfsdk:"only_with_cookie"`
	Organization              types.String       `tfsdk:"organization"`
	Project                   types.String       `tfsdk:"project"`
	ProxyStripHeaders         types.List         `tfsdk:"proxy_strip_headers"`
	ProxyStripRequestHeaders  types.List         `tfsdk:"proxy_strip_request_headers"`
	Rule                      types.String       `tfsdk:"rule"`
	RuleId                    types.String       `tfsdk:"rule_id"`
	To                        types.String       `tfsdk:"to"`
	Url                       types.List         `tfsdk:"url"`
	Uuid                      types.String       `tfsdk:"uuid"`
	WafConfig                 *WafConfigValue    `tfsdk:"waf_config"`
	WafEnabled                types.Bool         `tfsdk:"waf_enabled"`
}

type NotifyConfigValue struct {
	OriginStatusCodes []types.String `tfsdk:"origin_status_codes"`
	Period            types.String   `tfsdk:"period"`
	SlackWebhook      types.String   `tfsdk:"slack_webhook"`
}
type WafConfigValue struct {
	AllowIp                        types.List      `tfsdk:"allow_ip"`
	AllowRules                     types.List      `tfsdk:"allow_rules"`
	BlockIp                        types.List      `tfsdk:"block_ip"`
	BlockLists                     BlockListsValue `tfsdk:"block_lists"`
	BlockReferer                   types.List      `tfsdk:"block_referer"`
	BlockUa                        types.List      `tfsdk:"block_ua"`
	Httpbl                         HttpblValue     `tfsdk:"httpbl"`
	IpRatelimitCooldown            types.Int64     `tfsdk:"ip_ratelimit_cooldown"`
	IpRatelimitMode                types.String    `tfsdk:"ip_ratelimit_mode"`
	IpRatelimitRps                 types.Int64     `tfsdk:"ip_ratelimit_rps"`
	Mode                           types.String    `tfsdk:"mode"`
	NotifyEmail                    types.List      `tfsdk:"notify_email"`
	NotifySlack                    types.String    `tfsdk:"notify_slack"`
	NotifySlackHitsRpm             types.Int64     `tfsdk:"notify_slack_hits_rpm"`
	NotifySlackRpm                 types.Int64     `tfsdk:"notify_slack_rpm"`
	ParanoiaLevel                  types.Int64     `tfsdk:"paranoia_level"`
	RequestHeaderName              types.String    `tfsdk:"request_header_name"`
	RequestHeaderRatelimitCooldown types.Int64     `tfsdk:"request_header_ratelimit_cooldown"`
	RequestHeaderRatelimitMode     types.String    `tfsdk:"request_header_ratelimit_mode"`
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
	Enabled           types.Bool   `tfsdk:"enabled"`
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
			"rule_id": schema.StringAttribute{
				Computed: true,
			},
			"auth_pass": schema.StringAttribute{
				Optional: true,
			},
			"auth_user": schema.StringAttribute{
				Optional: true,
			},
			"cache_lifetime": schema.Int64Attribute{
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
			"domain": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"failover_lifetime": schema.StringAttribute{
				Optional: true,
			},
			"failover_mode": schema.BoolAttribute{
				Optional: true,
			},
			"failover_origin_status_codes": schema.ListAttribute{
				ElementType: types.StringType,
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

	if resp.Diagnostics.HasError() {
		return
	}

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

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleProxyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state ruleProxyResourceModel
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

func callRuleProxyCreateAPI(ctx context.Context, r *ruleProxyResource, data *ruleProxyResourceModel) (diags diag.Diagnostics) {
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
	req.SetTo(data.To.ValueString())
	req.SetHost(data.Host.ValueString())
	req.SetCacheLifetime(int32(data.CacheLifetime.ValueInt64()))

	if data.AuthPass.ValueString() != "" && data.AuthUser.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	if data.FailoverMode.ValueBool() == true {
		req.SetFailoverMode("true")
	} else {
		req.SetFailoverMode("false")
	}

	// Set strip headers.
	var stripHeaders []string
	for _, header := range data.ProxyStripHeaders.Elements() {
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
	httpbl := openapi.NewWAFConfigHttpblWithDefaults()
	httpbl.SetApiKey(data.WafConfig.Httpbl.ApiKey.ValueString())
	httpbl.SetBlockHarvester(data.WafConfig.Httpbl.BlockHarvester.ValueBool())
	httpbl.SetBlockSearchEngine(data.WafConfig.Httpbl.BlockSearchEngine.ValueBool())
	httpbl.SetBlockSpam(data.WafConfig.Httpbl.BlockSpam.ValueBool())
	httpbl.SetBlockSuspicious(data.WafConfig.Httpbl.BlockSuspicious.ValueBool())
	httpbl.SetHttpblEnabled(data.WafConfig.Httpbl.Enabled.ValueBool())
	wafConfig.SetHttpbl(*httpbl)

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

func callRuleProxyUpdateAPI(ctx context.Context, r *ruleProxyResource, data *ruleProxyResourceModel) (diags diag.Diagnostics) {
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
	req.SetTo(data.To.ValueString())
	req.SetHost(data.Host.ValueString())
	req.SetCacheLifetime(int32(data.CacheLifetime.ValueInt64()))

	if data.AuthPass.ValueString() != "" && data.AuthUser.ValueString() != "" {
		// Only set basic auth details if we have both.
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	if data.FailoverMode.ValueBool() == true {
		req.SetFailoverMode("true")
	} else {
		req.SetFailoverMode("false")
	}

	// Set strip headers.
	var stripHeaders []string
	for _, header := range data.ProxyStripHeaders.Elements() {
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
	httpbl := openapi.NewWAFConfigUpdateHttpblWithDefaults()
	httpbl.SetApiKey(data.WafConfig.Httpbl.ApiKey.ValueString())
	httpbl.SetBlockHarvester(data.WafConfig.Httpbl.BlockHarvester.ValueBool())
	httpbl.SetBlockSearchEngine(data.WafConfig.Httpbl.BlockSearchEngine.ValueBool())
	httpbl.SetBlockSpam(data.WafConfig.Httpbl.BlockSpam.ValueBool())
	httpbl.SetBlockSuspicious(data.WafConfig.Httpbl.BlockSuspicious.ValueBool())
	httpbl.SetHttpblEnabled(data.WafConfig.Httpbl.Enabled.ValueBool())
	wafConfig.SetHttpbl(*httpbl)

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

func callRuleProxyDeleteAPI(ctx context.Context, r *ruleProxyResource, rule *ruleProxyResourceModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule proxy", err.Error())
		return
	}

	return
}

func callRuleProxyReadAPI(ctx context.Context, r *ruleProxyResource, rule *ruleProxyResourceModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyRead(r.client.AuthContext, org, rule.Project.ValueString(), rule.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddWarning("Details: ", fmt.Sprintf("Org: %v, Project: %v, Uuid: %v", org, rule.Project.ValueString(), rule.RuleId.ValueString()))
		diags.AddError("Failed to read rule", err.Error())
		return
	}

	rule.Name = types.StringValue(api.GetName())
	rule.Uuid = types.StringValue(api.GetUuid())
	domains, d := types.ListValueFrom(ctx, types.StringType, api.GetDomain())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.Domain = domains
	urls, d := types.ListValueFrom(ctx, types.StringType, api.GetUrl())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.Url = urls
	rule.Ip = types.StringValue(api.GetIp())
	ips, d := types.ListValueFrom(ctx, types.StringType, api.GetIpIs())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.IpIs = types.List(ips)
	ipIsNot, d := types.ListValueFrom(ctx, types.StringType, api.GetIpIsNot())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.IpIsNot = types.List(ipIsNot)
	rule.Country = types.StringValue(api.GetCountry())
	countries, d := types.ListValueFrom(ctx, types.StringType, api.GetCountryIs())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.CountryIs = types.List(countries)
	rule.CountryIsNot, d = types.ListValueFrom(ctx, types.StringType, api.GetCountryIsNot())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.CountryIsNot = types.List(rule.CountryIsNot)

	rule.Method = types.StringValue(api.GetMethod())
	methods, d := types.ListValueFrom(ctx, types.StringType, api.GetMethodIs())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.MethodIs = types.List(methods)
	methodIsNot, d := types.ListValueFrom(ctx, types.StringType, api.GetMethodIsNot())
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.MethodIsNot = types.List(methodIsNot)

	// Rule specific fields.
	actionConfig, ok := api.GetActionConfigOk()

	if ok {
		rule.AuthPass = types.StringValue(actionConfig.GetAuthPass())
		rule.AuthUser = types.StringValue(actionConfig.GetAuthUser())
		if actionConfig.CacheLifetime != nil {
			rule.CacheLifetime = types.Int64Value(int64(actionConfig.GetCacheLifetime()))
		}
		rule.DisableSslVerify = types.BoolValue(actionConfig.GetDisableSslVerify())
		rule.FailoverMode = types.BoolValue(actionConfig.GetFailoverMode())
		if actionConfig.FailoverLifetime != nil {
			rule.FailoverLifetime = types.StringValue(actionConfig.GetFailoverLifetime())
		}
		failoverCodes, d := types.ListValueFrom(ctx, types.StringType, actionConfig.GetFailoverOriginStatusCodes())
		if d.HasError() {
			diags.Append(d...)
			return
		}
		rule.FailoverOriginStatusCodes = failoverCodes
		rule.FailoverOriginTtfb = types.StringValue(actionConfig.GetFailoverOriginTtfb())
		rule.Host = types.StringValue(actionConfig.GetHost())
		rule.Notify = types.StringValue(actionConfig.GetNotify())
		rule.OnlyProxy404 = types.BoolValue(actionConfig.GetOnlyProxy404())
		rule.To = types.StringValue(actionConfig.GetTo())

		proxyStripHeaders, d := types.ListValueFrom(ctx, types.StringType, actionConfig.GetProxyStripHeaders())
		rule.ProxyStripHeaders = proxyStripHeaders
		injectHeaders, d := types.MapValueFrom(ctx, types.StringType, actionConfig.GetInjectHeaders())
		rule.InjectHeaders = injectHeaders
		proxyStripRequestHeaders, d := types.ListValueFrom(ctx, types.StringType, actionConfig.GetProxyStripRequestHeaders())
		rule.ProxyStripRequestHeaders = proxyStripRequestHeaders

		// WafConfig specific read.
		wafconfig, ok := actionConfig.GetWafConfigOk()
		if !ok {
			diags.AddError("Failed to read WafConfig", "WafConfig is missing")
			return
		}
		if ok {
			if len(wafconfig.AllowRules) > 0 {
				allowRules, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetAllowRules())
				rule.WafConfig.AllowRules = allowRules
			} else {
				rule.WafConfig.AllowRules = types.ListNull(types.StringType)
			}
			if len(wafconfig.AllowIp) > 0 {
				allowIp, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetAllowIp())
				rule.WafConfig.AllowIp = allowIp
			} else {
				rule.WafConfig.AllowIp = types.ListNull(types.StringType)
			}
			if len(wafconfig.BlockIp) > 0 {
				blockIp, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetBlockIp())
				rule.WafConfig.BlockIp = blockIp
			} else {
				rule.WafConfig.BlockIp = types.ListNull(types.StringType)
			}
			if len(wafconfig.BlockReferer) > 0 {
				blockReferer, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetBlockReferer())
				rule.WafConfig.BlockReferer = blockReferer
			} else {
				rule.WafConfig.BlockReferer = types.ListNull(types.StringType)
			}
			if len(wafconfig.BlockUa) > 0 {
				blockUa, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetBlockUa())
				rule.WafConfig.BlockUa = blockUa
			} else {
				rule.WafConfig.BlockUa = types.ListNull(types.StringType)
			}

			httpbl, ok := wafconfig.GetHttpblOk()
			if ok {
				rule.WafConfig.Httpbl.ApiKey = types.StringValue(httpbl.GetApiKey())
				rule.WafConfig.Httpbl.BlockHarvester = types.BoolValue(httpbl.GetBlockHarvester())
				rule.WafConfig.Httpbl.BlockSearchEngine = types.BoolValue(httpbl.GetBlockSearchEngine())
				rule.WafConfig.Httpbl.BlockSpam = types.BoolValue(httpbl.GetBlockSpam())
				rule.WafConfig.Httpbl.BlockSuspicious = types.BoolValue(httpbl.GetBlockSuspicious())
				rule.WafConfig.Httpbl.Enabled = types.BoolValue(httpbl.GetHttpblEnabled())
			} else {
				rule.WafConfig.Httpbl.Enabled = types.BoolValue(false)
				rule.WafConfig.Httpbl.ApiKey = types.StringValue("")
				rule.WafConfig.Httpbl.BlockHarvester = types.BoolValue(false)
				rule.WafConfig.Httpbl.BlockSearchEngine = types.BoolValue(false)
				rule.WafConfig.Httpbl.BlockSpam = types.BoolValue(false)
				rule.WafConfig.Httpbl.BlockSuspicious = types.BoolValue(false)
			}
			rule.WafConfig.IpRatelimitMode = types.StringValue(wafconfig.GetIpRatelimitMode())
			if wafconfig.IpRatelimitCooldown != nil {
				rule.WafConfig.IpRatelimitCooldown = types.Int64Value(int64(wafconfig.GetIpRatelimitCooldown()))
			}
			if wafconfig.IpRatelimitRps != nil {
				rule.WafConfig.IpRatelimitRps = types.Int64Value(int64(wafconfig.GetIpRatelimitRps()))
			}
			rule.WafConfig.Mode = types.StringValue(wafconfig.GetMode())

			// NotifyEmail is a list of strings, so we need to convert it.
			if len(wafconfig.NotifyEmail) > 0 {
				notifyEmail, _ := types.ListValueFrom(ctx, types.StringType, wafconfig.GetNotifyEmail())
				rule.WafConfig.NotifyEmail = notifyEmail
			} else {
				rule.WafConfig.NotifyEmail = types.ListNull(types.StringType)
			}

			rule.WafConfig.NotifySlack = types.StringValue(wafconfig.GetNotifySlack())
			if wafconfig.NotifySlackHitsRpm != nil {
				rule.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(wafconfig.GetNotifySlackHitsRpm()))
			}
			if wafconfig.NotifySlackRpm != nil {
				rule.WafConfig.NotifySlackRpm = types.Int64Value(int64(wafconfig.GetNotifySlackRpm()))
			}
			if wafconfig.ParanoiaLevel != nil {
				rule.WafConfig.ParanoiaLevel = types.Int64Value(int64(wafconfig.GetParanoiaLevel()))
			}

			if wafconfig.RequestHeaderRatelimitCooldown != nil {
				rule.WafConfig.RequestHeaderRatelimitCooldown = types.Int64Value(int64(wafconfig.GetRequestHeaderRatelimitCooldown()))
			}
			if wafconfig.RequestHeaderRatelimitRps != nil {
				rule.WafConfig.RequestHeaderRatelimitRps = types.Int64Value(int64(wafconfig.GetRequestHeaderRatelimitRps()))
			}
			if wafconfig.WafRatelimitCooldown != nil {
				rule.WafConfig.WafRatelimitCooldown = types.Int64Value(int64(wafconfig.GetWafRatelimitCooldown()))
			}
			if wafconfig.WafRatelimitRps != nil {
				rule.WafConfig.WafRatelimitRps = types.Int64Value(int64(wafconfig.GetWafRatelimitRps()))
			}
			rule.WafConfig.RequestHeaderName = types.StringValue(wafconfig.GetRequestHeaderName())
			rule.WafConfig.RequestHeaderRatelimitMode = types.StringValue(wafconfig.GetRequestHeaderRatelimitMode())
			rule.WafConfig.WafRatelimitMode = types.StringValue(wafconfig.GetWafRatelimitMode())
		}
	}
	return
}
