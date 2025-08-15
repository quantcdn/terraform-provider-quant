package provider

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_proxy"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go"
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

// Helper functions for backwards compatibility with cache_lifetime field
// The field is now a string in the schema but the API still expects an integer

// parseCacheLifetime converts a string cache_lifetime value to int64 for API calls
// Supports backwards compatibility with existing integer configurations
func parseCacheLifetime(cacheLifetimeStr types.String) (int64, error) {
	if cacheLifetimeStr.IsNull() || cacheLifetimeStr.IsUnknown() {
		return 0, fmt.Errorf("cache_lifetime is null or unknown")
	}
	
	value := cacheLifetimeStr.ValueString()
	if value == "" {
		return 0, fmt.Errorf("cache_lifetime is empty")
	}
	
	// Parse as integer (handles both string representations of integers and actual integers)
	cacheLifetime, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid cache_lifetime value '%s': must be a valid integer", value)
	}
	
	return cacheLifetime, nil
}

// formatCacheLifetime converts a string cache_lifetime value to types.String for Terraform state
func formatCacheLifetime(cacheLifetime string) types.String {
	return types.StringValue(cacheLifetime)
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
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ruleProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // Serialise rule modifications to avoid backend JSON races
    if r.client != nil && r.client.RulesMutex != nil {
        r.client.RulesMutex.Lock()
        defer r.client.RulesMutex.Unlock()
    }
	var data resource_rule_proxy.RuleProxyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	diags := callRuleProxyCreateAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = callRuleProxyReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
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
	diags := callRuleProxyReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // Serialise rule modifications to avoid backend JSON races
    if r.client != nil && r.client.RulesMutex != nil {
        r.client.RulesMutex.Lock()
        defer r.client.RulesMutex.Unlock()
    }
	var plan resource_rule_proxy.RuleProxyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state resource_rule_proxy.RuleProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	// Update API call logic
	diags := callRuleProxyUpdateAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read updated state
	diags = callRuleProxyReadAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // Serialise rule modifications to avoid backend JSON races
    if r.client != nil && r.client.RulesMutex != nil {
        r.client.RulesMutex.Lock()
        defer r.client.RulesMutex.Unlock()
    }
	var data resource_rule_proxy.RuleProxyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	diags := callRuleProxyDeleteAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *ruleProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_proxy.RuleProxyModel
	var err error
	data.Project, data.RuleId, err = utils.GetRuleImportId(req.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse import ID. Error: %s", err.Error()),
		)
		return
	}

	// Read API call logic
	diags := callRuleProxyReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleProxyCreateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	req := *quantadmingo.NewRuleProxyRequestCreateWithDefaults()
	req.SetName(data.Name.ValueString())

	// Domain handling
	if !data.Domain.IsNull() {
		var domains []string
		if !data.Domain.IsUnknown() {
			diags.Append(data.Domain.ElementsAs(ctx, &domains, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetDomain(domains)
	}

	// URL handling
	if !data.Url.IsNull() {
		var urls []string
		if !data.Url.IsUnknown() {
			diags.Append(data.Url.ElementsAs(ctx, &urls, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetUrl(urls)
	}

	// Country handling
	if !data.Country.IsNull() {
		req.SetCountry(data.Country.ValueString())
		if data.Country.ValueString() == "country_is" && !data.CountryIs.IsNull() {
			var countryList []string
			if !data.CountryIs.IsUnknown() {
				diags.Append(data.CountryIs.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIs(countryList)
		} else if data.Country.ValueString() == "country_is_not" && !data.CountryIsNot.IsNull() {
			var countryList []string
			if !data.CountryIsNot.IsUnknown() {
				diags.Append(data.CountryIsNot.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIsNot(countryList)
		}
	}

	// IP handling
	if !data.Ip.IsNull() {
		req.SetIp(data.Ip.ValueString())
		if data.Ip.ValueString() == "ip_is" && !data.IpIs.IsNull() {
			var ipList []string
			if !data.IpIs.IsUnknown() {
				diags.Append(data.IpIs.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIs(ipList)
		} else if data.Ip.ValueString() == "ip_is_not" && !data.IpIsNot.IsNull() {
			var ipList []string
			if !data.IpIsNot.IsUnknown() {
				diags.Append(data.IpIsNot.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIsNot(ipList)
		}
	}

	// Method handling
	if !data.Method.IsNull() {
		req.SetMethod(data.Method.ValueString())
		if data.Method.ValueString() == "method_is" && !data.MethodIs.IsNull() {
			var methodList []string
			if !data.MethodIs.IsUnknown() {
				diags.Append(data.MethodIs.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIs(methodList)
		} else if data.Method.ValueString() == "method_is_not" && !data.MethodIsNot.IsNull() {
			var methodList []string
			if !data.MethodIsNot.IsUnknown() {
				diags.Append(data.MethodIsNot.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIsNot(methodList)
		}
	}

	// Proxy configuration
	// Only set `to` when application_proxy is not true; otherwise backend computes it
	appProxy := !data.ApplicationProxy.IsNull() && !data.ApplicationProxy.IsUnknown() && data.ApplicationProxy.ValueBool()
	if !appProxy {
		if !data.To.IsNull() && !data.To.IsUnknown() && data.To.ValueString() != "" {
			req.SetTo(data.To.ValueString())
		}
	}
	if !data.Host.IsNull() && !data.Host.IsUnknown() && data.Host.ValueString() != "" {
		req.SetHost(data.Host.ValueString())
	}
	// Application proxy configuration
	if !data.ApplicationProxy.IsNull() && !data.ApplicationProxy.IsUnknown() {
		req.SetApplicationProxy(data.ApplicationProxy.ValueBool())
	}
	if !data.ApplicationName.IsNull() && !data.ApplicationName.IsUnknown() {
		req.SetApplicationName(data.ApplicationName.ValueString())
	}
	if !data.ApplicationEnvironment.IsNull() && !data.ApplicationEnvironment.IsUnknown() {
		req.SetApplicationEnvironment(data.ApplicationEnvironment.ValueString())
	}
	if !data.ApplicationContainer.IsNull() && !data.ApplicationContainer.IsUnknown() {
		req.SetApplicationContainer(data.ApplicationContainer.ValueString())
	}
	if !data.ApplicationPort.IsNull() && !data.ApplicationPort.IsUnknown() {
		// API expects integer; cast to int32 which is typical for ports
		req.SetApplicationPort(int32(data.ApplicationPort.ValueInt64()))
	}
	if !data.CacheLifetime.IsNull() && !data.CacheLifetime.IsUnknown() {
		cacheLifetime, err := parseCacheLifetime(data.CacheLifetime)
		if err != nil {
			diags.AddError(
				"Invalid cache_lifetime value",
				fmt.Sprintf("Could not parse cache_lifetime: %s", err.Error()),
			)
			return
		}
		req.SetCacheLifetime(strconv.FormatInt(cacheLifetime, 10))
	}

	if !data.AuthUser.IsNull() && !data.AuthPass.IsNull() {
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	// Weight handling
	if !data.Weight.IsNull() && !data.Weight.IsUnknown() {
		weight := int32(data.Weight.ValueInt64())
		req.SetWeight(weight)
	}

	// Strip headers handling
	if !data.ProxyStripHeaders.IsNull() {
		var stripHeaders []string
		if !data.ProxyStripHeaders.IsUnknown() {
			diags.Append(data.ProxyStripHeaders.ElementsAs(ctx, &stripHeaders, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetProxyStripHeaders(stripHeaders)
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	// Proxy alert configuration
	if !data.ProxyAlertEnabled.IsNull() {
		req.SetProxyAlertEnabled(data.ProxyAlertEnabled.ValueBool())
	}

	// Failover configuration
	req.SetFailoverMode(data.FailoverMode.ValueBool())
	req.SetFailoverOriginTtfb(data.FailoverOriginTtfb.ValueString())
	if !data.FailoverOriginStatusCodes.IsNull() {
		var statusCodes []string
		if !data.FailoverOriginStatusCodes.IsUnknown() {
			diags.Append(data.FailoverOriginStatusCodes.ElementsAs(ctx, &statusCodes, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetFailoverOriginStatusCodes(statusCodes)
	}

	// origin_timeout: pass through as string when present (new SDK), fallback to int32 if only int is supported
	if !data.OriginTimeout.IsNull() && !data.OriginTimeout.IsUnknown() {
		val := data.OriginTimeout.ValueString()
		if m, ok := any(&req).(interface{ SetOriginTimeout(string) }); ok {
			m.SetOriginTimeout(val)
		}
	}

	// WAF configuration
	req.SetWafEnabled(data.WafEnabled.ValueBool())
	if data.WafEnabled.ValueBool() {
		wafConfig := quantadmingo.NewWAFConfigWithDefaults()
		wafConfig.SetMode(data.WafConfig.Mode.ValueString())
		wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

		// WAF rules handling
		if !data.WafConfig.AllowRules.IsNull() {
			var allowRules []string
			if !data.WafConfig.AllowRules.IsUnknown() {
				diags.Append(data.WafConfig.AllowRules.ElementsAs(ctx, &allowRules, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetAllowRules(allowRules)
		}

		if !data.WafConfig.AllowIp.IsNull() {
			var allowIp []string
			if !data.WafConfig.AllowIp.IsUnknown() {
				diags.Append(data.WafConfig.AllowIp.ElementsAs(ctx, &allowIp, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetAllowIp(allowIp)
		}

		if !data.WafConfig.BlockIp.IsNull() {
			var blockIp []string
			if !data.WafConfig.BlockIp.IsUnknown() {
				diags.Append(data.WafConfig.BlockIp.ElementsAs(ctx, &blockIp, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetBlockIp(blockIp)
		}

		if !data.WafConfig.BlockUa.IsNull() {
			var blockUa []string
			if !data.WafConfig.BlockUa.IsUnknown() {
				diags.Append(data.WafConfig.BlockUa.ElementsAs(ctx, &blockUa, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetBlockUa(blockUa)
		}

		if !data.WafConfig.BlockReferer.IsNull() {
			var blockReferer []string
			if !data.WafConfig.BlockReferer.IsUnknown() {
				diags.Append(data.WafConfig.BlockReferer.ElementsAs(ctx, &blockReferer, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetBlockReferer(blockReferer)
		}

		if !data.WafConfig.NotifyEmail.IsNull() {
			var notifyEmail []string
			if !data.WafConfig.NotifyEmail.IsUnknown() {
				diags.Append(data.WafConfig.NotifyEmail.ElementsAs(ctx, &notifyEmail, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetNotifyEmail(notifyEmail)
		}

		if !data.WafConfig.NotifySlack.IsNull() {
			wafConfig.SetNotifySlack(data.WafConfig.NotifySlack.ValueString())
		}

		if !data.WafConfig.NotifySlackHitsRpm.IsNull() {
			wafConfig.SetNotifySlackHitsRpm(int32(data.WafConfig.NotifySlackHitsRpm.ValueInt64()))
		}

		if !data.WafConfig.RequestHeaderName.IsNull() {
			wafConfig.SetRequestHeaderName(data.WafConfig.RequestHeaderName.ValueString())
		}
		req.SetWafConfig(*wafConfig)
	}

	// Make the API call
	api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyCreate(r.client.AuthContext, r.client.Organization, data.Project.ValueString()).RuleProxyRequestCreate(req).Execute()
	if err != nil {
		diags.AddError(
			"Error creating rule proxy",
			fmt.Sprintf("Could not create rule proxy, unexpected error: %s", err.Error()),
		)
		return
	}

	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())

	return
}

func callRuleProxyUpdateAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to update unknown rule, please update terraform state.",
		)
		return
	}

	req := *quantadmingo.NewRuleProxyRequestUpdateWithDefaults()
	req.SetName(data.Name.ValueString())

	// Domain handling
	if !data.Domain.IsNull() {
		var domains []string
		if !data.Domain.IsUnknown() {
			diags.Append(data.Domain.ElementsAs(ctx, &domains, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetDomain(domains)
	}

	// URL handling
	if !data.Url.IsNull() {
		var urls []string
		if !data.Url.IsUnknown() {
			diags.Append(data.Url.ElementsAs(ctx, &urls, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetUrl(urls)
	}

	// Country handling
	if !data.Country.IsNull() {
		req.SetCountry(data.Country.ValueString())
		if data.Country.ValueString() == "country_is" {
			var countryList []string
			if !data.CountryIs.IsUnknown() {
				diags.Append(data.CountryIs.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIs(countryList)
		} else if data.Country.ValueString() == "country_is_not" {
			var countryList []string
			if !data.CountryIsNot.IsUnknown() {
				diags.Append(data.CountryIsNot.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIsNot(countryList)
		}
	}

	// IP handling
	if !data.Ip.IsNull() {
		req.SetIp(data.Ip.ValueString())
		if data.Ip.ValueString() == "ip_is" {
			var ipList []string
			if !data.IpIs.IsUnknown() {
				diags.Append(data.IpIs.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIs(ipList)
		} else if data.Ip.ValueString() == "ip_is_not" {
			var ipList []string
			if !data.IpIsNot.IsUnknown() {
				diags.Append(data.IpIsNot.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIsNot(ipList)
		}
	}

	// Method handling
	if !data.Method.IsNull() {
		req.SetMethod(data.Method.ValueString())
		if data.Method.ValueString() == "method_is" {
			var methodList []string
			if !data.MethodIs.IsUnknown() {
				diags.Append(data.MethodIs.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIs(methodList)
		} else if data.Method.ValueString() == "method_is_not" {
			var methodList []string
			if !data.MethodIsNot.IsUnknown() {
				diags.Append(data.MethodIsNot.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIsNot(methodList)
		}
	}

	// Proxy configuration
	// Only set `to` when application_proxy is not true; otherwise backend computes it
	appProxy := !data.ApplicationProxy.IsNull() && !data.ApplicationProxy.IsUnknown() && data.ApplicationProxy.ValueBool()
	if !appProxy {
		if !data.To.IsNull() && !data.To.IsUnknown() && data.To.ValueString() != "" {
			req.SetTo(data.To.ValueString())
		}
	}
	if !data.Host.IsNull() && !data.Host.IsUnknown() && data.Host.ValueString() != "" {
		req.SetHost(data.Host.ValueString())
	}
	// Application proxy configuration
	if !data.ApplicationProxy.IsNull() && !data.ApplicationProxy.IsUnknown() {
		req.SetApplicationProxy(data.ApplicationProxy.ValueBool())
	}
	if !data.ApplicationName.IsNull() && !data.ApplicationName.IsUnknown() {
		req.SetApplicationName(data.ApplicationName.ValueString())
	}
	if !data.ApplicationEnvironment.IsNull() && !data.ApplicationEnvironment.IsUnknown() {
		req.SetApplicationEnvironment(data.ApplicationEnvironment.ValueString())
	}
	if !data.ApplicationContainer.IsNull() && !data.ApplicationContainer.IsUnknown() {
		req.SetApplicationContainer(data.ApplicationContainer.ValueString())
	}
	if !data.ApplicationPort.IsNull() && !data.ApplicationPort.IsUnknown() {
		// API expects integer; cast to int32 which is typical for ports
		req.SetApplicationPort(int32(data.ApplicationPort.ValueInt64()))
	}
	if !data.CacheLifetime.IsNull() && !data.CacheLifetime.IsUnknown() {
		cacheLifetime, err := parseCacheLifetime(data.CacheLifetime)
		if err != nil {
			diags.AddError(
				"Invalid cache_lifetime value",
				fmt.Sprintf("Could not parse cache_lifetime: %s", err.Error()),
			)
			return
		}
		req.SetCacheLifetime(strconv.FormatInt(cacheLifetime, 10))
	}

	if !data.AuthUser.IsNull() && !data.AuthPass.IsNull() {
		req.SetAuthUser(data.AuthUser.ValueString())
		req.SetAuthPass(data.AuthPass.ValueString())
	}

	// Weight handling
	if !data.Weight.IsNull() && !data.Weight.IsUnknown() {
		weight := int32(data.Weight.ValueInt64())
		req.SetWeight(weight)
	}

	// Strip headers handling
	if !data.ProxyStripHeaders.IsNull() && !data.ProxyStripHeaders.IsUnknown() {
		var stripHeaders []string
		diags.Append(data.ProxyStripHeaders.ElementsAs(ctx, &stripHeaders, false)...)
		if diags.HasError() {
			return
		}
		req.SetProxyStripHeaders(stripHeaders)
	} else {
		req.SetProxyStripHeaders([]string{})
	}

	req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
	req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())

	// Proxy alert configuration
	if !data.ProxyAlertEnabled.IsNull() {
		req.SetProxyAlertEnabled(data.ProxyAlertEnabled.ValueBool())
	}

	// Failover configuration
	req.SetFailoverMode(data.FailoverMode.ValueBool())
	req.SetFailoverOriginTtfb(data.FailoverOriginTtfb.ValueString())
	if !data.FailoverOriginStatusCodes.IsNull() {
		var statusCodes []string
		if !data.FailoverOriginStatusCodes.IsUnknown() {
			diags.Append(data.FailoverOriginStatusCodes.ElementsAs(ctx, &statusCodes, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetFailoverOriginStatusCodes(statusCodes)
	}

	// WAF configuration
	req.SetWafEnabled(data.WafEnabled.ValueBool())
	if data.WafEnabled.ValueBool() {
		wafConfig := *quantadmingo.NewWAFConfigUpdateWithDefaults()
		wafConfig.SetMode(data.WafConfig.Mode.ValueString())
		wafConfig.SetParanoiaLevel(int32(data.WafConfig.ParanoiaLevel.ValueInt64()))

		// WAF rules handling
		if !data.WafConfig.AllowRules.IsNull() {
			var allowRules []string
			if !data.WafConfig.AllowRules.IsUnknown() {
				diags.Append(data.WafConfig.AllowRules.ElementsAs(ctx, &allowRules, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetAllowRules(allowRules)
		}

		if !data.WafConfig.AllowIp.IsNull() {
			var allowIp []string
			if !data.WafConfig.AllowIp.IsUnknown() {
				diags.Append(data.WafConfig.AllowIp.ElementsAs(ctx, &allowIp, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetAllowIp(allowIp)
		}

		if !data.WafConfig.BlockIp.IsNull() {
			var blockIp []string
			if !data.WafConfig.BlockIp.IsUnknown() {
				diags.Append(data.WafConfig.BlockIp.ElementsAs(ctx, &blockIp, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetBlockIp(blockIp)
		}

		if !data.WafConfig.BlockUa.IsNull() {
			var blockUa []string
			if !data.WafConfig.BlockUa.IsUnknown() {
				diags.Append(data.WafConfig.BlockUa.ElementsAs(ctx, &blockUa, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetBlockUa(blockUa)
		}

		if !data.WafConfig.BlockReferer.IsNull() {
			var blockReferer []string
			if !data.WafConfig.BlockReferer.IsUnknown() {
				diags.Append(data.WafConfig.BlockReferer.ElementsAs(ctx, &blockReferer, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetBlockReferer(blockReferer)
		}

		if !data.WafConfig.NotifyEmail.IsNull() {
			var notifyEmail []string
			if !data.WafConfig.NotifyEmail.IsUnknown() {
				diags.Append(data.WafConfig.NotifyEmail.ElementsAs(ctx, &notifyEmail, false)...)
				if diags.HasError() {
					return
				}
			}
			wafConfig.SetNotifyEmail(notifyEmail)
		}

		wafConfig.SetNotifySlack(data.WafConfig.NotifySlack.ValueString())
		wafConfig.SetNotifySlackHitsRpm(int32(data.WafConfig.NotifySlackHitsRpm.ValueInt64()))
		wafConfig.SetRequestHeaderName(data.WafConfig.RequestHeaderName.ValueString())

		req.SetWafConfig(wafConfig)
	}

	// Make the API call
	_, _, err := r.client.Instance.RulesProxyAPI.RulesProxyUpdate(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.RuleId.ValueString(),
	).RuleProxyRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError(
			"Error updating rule proxy",
			fmt.Sprintf("Could not update rule proxy, unexpected error: %s", err.Error()),
		)
		return
	}

	return
}

func callRuleProxyReadAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to read unknown rule, please update terraform state.",
		)
		return
	}

	// Preserve request-only application fields from prior state
	prevApplicationProxy := data.ApplicationProxy
	prevApplicationName := data.ApplicationName
	prevApplicationEnvironment := data.ApplicationEnvironment
	prevApplicationContainer := data.ApplicationContainer
	prevApplicationPort := data.ApplicationPort

	// Add detailed logging
	fmt.Printf("Reading rule proxy with ID: %s for project: %s\n",
		data.RuleId.ValueString(), data.Project.ValueString())

	api, resp, err := r.client.Instance.RulesProxyAPI.RulesProxyRead(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.RuleId.ValueString(),
	).Execute()

	// Enhanced error handling
	if err != nil {
		// Log detailed error information
		fmt.Printf("Error response: %+v\n", resp)

		// Check if it's a 404 error, which might indicate the rule was deleted
		if resp != nil && resp.StatusCode == 404 {
			diags.AddError(
				"Rule proxy not found",
				fmt.Sprintf("The rule proxy with ID %s no longer exists. It may have been deleted outside of Terraform.",
					data.RuleId.ValueString()),
			)
			return
		}

		// For 500 errors, try to get more information
		if resp != nil && resp.StatusCode == 500 {
			// Try to list all rules to see if there's a general API issue
			allRules, _, listErr := r.client.Instance.RulesProxyAPI.RulesProxyList(
				r.client.AuthContext,
				r.client.Organization,
				data.Project.ValueString(),
			).Execute()

			if listErr == nil {
				fmt.Printf("Successfully listed %d proxy rules\n", len(allRules))
				// Check if our rule exists in the list
				ruleFound := false
				for _, rule := range allRules {
					if rule.GetRuleId() == data.RuleId.ValueString() {
						ruleFound = true
						break
					}
				}

				if ruleFound {
					fmt.Printf("Rule with ID %s exists in the list but can't be read directly\n",
						data.RuleId.ValueString())
				} else {
					fmt.Printf("Rule with ID %s does not exist in the list\n",
						data.RuleId.ValueString())
				}
			} else {
				fmt.Printf("Error listing rules: %v\n", listErr)
			}
		}

		diags.AddError(
			"Error reading rule proxy",
			fmt.Sprintf("Could not read rule proxy, unexpected error: %s", err.Error()),
		)
		return
	}

	// Get the rules action config
	actionConfig := api.GetActionConfig()

	// Set basic fields
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())
	data.Name = types.StringValue(api.GetName())
	data.Organization = types.StringValue(r.client.Organization)

	// Set values that are not present in the API response
	data.Action = types.StringValue(api.GetAction())
	data.InjectHeaders = types.MapNull(types.StringType)
	data.OnlyWithCookie = types.StringNull()
	if api.Weight != nil {
		data.Weight = types.Int64Value(int64(*api.Weight))
	} else {
		data.Weight = types.Int64Value(0)
	}

	// Convert API lists to types.List - handle nil values
	domains := api.GetDomain()
	if domains == nil {
		data.Domain = types.ListNull(types.StringType)
	} else {
		domainList, diag := types.ListValueFrom(ctx, types.StringType, domains)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.Domain = domainList
	}

	urls := api.GetUrl()
	if urls == nil {
		data.Url = types.ListNull(types.StringType)
	} else {
		urlList, diag := types.ListValueFrom(ctx, types.StringType, urls)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.Url = urlList
	}

	// Handle proxy configuration
	data.To = types.StringValue(actionConfig.GetTo())
	data.Host = types.StringValue(actionConfig.GetHost())

	// Map origin_timeout (handle string or int32 depending on SDK version)
	// Prefer using the raw field via Ok getter patterns when available
	// Fallback to known getters
	{
		// Try Ok getter if available (value, ok)
		if v, ok := actionConfig.GetOriginTimeoutOk(); ok {
			formatted := fmt.Sprintf("%v", *v)
			if formatted == "" || formatted == "0" {
				data.OriginTimeout = types.StringNull()
			} else {
				data.OriginTimeout = types.StringValue(formatted)
			}
		} else {
			// Fallback to non-Ok getter
			formatted := fmt.Sprintf("%v", actionConfig.GetOriginTimeout())
			if formatted == "" || formatted == "0" {
				data.OriginTimeout = types.StringNull()
			} else {
				data.OriginTimeout = types.StringValue(formatted)
			}
		}
	}

	// Application proxy fields are request-only; not present in response
	// Preserve values from prior state if present to keep runs consistent
	if !prevApplicationProxy.IsUnknown() && !prevApplicationProxy.IsNull() {
		data.ApplicationProxy = prevApplicationProxy
	} else {
		data.ApplicationProxy = types.BoolValue(false)
	}
	if !prevApplicationName.IsUnknown() && !prevApplicationName.IsNull() {
		data.ApplicationName = prevApplicationName
	} else {
		data.ApplicationName = types.StringNull()
	}
	if !prevApplicationEnvironment.IsUnknown() && !prevApplicationEnvironment.IsNull() {
		data.ApplicationEnvironment = prevApplicationEnvironment
	} else {
		data.ApplicationEnvironment = types.StringNull()
	}
	if !prevApplicationContainer.IsUnknown() && !prevApplicationContainer.IsNull() {
		data.ApplicationContainer = prevApplicationContainer
	} else {
		data.ApplicationContainer = types.StringNull()
	}
	if !prevApplicationPort.IsUnknown() && !prevApplicationPort.IsNull() {
		data.ApplicationPort = prevApplicationPort
	} else {
		data.ApplicationPort = types.Int64Null()
	}

	// Handle cache_lifetime - always use the API value, convert to string for backwards compatibility
	if !data.CacheLifetime.IsNull() {
		data.CacheLifetime = formatCacheLifetime(actionConfig.GetCacheLifetime())
	}
	// If data.CacheLifetime.IsNull(), leave it null (omitted case)

	data.DisableSslVerify = types.BoolValue(actionConfig.GetDisableSslVerify())
	data.OnlyProxy404 = types.BoolValue(actionConfig.GetOnlyProxy404())
	data.ProxyAlertEnabled = types.BoolValue(actionConfig.GetProxyAlertEnabled())

	// Ensure computed fields are known to Terraform after apply
	// static_error_page and static_error_page_status_codes are not present in response; set to known empty values
	data.StaticErrorPage = types.StringValue("")
	emptyList, _ := types.ListValue(types.StringType, []attr.Value{})
	data.StaticErrorPageStatusCodes = emptyList

	data.Country = types.StringValue(api.GetCountry())
	if api.GetCountry() == "country_is" {
		countries := api.GetCountryIs()
		if countries == nil {
			data.CountryIs = types.ListNull(types.StringType)
		} else {
			countriesList, diag := types.ListValueFrom(ctx, types.StringType, countries)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.CountryIs = countriesList
		}
		data.CountryIsNot = types.ListNull(types.StringType)
	} else if api.GetCountry() == "country_is_not" {
		countriesNot := api.GetCountryIsNot()
		if countriesNot == nil {
			data.CountryIsNot = types.ListNull(types.StringType)
		} else {
			countriesNotList, diag := types.ListValueFrom(ctx, types.StringType, countriesNot)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.CountryIsNot = countriesNotList
		}
		data.CountryIs = types.ListNull(types.StringType)
	} else {
		// If country is not set or is a different value, set both to null
		data.CountryIs = types.ListNull(types.StringType)
		data.CountryIsNot = types.ListNull(types.StringType)
	}

	data.Ip = types.StringValue(api.GetIp())
	if api.GetIp() == "ip_is" {
		ips := api.GetIpIs()
		if ips == nil {
			data.IpIs = types.ListNull(types.StringType)
		} else {
			isList, diag := types.ListValueFrom(ctx, types.StringType, ips)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.IpIs = isList
		}
		data.IpIsNot = types.ListNull(types.StringType)
	} else if api.GetIp() == "ip_is_not" {
		ipsNot := api.GetIpIsNot()
		if ipsNot == nil {
			data.IpIsNot = types.ListNull(types.StringType)
		} else {
			isNotList, diag := types.ListValueFrom(ctx, types.StringType, ipsNot)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.IpIsNot = isNotList
		}
		data.IpIs = types.ListNull(types.StringType)
	} else {
		data.IpIs = types.ListNull(types.StringType)
		data.IpIsNot = types.ListNull(types.StringType)
	}

	data.Method = types.StringValue(api.GetMethod())
	if api.GetMethod() == "method_is" {
		methods := api.GetMethodIs()
		if methods == nil {
			data.MethodIs = types.ListNull(types.StringType)
		} else {
			methodIsList, diag := types.ListValueFrom(ctx, types.StringType, methods)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.MethodIs = methodIsList
		}
		data.MethodIsNot = types.ListNull(types.StringType)
	} else if api.GetMethod() == "method_is_not" {
		methodsNot := api.GetMethodIsNot()
		if methodsNot == nil {
			data.MethodIsNot = types.ListNull(types.StringType)
		} else {
			methodIsNotList, diag := types.ListValueFrom(ctx, types.StringType, methodsNot)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.MethodIsNot = methodIsNotList
		}
		data.MethodIs = types.ListNull(types.StringType)
	} else {
		data.MethodIs = types.ListNull(types.StringType)
		data.MethodIsNot = types.ListNull(types.StringType)
	}

	data.Rule = types.StringNull() // Obsolete field

	stripHeaders := actionConfig.GetProxyStripHeaders()
	if stripHeaders == nil {
		data.ProxyStripHeaders = types.ListNull(types.StringType)
	} else {
		stripHeadersList, diag := types.ListValueFrom(ctx, types.StringType, stripHeaders)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.ProxyStripHeaders = stripHeadersList
	}

	proxyStripRequestHeaders := actionConfig.GetProxyStripRequestHeaders()
	if proxyStripRequestHeaders == nil {
		data.ProxyStripRequestHeaders = types.ListNull(types.StringType)
	} else {
		proxyStripRequestHeadersList, diag := types.ListValueFrom(ctx, types.StringType, proxyStripRequestHeaders)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.ProxyStripRequestHeaders = proxyStripRequestHeadersList
	}

	// Get the current state/plan values to preserve them if needed
	var planData resource_rule_proxy.RuleProxyModel
	if !data.FailoverOriginStatusCodes.IsNull() {
		planData = *data
	}

	// Handle WAF configuration
	data.WafEnabled = types.BoolValue(actionConfig.GetWafEnabled())
	if data.WafEnabled.ValueBool() {
		wafConfig := actionConfig.GetWafConfig()
		data.WafConfig.Mode = types.StringValue(wafConfig.GetMode())
		data.WafConfig.ParanoiaLevel = types.Int64Value(int64(wafConfig.GetParanoiaLevel()))

		// Convert WAF lists to types.List - handle nil values
		allowRules := wafConfig.GetAllowRules()
		if allowRules == nil {
			data.WafConfig.AllowRules = types.ListNull(types.StringType)
		} else {
			allowRulesList, diag := types.ListValueFrom(ctx, types.StringType, allowRules)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.AllowRules = allowRulesList
		}

		allowIp := wafConfig.GetAllowIp()
		if allowIp == nil {
			data.WafConfig.AllowIp = types.ListNull(types.StringType)
		} else {
			allowIpList, diag := types.ListValueFrom(ctx, types.StringType, allowIp)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.AllowIp = allowIpList
		}

		blockIp := wafConfig.GetBlockIp()
		if blockIp == nil {
			data.WafConfig.BlockIp = types.ListNull(types.StringType)
		} else {
			blockIpList, diag := types.ListValueFrom(ctx, types.StringType, blockIp)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.BlockIp = blockIpList
		}

		// Preserve values from plan if API returns empty
		if wafConfig.GetNotifySlack() == "" && !planData.WafConfig.NotifySlack.IsNull() && !planData.WafConfig.NotifySlack.IsUnknown() {
			data.WafConfig.NotifySlack = planData.WafConfig.NotifySlack
		} else {
			data.WafConfig.NotifySlack = types.StringValue(wafConfig.GetNotifySlack())
		}

		if wafConfig.GetNotifySlackHitsRpm() == 0 && !planData.WafConfig.NotifySlackHitsRpm.IsNull() && !planData.WafConfig.NotifySlackHitsRpm.IsUnknown() {
			data.WafConfig.NotifySlackHitsRpm = planData.WafConfig.NotifySlackHitsRpm
		} else {
			data.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(wafConfig.GetNotifySlackHitsRpm()))
		}

		data.WafConfig.RequestHeaderName = types.StringValue(wafConfig.GetRequestHeaderName())
	} else {
		// WAF is disabled, but if we have a waf_config block in the configuration,
		// we need to populate it with default/empty values to avoid unknown values
		if !data.WafConfig.Mode.IsNull() && !data.WafConfig.Mode.IsUnknown() {
			// Keep the configured mode (likely from test configuration)
			// Set other fields to appropriate defaults - use empty lists instead of null to match schema expectations
			emptyStringList, _ := types.ListValue(types.StringType, []attr.Value{})
			emptyBoolMap, _ := types.MapValue(types.BoolType, map[string]attr.Value{})

			data.WafConfig.AllowRules = emptyStringList
			data.WafConfig.AllowIp = emptyStringList
			data.WafConfig.BlockIp = emptyStringList
			data.WafConfig.BlockUa = emptyStringList
			data.WafConfig.BlockReferer = emptyStringList
			data.WafConfig.NotifyEmail = emptyStringList
			data.WafConfig.NotifySlack = types.StringValue("")
			data.WafConfig.NotifySlackHitsRpm = types.Int64Null()
			data.WafConfig.RequestHeaderName = types.StringValue("")
			data.WafConfig.HttpblEnabled = emptyBoolMap

			// Set default values for rate limiting fields
			data.WafConfig.IpRatelimitCooldown = types.Int64Value(30)
			data.WafConfig.IpRatelimitMode = types.StringValue("disabled")
			data.WafConfig.IpRatelimitRps = types.Int64Value(5)
			data.WafConfig.RequestHeaderRatelimitCooldown = types.Int64Value(30)
			data.WafConfig.RequestHeaderRatelimitMode = types.StringValue("disabled")
			data.WafConfig.RequestHeaderRatelimitRps = types.Int64Value(5)
			data.WafConfig.WafRatelimitCooldown = types.Int64Value(300)
			data.WafConfig.WafRatelimitHits = types.Int64Value(10)
			data.WafConfig.WafRatelimitMode = types.StringValue("disabled")
			data.WafConfig.WafRatelimitRps = types.Int64Value(5)

			// Keep the paranoia level from configuration or use default
			if data.WafConfig.ParanoiaLevel.IsNull() || data.WafConfig.ParanoiaLevel.IsUnknown() {
				data.WafConfig.ParanoiaLevel = types.Int64Value(1)
			}
		}
	}

	// Handle failover configuration
	data.FailoverMode = types.BoolValue(actionConfig.GetFailoverMode())
	data.FailoverOriginTtfb = types.StringValue(actionConfig.GetFailoverOriginTtfb())

	// Preserve failover status codes if API returns empty but we had values
	failoverStatusCodes := actionConfig.GetFailoverOriginStatusCodes()
	if failoverStatusCodes == nil {
		if !planData.FailoverOriginStatusCodes.IsNull() && !planData.FailoverOriginStatusCodes.IsUnknown() {
			data.FailoverOriginStatusCodes = planData.FailoverOriginStatusCodes
		} else {
			data.FailoverOriginStatusCodes = types.ListNull(types.StringType)
		}
	} else if len(failoverStatusCodes) == 0 && !planData.FailoverOriginStatusCodes.IsNull() && !planData.FailoverOriginStatusCodes.IsUnknown() {
		data.FailoverOriginStatusCodes = planData.FailoverOriginStatusCodes
	} else {
		statusCodesList, diag := types.ListValueFrom(ctx, types.StringType, failoverStatusCodes)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.FailoverOriginStatusCodes = statusCodesList
	}

	notifycfg := actionConfig.GetNotifyConfig()
	data.Notify = types.StringValue(*actionConfig.Notify)

	originStatusCodes := notifycfg.GetOriginStatusCodes()
	var originStatusCodesList basetypes.ListValue
	if originStatusCodes == nil {
		originStatusCodesList = types.ListNull(types.StringType)
	} else {
		var diag diag.Diagnostics
		originStatusCodesList, diag = types.ListValueFrom(ctx, types.StringType, originStatusCodes)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
	}

	data.NotifyConfig = resource_rule_proxy.NotifyConfigValue{
		OriginStatusCodes: originStatusCodesList,
		Period:            types.StringValue(notifycfg.GetPeriod()),
		SlackWebhook:      types.StringValue(notifycfg.GetSlackWebhook()),
	}

	return
}

func callRuleProxyDeleteAPI(ctx context.Context, r *ruleProxyResource, data *resource_rule_proxy.RuleProxyModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
		return
	}

	_, err := r.client.Instance.RulesProxyAPI.RulesProxyDelete(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.RuleId.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError(
			"Error deleting rule proxy",
			fmt.Sprintf("Could not delete rule proxy, unexpected error: %s", err.Error()),
		)
		return
	}

	return
}
