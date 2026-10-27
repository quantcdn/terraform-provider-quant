package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_auth"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_bot_challenge"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_content_filter"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_custom_response"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_function"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_headers"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_proxy"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_redirect"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_rule_serve_static"
)

// ruleExtraConditions points at the model fields of the asn, ua and
// traffic_source condition triplets and of cookie_name. Every rule model
// carries these fields, so one set of helpers maps them for all rule types.
type ruleExtraConditions struct {
	Asn, Ua, TrafficSource       *types.String
	AsnIs, UaIs, TrafficSourceIs *types.List
	AsnIsNot, UaIsNot            *types.List
	TrafficSourceIsNot           *types.List
	CookieName                   *types.String
}

// ruleExtraConditionsRequest is the part of every V2Rule*Request that sets
// the extra conditions.
type ruleExtraConditionsRequest interface {
	SetAsn(string)
	SetAsnIs([]string)
	SetAsnIsNot([]string)
	SetUa(string)
	SetUaIs([]string)
	SetUaIsNot([]string)
	SetTrafficSource(string)
	SetTrafficSourceIs([]string)
	SetTrafficSourceIsNot([]string)
	SetCookieName(string)
}

// ruleExtraConditionsResponse is the part of every V2Rule* response that
// returns the extra conditions. The API returns the cookie name as
// only_with_cookie; it never returns cookie_name.
type ruleExtraConditionsResponse interface {
	GetAsn() string
	GetAsnIs() []string
	GetAsnIsNot() []string
	GetUa() string
	GetUaIs() []string
	GetUaIsNot() []string
	GetTrafficSource() string
	GetTrafficSourceIs() []string
	GetTrafficSourceIsNot() []string
	GetOnlyWithCookie() string
}

// buildRuleExtraConditions sends the known asn, ua, traffic_source and
// cookie_name values. A null or unknown selector sends nothing for its
// triplet. An empty cookie_name is sent, because it clears the cookie.
func buildRuleExtraConditions(ctx context.Context, c ruleExtraConditions, req ruleExtraConditionsRequest) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.Append(buildConditionalList(ctx, *c.Asn, *c.AsnIs, *c.AsnIsNot,
		req.SetAsn, req.SetAsnIs, req.SetAsnIsNot)...)
	diags.Append(buildConditionalList(ctx, *c.Ua, *c.UaIs, *c.UaIsNot,
		req.SetUa, req.SetUaIs, req.SetUaIsNot)...)
	diags.Append(buildConditionalList(ctx, *c.TrafficSource, *c.TrafficSourceIs, *c.TrafficSourceIsNot,
		req.SetTrafficSource, req.SetTrafficSourceIs, req.SetTrafficSourceIsNot)...)
	if isKnown(*c.CookieName) {
		req.SetCookieName(c.CookieName.ValueString())
	}
	return diags
}

// setRuleExtraConditionsFromAPI reads the extra conditions back from an API
// response. An absent selector reads back as null, with empty lists, the same
// as the country, ip and method triplets.
func setRuleExtraConditionsFromAPI(ctx context.Context, api ruleExtraConditionsResponse, c ruleExtraConditions) diag.Diagnostics {
	var diags diag.Diagnostics
	asn, ua, ts := api.GetAsn(), api.GetUa(), api.GetTrafficSource()
	diags.Append(setConditionalList(ctx, &asn, api.GetAsnIs(), api.GetAsnIsNot(),
		c.Asn, c.AsnIs, c.AsnIsNot)...)
	diags.Append(setConditionalList(ctx, &ua, api.GetUaIs(), api.GetUaIsNot(),
		c.Ua, c.UaIs, c.UaIsNot)...)
	diags.Append(setConditionalList(ctx, &ts, api.GetTrafficSourceIs(), api.GetTrafficSourceIsNot(),
		c.TrafficSource, c.TrafficSourceIs, c.TrafficSourceIsNot)...)
	*c.CookieName = cookieNameFromAPI(api.GetOnlyWithCookie(), *c.CookieName)
	return diags
}

// cookieNameFromAPI returns the write-only cookie_name from the
// only_with_cookie value the API returns. An empty value reads back as null,
// except when the prior value is an explicit empty string (the clear request),
// which is kept so that the apply result matches the plan.
func cookieNameFromAPI(onlyWithCookie string, prior types.String) types.String {
	if onlyWithCookie != "" {
		return types.StringValue(onlyWithCookie)
	}
	if isKnown(prior) && prior.ValueString() == "" {
		return prior
	}
	return types.StringNull()
}

func authExtraConditions(m *resource_rule_auth.RuleAuthModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func botChallengeExtraConditions(m *resource_rule_bot_challenge.RuleBotChallengeModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func contentFilterExtraConditions(m *resource_rule_content_filter.RuleContentFilterModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func customResponseExtraConditions(m *resource_rule_custom_response.RuleCustomResponseModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func functionExtraConditions(m *resource_rule_function.RuleFunctionModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func headersExtraConditions(m *resource_rule_headers.RuleHeadersModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func proxyExtraConditions(m *resource_rule_proxy.RuleProxyModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func redirectExtraConditions(m *resource_rule_redirect.RuleRedirectModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}

func serveStaticExtraConditions(m *resource_rule_serve_static.RuleServeStaticModel) ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TrafficSourceIs, TrafficSourceIsNot: &m.TrafficSourceIsNot,
		CookieName: &m.CookieName,
	}
}
