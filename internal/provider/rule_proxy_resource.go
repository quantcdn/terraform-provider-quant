package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &ruleProxy{}
	_ resource.ResourceWithConfigure = &ruleProxy{}
)

// NewruleProxy is a helper function to simplify the provider implementation.
func NewRuleProxyResource() resource.Resource {
	return &ruleProxy{}
}

// ruleProxy is the resource implementation.
type ruleProxy struct {
	client *client.Client
}

type ruleProxyModel struct {
	Name     types.String `tfsdk:"name"`
	Uuid     types.String `tfsdk:"uuid"`
	Project  types.String `tfsdk:"project"`
	Disabled types.Bool   `tfsdk:"disabled"`

	Domain types.String `tfsdk:"domain"`

	// Rule selection.
	CountryInclude types.Bool     `tfsdk:"country_include"`
	Countries      []types.String `tfsdk:"countries"`
	MethodInclude  types.Bool     `tfsdk:"method_include"`
	Methods        []types.String `tfsdk:"methods"`
	IpInclude      types.Bool     `tfsdk:"ip_include"`
	Ips            []types.String `tfsdk:"ips"`
	OnlyWithCookie types.String   `tfsdk:"only_with_cookie"`

	// Rule behaviour.
	Url              types.String   `tfsdk:"url"`
	To               types.String   `tfsdk:"to"`
	Host             types.String   `tfsdk:"host"`
	AuthUser         types.String   `tfsdk:"auth_user"`
	AuthPass         types.String   `tfsdk:"auth_pass"`
	DisableSSLVerify types.String   `tfsdk:"disable_ssl_verify"`
	CacheLifetime    types.Int64    `tfsdk:"cache_lifetime"`
	Only404          types.Bool     `tfsdk:"only_404"`
	StripHeaders     []types.String `tfsdk:"strip_headers"`
	WafEnable        types.Bool     `tfsdk:"waf_enabled"`
	WafConfig        WafConfig      `tfsdk:"waf_config"`
}

// @todo: Move ot a separate location
type WafConfig struct {
	Mode               types.String   `tfsdk:"mode"`
	ParanoiaLevel      types.Int64    `tfsdk:"paranoia_level"`
	AllowRules         []types.Int64  `tfsdk:"allow_rules"`
	AllowIp            []types.String `tfsdk:"allow_ip"`
	BlockIp            []types.String `tfsdk:"block_ip"`
	BlockUa            []types.String `tfsdk:"block_ua"`
	BlockReferer       []types.String `tfsdk:"block_referer"`
	NotifySlack        types.String   `tfsdk:"notify_slack"`
	NotifySlackHitsRpm types.Int64    `tfsdk:"notify_slack_rpm"`
	Httpbl             struct {
		Enabled           types.Bool `tfsdk:"httpbl_enabled"`
		BlockSuspicious   types.Bool `tfsdk:"block_suspicious"`
		BlockHarvester    types.Bool `tfsdk:"block_harvester"`
		BlockSpam         types.Bool `tfsdk:"block_spam"`
		BlockSearchEgnine types.Bool `tfsdk:"block_search_engine"`
	} `tfsdk:"httbl"`
}

// Configure adds the provider configured client to the resource.
func (r *ruleProxy) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *hashicups.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *ruleProxy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_proxy"
}

// Schema defines the schema for the resource.
func (r *ruleProxy) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "A name for the rule",
				Optional:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The project machine name",
				Required:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "If this rule is disabled",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain the rule applies to",
				Optional:            true,
			},
			"countries": schema.ListAttribute{
				MarkdownDescription: "A list of countries",
				Optional:            true,
			},
			"country_include": schema.BoolAttribute{
				MarkdownDescription: "Include the country list",
				Optional:            true,
			},
			"methods": schema.ListAttribute{
				MarkdownDescription: "A list of HTTP methods",
				Optional:            true,
			},
			"method_include": schema.BoolAttribute{
				MarkdownDescription: "Include the methods",
				Optional:            true,
			},
			"ips": schema.ListAttribute{
				MarkdownDescription: "A list of IP addresses",
				Optional:            true,
			},
			"ip_include": schema.BoolAttribute{
				MarkdownDescription: "Include hte IP addresses",
				Optional:            true,
			},
			"only_with_cookie": schema.StringAttribute{
				MarkdownDescription: "Apply rule only if the cookie is present",
				Optional:            true,
			},

			// Rule behaviours
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL pattern to apply the rule to",
				Required:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("*"),
			},
			"to": schema.StringAttribute{
				MarkdownDescription: "The origin hostname to proxy to",
				Required:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "The host header to send to the origin server",
				Optional:            true,
			},
			"auth_user": schema.StringAttribute{
				MarkdownDescription: "HTTP authentication username to pass to the origin server",
				Optional:            true,
			},
			"auth_pass": schema.StringAttribute{
				MarkdownDescription: "HTTP authentication password to pass to the origin server",
				Optional:            true,
			},
			"disable_ssl_verify": schema.BoolAttribute{
				MarkdownDescription: "Disable TLS verification between Quant and the origin server",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"cache_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Override the cache TTL from the origin server",
				Optional:            true,
			},
			"only_404": schema.BoolAttribute{
				MarkdownDescription: "Only proxy requests if a 404 is detected from Quant's static archive",
				Optional:            true,
			},
			"strip_headers": schema.ListAttribute{
				MarkdownDescription: "Strip headers from the request to origin",
				Optional:            true,
			},
			"waf_enabled": schema.BoolAttribute{
				MarkdownDescription: "If the proxy should have the WAF enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"waf_config": schema.SingleNestedAttribute{
				MarkdownDescription: "WAF configuration for this rule",
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "The mode to run the WAF in",
						Required:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("report"),
					},
					"paranoia_level": schema.Int64Attribute{
						MarkdownDescription: "The paranoia level to run the WAF in",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(1),
					},
					"allow_rules": schema.ListAttribute{
						MarkdownDescription: "A list of rule ids to allow through the WAF",
						Optional:            true,
					},
					"allow_ip": schema.ListAttribute{
						MarkdownDescription: "A list of IP addresses that are excluded form the WAF",
						Optional:            true,
					},
					"block_ip": schema.ListAttribute{
						MarkdownDescription: "A list of IP addresses that are blocked",
						Optional:            true,
					},
					"block_ua": schema.ListAttribute{
						MarkdownDescription: "A list of user agents that are blocked",
						Optional:            true,
					},
					"block_referer": schema.ListAttribute{
						MarkdownDescription: "A list of referer host names that are blocked",
						Optional:            true,
					},
					"notify_slack": schema.StringAttribute{
						MarkdownDescription: "A slack webhook URL to notify",
						Optional:            true,
					},
					"notify_slack_rpm": schema.Int64Attribute{
						MarkdownDescription: "Throttle the notifications to slack",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(5),
					},
					"httpbl": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								MarkdownDescription: "Enable HTTPBL integration",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"block_suspicious": schema.BoolAttribute{
								MarkdownDescription: "Block suscpicious requests",
								Optional:            true,
							},
							"block_harvester": schema.BoolAttribute{
								MarkdownDescription: "Block known harvesters",
								Optional:            true,
							},
							"block_spam": schema.BoolAttribute{
								MarkdownDescription: "Block known spam actors",
								Optional:            true,
							},
							"block_search_engine": schema.BoolAttribute{
								MarkdownDescription: "Block search engines",
								Optional:            true,
							},
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ruleProxy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleProxyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CountryInclude.IsNull() && plan.IpInclude.IsNull() && plan.MethodInclude.IsNull() {
		resp.Diagnostics.AddError(
			"Rule criteria is missing",
			"Could not crete a rule due to missing criteria; must provide country, ip and/or method",
		)
	}

	proxy := openapi.NewRuleProxyRequest()

	proxy.SetName(plan.Name.ValueString())
	proxy.SetDisabled(plan.Disabled.ValueBool())
	proxy.SetDomain(plan.Domain.ValueString())

	if !plan.CountryInclude.IsNull() && !plan.CountryInclude.IsUnknown() {
		if plan.CountryInclude.ValueBool() {
			proxy.SetCountry("country_is")
			proxy.SetCountryIs(helpers.FlattenToStrings(plan.Countries))
		} else {
			proxy.SetCountry("country_is_not")
			proxy.SetCountryIsNot(helpers.FlattenToStrings(plan.Countries))
		}
	}

	if !plan.MethodInclude.IsNull() && !plan.MethodInclude.IsUnknown() {
		if plan.MethodInclude.ValueBool() {
			proxy.SetMethod("method_is")
			proxy.SetMethodIs(helpers.FlattenToStrings(plan.Methods))
		} else {
			proxy.SetMethod("method_is_not")
			proxy.SetMethodIsNot(helpers.FlattenToStrings(plan.Methods))
		}
	}

	if !plan.IpInclude.IsNull() && !plan.IpInclude.IsUnknown() {
		if plan.IpInclude.ValueBool() {
			proxy.SetIp("ip_is")
			proxy.SetIpIs(helpers.FlattenToStrings(plan.Ips))
		} else {
			proxy.SetIp("ip_is_not")
			proxy.SetIpIsNot(helpers.FlattenToStrings(plan.Ips))
		}
	}

	// Rule behaviour.
	proxy.SetUrl(plan.Url.ValueString())
	proxy.SetTo(plan.To.ValueString())
	proxy.SetHost(plan.Host.ValueString())
	proxy.SetAuthUser(plan.AuthUser.ValueString())
	proxy.SetAuthPass(plan.AuthPass.ValueString())
	// proxy.SetDisableSslVerify(plan.DisableSSLVerify.BoolValue())
	proxy.SetCacheLifetime(int32(plan.CacheLifetime.ValueInt64()))
	proxy.SetOnlyProxy404(plan.Only404.ValueBool())
	proxy.SetStripHeaders(helpers.FlattenToStrings(plan.StripHeaders))
	proxy.SetWafEnabled(plan.WafEnable.ValueBool())

	var wafConfig openapi.RuleProxyRequestWafConfig
	wafConfig.SetMode(plan.WafConfig.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(plan.WafConfig.ParanoiaLevel.ValueInt64()))

	// @todo: Update client — IPs should probably be strings.
	// wafConfig.SetAllowIp(helpers.FlattenToInt32(plan.WafConfig.AllowIp))
	// wafConfig.SetBlockIp(helpers.FlattenToInt32(plan.WafConfig.BlockIp))

	wafConfig.SetAllowRules(helpers.FlattenToInt32(plan.WafConfig.AllowRules))
	wafConfig.SetBlockReferer(helpers.FlattenToStrings(plan.WafConfig.BlockReferer))
	wafConfig.SetBlockUa(helpers.FlattenToStrings(plan.WafConfig.BlockUa))
	wafConfig.SetNotifySlack(plan.WafConfig.NotifySlack.ValueString())
	wafConfig.SetNotifySlackHitsRpm(int32(plan.WafConfig.NotifySlackHitsRpm.ValueInt64()))

	proxy.SetWafConfig(wafConfig)

	organization := r.client.Organization
	project := plan.Project.ValueString()

	client := r.client.Admin.RulesAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectRulesProxyPost(context.Background(), organization, project).RuleProxyRequest(*proxy).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
	}

	plan.Uuid = types.StringValue(*res.GetData().Rules[0].Uuid)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ruleProxy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleProxyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := r.client.Organization
	project := state.Project.ValueString()

	client := r.client.Admin.RulesAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectRuleGet(context.Background(), organization, project, state.Uuid.ValueString()).Execute()
	rule := res.Data.Rules[0]

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not update rule, unexpected error: "+err.Error(),
		)
	}

	state.Uuid = types.StringValue(*rule.Uuid)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ruleProxy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleProxyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	proxy := openapi.NewRuleProxyRequest()

	proxy.SetName(plan.Name.ValueString())
	proxy.SetDisabled(plan.Disabled.ValueBool())
	proxy.SetDomain(plan.Domain.ValueString())

	if !plan.CountryInclude.IsNull() && !plan.CountryInclude.IsUnknown() {
		if plan.CountryInclude.ValueBool() {
			proxy.SetCountry("country_is")
			proxy.SetCountryIs(helpers.FlattenToStrings(plan.Countries))
		} else {
			proxy.SetCountry("country_is_not")
			proxy.SetCountryIsNot(helpers.FlattenToStrings(plan.Countries))
		}
	}

	if !plan.MethodInclude.IsNull() && !plan.MethodInclude.IsUnknown() {
		if plan.MethodInclude.ValueBool() {
			proxy.SetMethod("method_is")
			proxy.SetMethodIs(helpers.FlattenToStrings(plan.Methods))
		} else {
			proxy.SetMethod("method_is_not")
			proxy.SetMethodIsNot(helpers.FlattenToStrings(plan.Methods))
		}
	}

	if !plan.IpInclude.IsNull() && !plan.IpInclude.IsUnknown() {
		if plan.IpInclude.ValueBool() {
			proxy.SetIp("ip_is")
			proxy.SetIpIs(helpers.FlattenToStrings(plan.Ips))
		} else {
			proxy.SetIp("ip_is_not")
			proxy.SetIpIsNot(helpers.FlattenToStrings(plan.Ips))
		}
	}

	// Rule behaviour.
	proxy.SetUrl(plan.Url.ValueString())
	proxy.SetTo(plan.To.ValueString())
	proxy.SetHost(plan.Host.ValueString())
	proxy.SetAuthUser(plan.AuthUser.ValueString())
	proxy.SetAuthPass(plan.AuthPass.ValueString())
	// proxy.SetDisableSslVerify(plan.DisableSSLVerify.BoolValue())
	proxy.SetCacheLifetime(int32(plan.CacheLifetime.ValueInt64()))
	proxy.SetOnlyProxy404(plan.Only404.ValueBool())
	proxy.SetStripHeaders(helpers.FlattenToStrings(plan.StripHeaders))
	proxy.SetWafEnabled(plan.WafEnable.ValueBool())

	var wafConfig openapi.RuleProxyRequestWafConfig
	wafConfig.SetMode(plan.WafConfig.Mode.ValueString())
	wafConfig.SetParanoiaLevel(int32(plan.WafConfig.ParanoiaLevel.ValueInt64()))

	// @todo: Update client — IPs should probably be strings.
	// wafConfig.SetAllowIp(helpers.FlattenToInt32(plan.WafConfig.AllowIp))
	// wafConfig.SetBlockIp(helpers.FlattenToInt32(plan.WafConfig.BlockIp))

	wafConfig.SetAllowRules(helpers.FlattenToInt32(plan.WafConfig.AllowRules))
	wafConfig.SetBlockReferer(helpers.FlattenToStrings(plan.WafConfig.BlockReferer))
	wafConfig.SetBlockUa(helpers.FlattenToStrings(plan.WafConfig.BlockUa))
	wafConfig.SetNotifySlack(plan.WafConfig.NotifySlack.ValueString())
	wafConfig.SetNotifySlackHitsRpm(int32(plan.WafConfig.NotifySlackHitsRpm.ValueInt64()))

	proxy.SetWafConfig(wafConfig)

	organization := r.client.Organization
	project := plan.Project.ValueString()

	client := r.client.Admin.RulesAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesProxyRulePatch(context.Background(), organization, project, plan.Uuid.ValueString()).RuleProxyRequest(*proxy).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not update rule, unexpected error: "+err.Error(),
		)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ruleProxy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleProxyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Project.IsNull() {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Invalid state: project machine name is unknown.",
		)
		return
	}

	organization := r.client.Organization
	project := state.Project.ValueString()

	client := r.client.Admin.RulesAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesProxyRuleDelete(r.client.Auth, organization, project, state.Uuid.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}
