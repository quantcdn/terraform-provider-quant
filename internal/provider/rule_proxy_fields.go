package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_proxy"
)

// The helpers in this file split the proxy request build and read-back into
// small steps. They keep the behavior of the inline code they replace.

// setProxyTarget sends 'to' unless application_proxy is true; the backend
// computes 'to' for an application proxy, and the mapper already sent the
// application fields.
func setProxyTarget(data *resource_rule_proxy.RuleProxyModel, req *quantadmingo.V2RuleProxyRequest) {
	if isKnown(data.ApplicationProxy) && data.ApplicationProxy.ValueBool() {
		return
	}
	if isKnown(data.To) && data.To.ValueString() != "" {
		req.SetTo(data.To.ValueString())
	}
}

// setProxyCacheLifetime validates and sends cache_lifetime, a string in the
// schema that must hold an integer.
func setProxyCacheLifetime(data *resource_rule_proxy.RuleProxyModel, req *quantadmingo.V2RuleProxyRequest) diag.Diagnostics {
	var diags diag.Diagnostics
	if !isKnown(data.CacheLifetime) || data.CacheLifetime.ValueString() == "" {
		return diags
	}
	cacheLifetime, err := parseCacheLifetime(data.CacheLifetime)
	if err != nil {
		diags.AddError("Invalid cache_lifetime value",
			fmt.Sprintf("Could not parse cache_lifetime: %s", err.Error()))
		return diags
	}
	req.SetCacheLifetime(strconv.FormatInt(cacheLifetime, 10))
	return diags
}

// setProxyInjectHeaders sends inject_headers, a map the mapper does not handle.
func setProxyInjectHeaders(ctx context.Context, data *resource_rule_proxy.RuleProxyModel, req *quantadmingo.V2RuleProxyRequest) diag.Diagnostics {
	if !isKnown(data.InjectHeaders) {
		return nil
	}
	hdrs := make(map[string]string)
	diags := data.InjectHeaders.ElementsAs(ctx, &hdrs, false)
	if diags.HasError() {
		return diags
	}
	req.SetInjectHeaders(hdrs)
	return diags
}

// injectHeadersFromAPI reads inject_headers from the action config. An absent
// or empty map reads back as null.
func injectHeadersFromAPI(actionConfig quantadmingo.V2RuleProxyAction) types.Map {
	hdrs := actionConfig.GetInjectHeaders()
	if len(hdrs) == 0 {
		return types.MapNull(types.StringType)
	}
	hdrMap := make(map[string]attr.Value, len(hdrs))
	for k, v := range hdrs {
		hdrMap[k] = types.StringValue(v)
	}
	m, _ := types.MapValue(types.StringType, hdrMap)
	return m
}

// setProxyTargetFromAPI reads 'to' and host. When application_proxy is true
// the API generates 'to', so the prior value is kept.
func setProxyTargetFromAPI(actionConfig quantadmingo.V2RuleProxyAction, prior, data *resource_rule_proxy.RuleProxyModel) {
	if !prior.ApplicationProxy.IsNull() && prior.ApplicationProxy.ValueBool() {
		data.To = prior.To
	} else {
		data.To = types.StringValue(actionConfig.GetTo())
	}
	if h := actionConfig.GetHost(); h != "" {
		data.Host = types.StringValue(h)
	} else {
		data.Host = types.StringNull()
	}
}

// originTimeoutFromAPI reads origin_timeout; absent, empty or 0 is null.
func originTimeoutFromAPI(actionConfig quantadmingo.V2RuleProxyAction) types.String {
	v, ok := actionConfig.GetOriginTimeoutOk()
	if !ok {
		return types.StringNull()
	}
	formatted := fmt.Sprintf("%v", *v)
	if formatted == "" || formatted == "0" {
		return types.StringNull()
	}
	return types.StringValue(formatted)
}

// restoreProxyApplicationFields keeps the request-only application fields
// from the prior value, because the API does not return them.
func restoreProxyApplicationFields(prior, data *resource_rule_proxy.RuleProxyModel) {
	data.ApplicationProxy = types.BoolValue(false)
	if isKnown(prior.ApplicationProxy) {
		data.ApplicationProxy = prior.ApplicationProxy
	}
	data.ApplicationName = knownStringOrNull(prior.ApplicationName)
	data.ApplicationEnvironment = knownStringOrNull(prior.ApplicationEnvironment)
	data.ApplicationContainer = knownStringOrNull(prior.ApplicationContainer)
	data.ApplicationPort = types.Int64Null()
	if isKnown(prior.ApplicationPort) {
		data.ApplicationPort = prior.ApplicationPort
	}
}

func knownStringOrNull(v types.String) types.String {
	if isKnown(v) {
		return v
	}
	return types.StringNull()
}

// setProxyFailoverFromAPI reads the failover fields. An absent or "0"
// lifetime or TTFB keeps a known prior value and turns unknown into null.
func setProxyFailoverFromAPI(ctx context.Context, actionConfig quantadmingo.V2RuleProxyAction, data *resource_rule_proxy.RuleProxyModel) diag.Diagnostics {
	data.FailoverMode = types.BoolValue(actionConfig.GetFailoverMode())
	data.FailoverLifetime = nonZeroStringOrPrior(actionConfig.GetFailoverLifetime(), data.FailoverLifetime)
	data.FailoverOriginTtfb = nonZeroStringOrPrior(actionConfig.GetFailoverOriginTtfb(), data.FailoverOriginTtfb)

	codes := actionConfig.GetFailoverOriginStatusCodes()
	if codes == nil {
		data.FailoverOriginStatusCodes = types.ListNull(types.StringType)
		return nil
	}
	codesList, diags := types.ListValueFrom(ctx, types.StringType, codes)
	data.FailoverOriginStatusCodes = codesList
	return diags
}

func nonZeroStringOrPrior(v string, prior types.String) types.String {
	if v != "" && v != "0" {
		return types.StringValue(v)
	}
	if prior.IsUnknown() {
		return types.StringNull()
	}
	return prior
}

// setProxyStaticErrorPageFromAPI reads the static error page fields. An
// absent value keeps a known prior value and turns unknown into null.
func setProxyStaticErrorPageFromAPI(ctx context.Context, actionConfig quantadmingo.V2RuleProxyAction, data *resource_rule_proxy.RuleProxyModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if v := actionConfig.GetStaticErrorPage(); v != "" {
		data.StaticErrorPage = types.StringValue(v)
	} else if data.StaticErrorPage.IsUnknown() {
		data.StaticErrorPage = types.StringNull()
	}
	if codes := actionConfig.GetStaticErrorPageStatusCodes(); len(codes) > 0 {
		diags.Append(setStringListFromAPI(ctx, codes, &data.StaticErrorPageStatusCodes)...)
	} else if data.StaticErrorPageStatusCodes.IsUnknown() {
		data.StaticErrorPageStatusCodes = types.ListNull(types.StringType)
	}
	return diags
}
