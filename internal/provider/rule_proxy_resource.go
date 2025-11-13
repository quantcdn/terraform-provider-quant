package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_proxy"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

	// No need to read immediately after create - we have all the data from the create response

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

	// Read after update to populate computed fields correctly
	// Note: We capture the new UUID in callRuleProxyUpdateAPI first, then read with it
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
	req := *quantadmingo.NewV2RuleProxyRequestWithDefaults()
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
		wafConfig := quantadmingo.NewWafConfig()
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

		// V2 API may not have RequestHeaderName in WafConfig
		// if !data.WafConfig.RequestHeaderName.IsNull() {
		// 	wafConfig.SetRequestHeaderName(data.WafConfig.RequestHeaderName.ValueString())
		// }
		req.SetWafConfig(*wafConfig)
	}

	// Make the API call
	api, _, err := r.client.Instance.RulesAPI.RulesProxyCreate(r.client.AuthContext, r.client.Organization, data.Project.ValueString()).V2RuleProxyRequest(req).Execute()
	if err != nil {
		diags.AddError(
			"Error creating rule proxy",
			fmt.Sprintf("Could not create rule proxy, unexpected error: %s", err.Error()),
		)
		return
	}

	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())
	data.Organization = types.StringValue(r.client.Organization)
	data.Action = types.StringValue("proxy")

	// Set fields that may not be returned by API to null/empty
	data.Rule = types.StringValue("")

	// Get action_config for additional fields
	actionConfig := api.GetActionConfig()

	// StaticErrorPage fields ARE supported in action_config (API team confirmed)
	if actionConfig.HasStaticErrorPage() {
		data.StaticErrorPage = types.StringValue(actionConfig.GetStaticErrorPage())
	} else {
		data.StaticErrorPage = types.StringNull()
	}
	if actionConfig.HasStaticErrorPageStatusCodes() {
		statusCodes, _ := types.ListValueFrom(ctx, types.StringType, actionConfig.GetStaticErrorPageStatusCodes())
		data.StaticErrorPageStatusCodes = statusCodes
	} else {
		emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		data.StaticErrorPageStatusCodes = emptyList
	}

	// Only set proxy_strip_headers to empty if not already set in config
	if data.ProxyStripHeaders.IsNull() || data.ProxyStripHeaders.IsUnknown() {
		emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		data.ProxyStripHeaders = emptyList
	}
	if data.ProxyStripRequestHeaders.IsNull() || data.ProxyStripRequestHeaders.IsUnknown() {
		emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		data.ProxyStripRequestHeaders = emptyList
	}

	// Set only_with_cookie if returned
	if api.OnlyWithCookie != nil && *api.OnlyWithCookie != "" {
		data.OnlyWithCookie = types.StringValue(*api.OnlyWithCookie)
	} else {
		data.OnlyWithCookie = types.StringNull()
	}

	// Set optional fields to null if not in input
	if data.OriginTimeout.IsNull() || data.OriginTimeout.IsUnknown() {
		data.OriginTimeout = types.StringNull()
	}
	if data.ProxyAlertEnabled.IsNull() || data.ProxyAlertEnabled.IsUnknown() {
		data.ProxyAlertEnabled = types.BoolNull()
	}
	if data.CacheLifetime.IsNull() || data.CacheLifetime.IsUnknown() {
		data.CacheLifetime = types.StringNull()
	}

	// Application proxy fields
	if data.ApplicationName.IsNull() || data.ApplicationName.IsUnknown() {
		data.ApplicationName = types.StringNull()
	}
	if data.ApplicationEnvironment.IsNull() || data.ApplicationEnvironment.IsUnknown() {
		data.ApplicationEnvironment = types.StringNull()
	}
	if data.ApplicationContainer.IsNull() || data.ApplicationContainer.IsUnknown() {
		data.ApplicationContainer = types.StringNull()
	}
	if data.ApplicationPort.IsNull() || data.ApplicationPort.IsUnknown() {
		data.ApplicationPort = types.Int64Null()
	}

	// Set unused conditional lists to empty based on the condition type
	// API only returns the list that matches the condition (e.g., if country="country_is", only country_is is returned)
	emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
	if data.Method.IsNull() || data.Method.IsUnknown() {
		data.Method = types.StringNull()
		data.MethodIs = emptyList
		data.MethodIsNot = emptyList
	} else if data.Method.ValueString() == "method_is" {
		if data.MethodIsNot.IsNull() || data.MethodIsNot.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.MethodIsNot = emptyList
		}
	} else if data.Method.ValueString() == "method_is_not" {
		if data.MethodIs.IsNull() || data.MethodIs.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.MethodIs = emptyList
		}
	}

	if data.Country.IsNull() || data.Country.IsUnknown() {
		data.Country = types.StringNull()
		data.CountryIs = emptyList
		data.CountryIsNot = emptyList
	} else if data.Country.ValueString() == "country_is" {
		if data.CountryIsNot.IsNull() || data.CountryIsNot.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.CountryIsNot = emptyList
		}
	} else if data.Country.ValueString() == "country_is_not" {
		if data.CountryIs.IsNull() || data.CountryIs.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.CountryIs = emptyList
		}
	}

	if data.Ip.IsNull() || data.Ip.IsUnknown() {
		data.Ip = types.StringNull()
		data.IpIs = emptyList
		data.IpIsNot = emptyList
	} else if data.Ip.ValueString() == "ip_is" {
		if data.IpIsNot.IsNull() || data.IpIsNot.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.IpIsNot = emptyList
		}
	} else if data.Ip.ValueString() == "ip_is_not" {
		if data.IpIs.IsNull() || data.IpIs.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.IpIs = emptyList
		}
	}

	// Lists that should be empty if not provided
	if data.FailoverOriginStatusCodes.IsNull() || data.FailoverOriginStatusCodes.IsUnknown() {
		data.FailoverOriginStatusCodes = emptyList
	}
	if data.InjectHeaders.IsNull() || data.InjectHeaders.IsUnknown() {
		emptyMap, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
		data.InjectHeaders = emptyMap
	}

	// Note: NotifyConfig no longer at top-level in V2 schema (it's in action_config)
	// Note: WafConfig.RequestHeaderName removed - use waf_config.thresholds instead

	// Note: failover_s3_bucket and failover_s3_region have been removed from the schema as they are no longer supported by the API

	// Set action_config to null if unknown (not returned in V2 create response)
	if data.ActionConfig.IsNull() || data.ActionConfig.IsUnknown() {
		data.ActionConfig = resource_rule_proxy.NewActionConfigValueNull()
	}

	// Set host to null if unknown
	if data.Host.IsNull() || data.Host.IsUnknown() {
		data.Host = types.StringNull()
	}

	// Set waf_config to null if unknown
	if data.WafConfig.IsNull() || data.WafConfig.IsUnknown() {
		data.WafConfig = resource_rule_proxy.NewWafConfigValueNull()
	}

	// Explicitly set WafConfig nested fields that weren't in the input
	// These need to be set to prevent "unknown value" errors
	if !data.WafConfig.IsNull() && !data.WafConfig.IsUnknown() {
		// If these fields are unknown, set them to null/empty
		if data.WafConfig.NotifySlack.IsUnknown() {
			data.WafConfig.NotifySlack = types.StringNull()
		}
		if data.WafConfig.NotifySlackHitsRpm.IsUnknown() {
			data.WafConfig.NotifySlackHitsRpm = types.Int64Null()
		}
		if data.WafConfig.AllowIp.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.AllowIp = emptyList
		}
		if data.WafConfig.AllowRules.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.AllowRules = emptyList
		}
		if data.WafConfig.BlockAsn.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.BlockAsn = emptyList
		}
		if data.WafConfig.BlockIp.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.BlockIp = emptyList
		}
		if data.WafConfig.BlockLists.IsUnknown() {
			data.WafConfig.BlockLists = types.ObjectNull(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx))
		}
		if data.WafConfig.BlockReferer.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.BlockReferer = emptyList
		}
		if data.WafConfig.BlockUa.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.BlockUa = emptyList
		}
		if data.WafConfig.Httpbl.IsUnknown() {
			data.WafConfig.Httpbl = types.ObjectNull(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx))
		}
		if data.WafConfig.NotifyEmail.IsUnknown() {
			emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
			data.WafConfig.NotifyEmail = emptyList
		}

		// Set thresholds to empty list if unknown (API may not return it)
		if data.WafConfig.Thresholds.IsUnknown() {
			emptyThresholdsList, _ := types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx),
			}, []attr.Value{})
			data.WafConfig.Thresholds = emptyThresholdsList
		}
	}

	// Read back from API to get computed fields and ensure state consistency
	// The DB-backed API has eliminated eventual consistency, so this is safe
	readDiags := callRuleProxyReadAPI(ctx, r, data)
	diags.Append(readDiags...)

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

	req := *quantadmingo.NewV2RuleProxyRequestWithDefaults()
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
		wafConfig := quantadmingo.NewWafConfig()
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
		// V2 API may not have RequestHeaderName in WafConfig
		// wafConfig.SetRequestHeaderName(data.WafConfig.RequestHeaderName.ValueString())

		req.SetWafConfig(*wafConfig)
	}

	// Make the API call
	api, _, err := r.client.Instance.RulesAPI.RulesProxyUpdate(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.Uuid.ValueString(),
	).V2RuleProxyRequest(req).Execute()

	if err != nil {
		diags.AddError(
			"Error updating rule proxy",
			fmt.Sprintf("Could not update rule proxy, unexpected error: %s", err.Error()),
		)
		return
	}

	// CRITICAL: UUID changes after every update - must capture the new UUID from the response
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())

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
	prevTo := data.To // Preserve 'to' field for application proxy

	// Add detailed logging
	tflog.Debug(ctx, "=== RULE PROXY READ REQUEST ===", map[string]interface{}{
		"organization": r.client.Organization,
		"project":      data.Project.ValueString(),
		"rule_id":      data.RuleId.ValueString(),
		"uuid":         data.Uuid.ValueString(),
	})

	// Use shared retry logic for eventual consistency
	api, httpResp, err := utils.RetryRuleRead(ctx, func() (*quantadmingo.V2RuleProxy, *http.Response, error) {
		return r.client.Instance.RulesAPI.RulesProxyRead(
			r.client.AuthContext,
			r.client.Organization,
			data.Project.ValueString(),
			data.Uuid.ValueString(),
		).Execute()
	}, "rule_proxy")

	// Debug: Log the API response
	if httpResp != nil && httpResp.Body != nil {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		tflog.Debug(ctx, "=== RULE PROXY READ RESPONSE ===", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(bodyBytes),
		})
	}

	// Enhanced error handling
	if err != nil {
		// Try to parse API error response
		if httpResp != nil && httpResp.Body != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			var apiError struct {
				Error   bool   `json:"error"`
				Message string `json:"message"`
			}
			if jsonErr := json.Unmarshal(bodyBytes, &apiError); jsonErr == nil && apiError.Message != "" {
				diags.AddError("Failed to read rule proxy", apiError.Message)
				return
			}
		}
		// Check if it's a 404 error, which might indicate the rule was deleted
		if httpResp != nil && httpResp.StatusCode == 404 {
			diags.AddError(
				"Rule proxy not found",
				fmt.Sprintf("The rule proxy with ID %s no longer exists. It may have been deleted outside of Terraform.",
					data.RuleId.ValueString()),
			)
			return
		}

		// For 500 errors, try to get more information
		if httpResp != nil && httpResp.StatusCode == 500 {
			// Try to list all rules to see if there's a general API issue
			allRules, _, listErr := r.client.Instance.RulesAPI.RulesProxyList(
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
	// When application_proxy is true, API auto-generates the 'to' field
	// Keep the original value (empty string) instead of API's generated URL to avoid inconsistency
	if !prevApplicationProxy.IsNull() && prevApplicationProxy.ValueBool() {
		data.To = prevTo
	} else {
		data.To = types.StringValue(actionConfig.GetTo())
	}
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

	// Note: StaticErrorPage and StaticErrorPageStatusCodes no longer in V2 schema

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

		blockUa := wafConfig.GetBlockUa()
		if blockUa == nil {
			data.WafConfig.BlockUa = types.ListNull(types.StringType)
		} else {
			blockUaList, diag := types.ListValueFrom(ctx, types.StringType, blockUa)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.BlockUa = blockUaList
		}

		blockReferer := wafConfig.GetBlockReferer()
		if blockReferer == nil {
			data.WafConfig.BlockReferer = types.ListNull(types.StringType)
		} else {
			blockRefererList, diag := types.ListValueFrom(ctx, types.StringType, blockReferer)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.BlockReferer = blockRefererList
		}

		blockAsn := wafConfig.GetBlockAsn()
		if blockAsn == nil {
			data.WafConfig.BlockAsn = types.ListNull(types.StringType)
		} else {
			blockAsnList, diag := types.ListValueFrom(ctx, types.StringType, blockAsn)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.BlockAsn = blockAsnList
		}

		notifyEmail := wafConfig.GetNotifyEmail()
		if notifyEmail == nil {
			data.WafConfig.NotifyEmail = types.ListNull(types.StringType)
		} else {
			notifyEmailList, diag := types.ListValueFrom(ctx, types.StringType, notifyEmail)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.WafConfig.NotifyEmail = notifyEmailList
		}

		// Set nested objects to null - these are returned by API but we'll set them to null for simplicity
		// They can be properly handled if users actually configure them
		data.WafConfig.Httpbl = types.ObjectNull(resource_rule_proxy.HttpblValue{}.AttributeTypes(ctx))
		data.WafConfig.BlockLists = types.ObjectNull(resource_rule_proxy.BlockListsValue{}.AttributeTypes(ctx))

		// Thresholds - set to empty list for now (API returns defaults but we'll ignore them unless user configures them)
		emptyThresholdsList, _ := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: resource_rule_proxy.ThresholdsValue{}.AttributeTypes(ctx)}, []resource_rule_proxy.ThresholdsValue{})
		data.WafConfig.Thresholds = emptyThresholdsList

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

		// V2 API may not have RequestHeaderName in WafConfig
		// data.WafConfig.RequestHeaderName = types.StringValue(wafConfig.GetRequestHeaderName())
	} else {
		// WAF is disabled, but if we have a waf_config block in the configuration,
		// we need to populate it with default/empty values to avoid unknown values
		if !data.WafConfig.Mode.IsNull() && !data.WafConfig.Mode.IsUnknown() {
			// Keep the configured mode (likely from test configuration)
			// Set other fields to appropriate defaults - use empty lists instead of null to match schema expectations
			emptyStringList, _ := types.ListValue(types.StringType, []attr.Value{})

			data.WafConfig.AllowRules = emptyStringList
			data.WafConfig.AllowIp = emptyStringList
			data.WafConfig.BlockIp = emptyStringList
			data.WafConfig.BlockUa = emptyStringList
			data.WafConfig.BlockReferer = emptyStringList
			data.WafConfig.NotifyEmail = emptyStringList
			data.WafConfig.NotifySlack = types.StringValue("")
			data.WafConfig.NotifySlackHitsRpm = types.Int64Null()
			// Note: These flattened fields no longer exist, use waf_config.thresholds instead:
			// RequestHeaderName, HttpblEnabled, IpRatelimit*, RequestHeaderRatelimit*, WafRatelimit*

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

	// Note: Notify and NotifyConfig no longer exist at top-level in V2 schema
	// notifycfg := actionConfig.GetNotifyConfig()
	// These fields moved to action_config.notify_config in V2

	// Set action_config to null since we expose all its fields as top-level attributes
	// This prevents "unknown value" errors
	data.ActionConfig = resource_rule_proxy.NewActionConfigValueNull()

	// Set static_error_page fields to null if they weren't populated
	if data.StaticErrorPage.IsUnknown() {
		data.StaticErrorPage = types.StringNull()
	}
	if data.StaticErrorPageStatusCodes.IsUnknown() {
		data.StaticErrorPageStatusCodes = types.ListNull(types.StringType)
	}

	// If waf_config wasn't properly populated, set to null
	// This happens when waf_enabled is set but no waf_config block in TF
	if data.WafConfig.IsUnknown() || data.WafConfig.Mode.IsUnknown() {
		data.WafConfig = resource_rule_proxy.NewWafConfigValueNull()
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

	_, err := r.client.Instance.RulesAPI.RulesProxyDelete(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.Uuid.ValueString(),
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
