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
    req := *quantadmingo.NewRuleProxyRequestWithDefaults()
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
    req.SetTo(data.To.ValueString())
    req.SetHost(data.Host.ValueString())
    req.SetCacheLifetime(int32(data.CacheLifetime.ValueInt64()))

    if !data.AuthUser.IsNull() && !data.AuthPass.IsNull() {
        req.SetAuthUser(data.AuthUser.ValueString())
        req.SetAuthPass(data.AuthPass.ValueString())
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


    // Failover configuration
    failover := quantadmingo.NewFailoverConfigWithDefaults()
    failover.SetFailoverMode(data.Failover.FailoverMode.ValueBool())
    failover.SetFailoverOriginTtfb(data.Failover.FailoverOriginTtfb.ValueString())

	if !data.Failover.FailoverOriginStatusCodes.IsNull() {
		var statusCodes []string
		if !data.Failover.FailoverOriginStatusCodes.IsUnknown() {
			diags.Append(data.Failover.FailoverOriginStatusCodes.ElementsAs(ctx, &statusCodes, false)...)
			if diags.HasError() {
				return
			}
		}
		failover.SetFailoverOriginStatusCodes(statusCodes)
	}
    req.SetFailover(*failover)

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
    api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyCreate(r.client.AuthContext, r.client.Organization, data.Project.ValueString()).RuleProxyRequest(req).Execute()
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
    req.SetTo(data.To.ValueString())
    req.SetHost(data.Host.ValueString())
    req.SetCacheLifetime(int32(data.CacheLifetime.ValueInt64()))

    if !data.AuthUser.IsNull() && !data.AuthPass.IsNull() {
        req.SetAuthUser(data.AuthUser.ValueString())
        req.SetAuthPass(data.AuthPass.ValueString())
    }

    // Strip headers handling
    var stripHeaders []string
    diags.Append(data.ProxyStripHeaders.ElementsAs(ctx, &stripHeaders, false)...)
    if diags.HasError() {
        return
    }
    req.SetProxyStripHeaders(stripHeaders)

    req.SetDisableSslVerify(data.DisableSslVerify.ValueBool())
    req.SetOnlyProxy404(data.OnlyProxy404.ValueBool())


    // Failover configuration
    failover := quantadmingo.NewFailoverConfigWithDefaults()
    failover.SetFailoverMode(data.Failover.FailoverMode.ValueBool())
    failover.SetFailoverOriginTtfb(data.Failover.FailoverOriginTtfb.ValueString())

	if !data.Failover.FailoverOriginStatusCodes.IsNull() {
		var statusCodes []string
		if !data.Failover.FailoverOriginStatusCodes.IsUnknown() {
			diags.Append(data.Failover.FailoverOriginStatusCodes.ElementsAs(ctx, &statusCodes, false)...)
			if diags.HasError() {
				return
			}
		}
		failover.SetFailoverOriginStatusCodes(statusCodes)
	}
    req.SetFailover(*failover)

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

    api, _, err := r.client.Instance.RulesProxyAPI.RulesProxyRead(
        r.client.AuthContext,
        r.client.Organization,
        data.Project.ValueString(),
        data.RuleId.ValueString(),
    ).Execute()

    if err != nil {
        diags.AddError(
            "Error reading rule proxy",
            fmt.Sprintf("Could not read rule proxy, unexpected error: %s", err.Error()),
        )
        return
    }

    // Set basic fields
    data.Uuid = types.StringValue(api.GetUuid())
    data.RuleId = types.StringValue(api.GetRuleId())
    data.Name = types.StringValue(api.GetName())
    data.Organization = types.StringValue(r.client.Organization)

	// Set values that are not present in the API response
	data.Action = types.StringValue(api.GetAction())
	data.InjectHeaders = types.MapNull(types.StringType)
	data.OnlyWithCookie = types.StringNull()
	data.Weight = types.Int64Value(0)

    // Convert API lists to types.List
    domainList, diag := types.ListValueFrom(ctx, types.StringType, api.GetDomain())
    if diag.HasError() {
        diags.Append(diag...)
        return
    }
    data.Domain = domainList

    urlList, diag := types.ListValueFrom(ctx, types.StringType, api.GetUrl())
    if diag.HasError() {
        diags.Append(diag...)
        return
    }
    data.Url = urlList

    // Handle proxy configuration
    data.To = types.StringValue(api.GetActionConfig().To)
    data.Host = types.StringValue(*api.GetActionConfig().Host)
	if lifetime := api.GetActionConfig().CacheLifetime; lifetime != nil {
		data.CacheLifetime = types.Int64Value(int64(*lifetime))
	} else {
		data.CacheLifetime = types.Int64Null()
	}
    data.DisableSslVerify = types.BoolValue(*api.GetActionConfig().DisableSslVerify)
    data.OnlyProxy404 = types.BoolValue(*api.GetActionConfig().OnlyProxy404)

	data.Country = types.StringValue(api.GetCountry())
    if api.GetCountry() == "country_is" {
        countriesList, diag := types.ListValueFrom(ctx, types.StringType, api.GetCountryIs())
        if diag.HasError() {
            diags.Append(diag...)
            return
        }
        data.CountryIs = countriesList
        data.CountryIsNot = types.ListNull(types.StringType)
    } else if api.GetCountry() == "country_is_not" {
        countriesNotList, diag := types.ListValueFrom(ctx, types.StringType, api.GetCountryIsNot())
        if diag.HasError() {
            diags.Append(diag...)
            return
        }
        data.CountryIs = types.ListNull(types.StringType)
        data.CountryIsNot = countriesNotList
    } else {
        // If country is not set or is a different value, set both to null
        data.CountryIs = types.ListNull(types.StringType)
        data.CountryIsNot = types.ListNull(types.StringType)
    }

	data.Ip = types.StringValue(api.GetIp())
	if api.GetIp() == "ip_is" {
		isList, diag := types.ListValueFrom(ctx, types.StringType, api.GetIpIs())
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.IpIs = isList
		data.IpIsNot = types.ListNull(types.StringType)
	} else if api.GetIp() == "ip_is_not" {
		isNotList, diag := types.ListValueFrom(ctx, types.StringType, api.GetIpIsNot())
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.IpIs = types.ListNull(types.StringType)
		data.IpIsNot = isNotList
	} else {
		data.IpIs = types.ListNull(types.StringType)
		data.IpIsNot = types.ListNull(types.StringType)
	}

	data.Method = types.StringValue(api.GetMethod())
	if api.GetMethod() == "method_is" {
		methodIsList, diag := types.ListValueFrom(ctx, types.StringType, api.GetMethodIs())
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.MethodIs = methodIsList
		data.MethodIsNot = types.ListNull(types.StringType)
	} else if api.GetMethod() == "method_is_not" {
		methodIsNotList, diag := types.ListValueFrom(ctx, types.StringType, api.GetMethodIsNot())
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.MethodIs = types.ListNull(types.StringType)
		data.MethodIsNot = methodIsNotList
	} else {
		data.MethodIs = types.ListNull(types.StringType)
		data.MethodIsNot = types.ListNull(types.StringType)
	}

	data.Rule = types.StringNull() // Obsolete field

    stripHeadersList, diag := types.ListValueFrom(ctx, types.StringType, api.GetActionConfig().ProxyStripHeaders)
    if diag.HasError() {
        diags.Append(diag...)
        return
    }
    data.ProxyStripHeaders = stripHeadersList
	proxyStripRequestHeadersList, diag := types.ListValueFrom(ctx, types.StringType, api.GetActionConfig().ProxyStripRequestHeaders)
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	data.ProxyStripRequestHeaders = proxyStripRequestHeadersList

    // Handle WAF configuration
    data.WafEnabled = types.BoolValue(api.GetActionConfig().WafEnabled)
    if data.WafEnabled.ValueBool() == true {
        wafConfig := api.GetActionConfig().WafConfig
        data.WafConfig.Mode = types.StringValue(wafConfig.GetMode())
        data.WafConfig.ParanoiaLevel = types.Int64Value(int64(wafConfig.GetParanoiaLevel()))

        // Convert WAF lists to types.List
        allowRulesList, diag := types.ListValueFrom(ctx, types.StringType, wafConfig.GetAllowRules())
        if diag.HasError() {
            diags.Append(diag...)
            return
        }
        data.WafConfig.AllowRules = allowRulesList

        allowIpList, diag := types.ListValueFrom(ctx, types.StringType, wafConfig.GetAllowIp())
        if diag.HasError() {
            diags.Append(diag...)
            return
        }
        data.WafConfig.AllowIp = allowIpList

        blockIpList, diag := types.ListValueFrom(ctx, types.StringType, wafConfig.GetBlockIp())
        if diag.HasError() {
            diags.Append(diag...)
            return
        }
        data.WafConfig.BlockIp = blockIpList

        // Set other WAF fields
        data.WafConfig.NotifySlack = types.StringValue(wafConfig.GetNotifySlack())
		data.WafConfig.NotifySlackHitsRpm = types.Int64Value(int64(wafConfig.GetNotifySlackHitsRpm()))
        data.WafConfig.RequestHeaderName = types.StringValue(wafConfig.GetRequestHeaderName())
		// @todo support thresholds.
		// data.WafConfig.Thresholds = types.ListNull(resource_rule_proxy.ThresholdsValue. []attr.Value{})
    }

	failover := api.GetActionConfig().Failover

	data.Failover = resource_rule_proxy.FailoverValue{
		FailoverMode: types.BoolValue(failover.GetFailoverMode()),
		FailoverOriginTtfb: types.StringValue(failover.GetFailoverOriginTtfb()),
	}
	statusCodesList, diag := types.ListValueFrom(ctx, types.StringType, failover.GetFailoverOriginStatusCodes())
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	data.Failover.FailoverOriginStatusCodes = statusCodesList

	notifycfg := api.GetActionConfig().NotifyConfig
	data.Notify = types.StringValue(*api.GetActionConfig().Notify)

	originStatusCodesList, diag := types.ListValueFrom(ctx, types.StringType, notifycfg.GetOriginStatusCodes())
	if diag.HasError() {
		diags.Append(diag...)
		return
	}
	data.NotifyConfig = resource_rule_proxy.NotifyConfigValue{
		OriginStatusCodes: originStatusCodesList,
		Period: types.StringValue(notifycfg.GetPeriod()),
		SlackWebhook: types.StringValue(notifycfg.GetSlackWebhook()),
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