package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
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

// testModelConditions is a standalone set of model fields for the helper tests.
type testModelConditions struct {
	Asn, Ua, TrafficSource, CookieName            types.String
	AsnIs, AsnIsNot, UaIs, UaIsNot, TsIs, TsIsNot types.List
}

func (m *testModelConditions) refs() ruleExtraConditions {
	return ruleExtraConditions{
		Asn: &m.Asn, AsnIs: &m.AsnIs, AsnIsNot: &m.AsnIsNot,
		Ua: &m.Ua, UaIs: &m.UaIs, UaIsNot: &m.UaIsNot,
		TrafficSource: &m.TrafficSource, TrafficSourceIs: &m.TsIs, TrafficSourceIsNot: &m.TsIsNot,
		CookieName: &m.CookieName,
	}
}

func nullTestConditions() *testModelConditions {
	nl := types.ListNull(types.StringType)
	return &testModelConditions{
		Asn: types.StringNull(), Ua: types.StringNull(), TrafficSource: types.StringNull(), CookieName: types.StringNull(),
		AsnIs: nl, AsnIsNot: nl, UaIs: nl, UaIsNot: nl, TsIs: nl, TsIsNot: nl,
	}
}

func strList(t *testing.T, vals ...string) types.List {
	t.Helper()
	l, d := types.ListValueFrom(context.Background(), types.StringType, vals)
	if d.HasError() {
		t.Fatalf("list: %v", d)
	}
	return l
}

// fillExtraConditions sets one value of every field family.
func fillExtraConditions(t *testing.T, c ruleExtraConditions) {
	t.Helper()
	*c.Asn = types.StringValue("asn_is")
	*c.AsnIs = strList(t, "15169")
	*c.AsnIsNot = types.ListNull(types.StringType)
	*c.Ua = types.StringValue("ua_is")
	*c.UaIs = strList(t, "Googlebot")
	*c.UaIsNot = types.ListNull(types.StringType)
	*c.TrafficSource = types.StringValue("traffic_source_is_not")
	*c.TrafficSourceIs = types.ListNull(types.StringType)
	*c.TrafficSourceIsNot = strList(t, "crawler")
	*c.CookieName = types.StringValue("session_id")
}

func TestBuildRuleExtraConditions_SendsEveryFamily(t *testing.T) {
	ctx := context.Background()
	m := nullTestConditions()
	fillExtraConditions(t, m.refs())

	req := quantadmingo.NewV2RuleAuthRequestWithDefaults()
	if d := buildRuleExtraConditions(ctx, m.refs(), req); d.HasError() {
		t.Fatalf("diags: %v", d)
	}

	got := map[string]string{
		"asn":            req.GetAsn() + "=" + strings.Join(req.GetAsnIs(), ","),
		"ua":             req.GetUa() + "=" + strings.Join(req.GetUaIs(), ","),
		"traffic_source": req.GetTrafficSource() + "=" + strings.Join(req.GetTrafficSourceIsNot(), ","),
		"cookie_name":    req.GetCookieName(),
	}
	want := map[string]string{
		"asn": "asn_is=15169", "ua": "ua_is=Googlebot",
		"traffic_source": "traffic_source_is_not=crawler", "cookie_name": "session_id",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s not sent: got %q, want %q", k, got[k], v)
		}
	}
	if req.HasUaIsNot() || req.HasTrafficSourceIs() {
		t.Error("null lists must not be sent")
	}
}

func TestBuildRuleExtraConditions_SkipsNullAndUnknown(t *testing.T) {
	ctx := context.Background()
	m := nullTestConditions()
	m.Ua = types.StringUnknown()
	m.UaIs = strList(t, "x")
	m.CookieName = types.StringUnknown()

	req := quantadmingo.NewV2RuleAuthRequestWithDefaults()
	if d := buildRuleExtraConditions(ctx, m.refs(), req); d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if req.HasAsn() || req.HasUa() || req.HasUaIs() || req.HasTrafficSource() || req.HasCookieName() {
		t.Error("null or unknown values must not be sent")
	}
}

func TestBuildRuleExtraConditions_SendsEmptyCookieNameToClear(t *testing.T) {
	m := nullTestConditions()
	m.CookieName = types.StringValue("")
	req := quantadmingo.NewV2RuleAuthRequestWithDefaults()
	buildRuleExtraConditions(context.Background(), m.refs(), req)
	if !req.HasCookieName() || req.GetCookieName() != "" {
		t.Error("an empty cookie_name must be sent to clear the cookie")
	}
}

func TestSetRuleExtraConditionsFromAPI_AbsentReadsNull(t *testing.T) {
	m := nullTestConditions()
	fillExtraConditions(t, m.refs())
	m.Ua = types.StringUnknown()

	api := &quantadmingo.V2RuleAuth{}
	if d := setRuleExtraConditionsFromAPI(context.Background(), api, m.refs()); d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	for name, v := range map[string]types.String{"asn": m.Asn, "ua": m.Ua, "traffic_source": m.TrafficSource, "cookie_name": m.CookieName} {
		if !v.IsNull() {
			t.Errorf("%s: want null, got %s", name, v)
		}
	}
	if m.UaIs.IsUnknown() || len(m.UaIs.Elements()) != 0 {
		t.Errorf("ua_is: want empty known list, got %s", m.UaIs)
	}
}

func TestSetRuleExtraConditionsFromAPI_ReadsValues(t *testing.T) {
	m := nullTestConditions()
	api := &quantadmingo.V2RuleAuth{}
	api.SetAsn("asn_is_not")
	api.SetAsnIsNot([]string{"13335"})
	api.SetUa("ua_is")
	api.SetUaIs([]string{"Googlebot"})
	api.SetTrafficSource("traffic_source_is")
	api.SetTrafficSourceIs([]string{"tor-exit"})
	api.SetOnlyWithCookie("session_id")

	if d := setRuleExtraConditionsFromAPI(context.Background(), api, m.refs()); d.HasError() {
		t.Fatalf("diags: %v", d)
	}
	if m.Asn.ValueString() != "asn_is_not" || len(m.AsnIsNot.Elements()) != 1 {
		t.Errorf("asn: %s %s", m.Asn, m.AsnIsNot)
	}
	if m.Ua.ValueString() != "ua_is" || len(m.UaIs.Elements()) != 1 {
		t.Errorf("ua: %s %s", m.Ua, m.UaIs)
	}
	if m.TrafficSource.ValueString() != "traffic_source_is" || len(m.TsIs.Elements()) != 1 {
		t.Errorf("traffic_source: %s %s", m.TrafficSource, m.TsIs)
	}
	if m.CookieName.ValueString() != "session_id" {
		t.Errorf("cookie_name: %s", m.CookieName)
	}
}

func TestCookieNameFromAPI(t *testing.T) {
	cases := []struct {
		name  string
		api   string
		prior types.String
		want  types.String
	}{
		{"value", "sid", types.StringNull(), types.StringValue("sid")},
		{"drift", "other", types.StringValue("sid"), types.StringValue("other")},
		{"absent", "", types.StringUnknown(), types.StringNull()},
		{"absent after value", "", types.StringValue("sid"), types.StringNull()},
		{"explicit clear kept", "", types.StringValue(""), types.StringValue("")},
	}
	for _, tc := range cases {
		if got := cookieNameFromAPI(tc.api, tc.prior); !got.Equal(tc.want) {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Wiring: each rule resource sends the fields and reads them back.
// ---------------------------------------------------------------------------

// ruleEchoServer stores the last write body and returns it on reads, with the
// required response properties and cookie_name returned as only_with_cookie.
type ruleEchoServer struct {
	mu     sync.Mutex
	last   map[string]any
	stored map[string]any
}

func (s *ruleEchoServer) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Method != http.MethodGet {
		body, _ := io.ReadAll(r.Body)
		s.last = map[string]any{}
		_ = json.Unmarshal(body, &s.last)
		s.stored = map[string]any{}
		for k, v := range s.last {
			s.stored[k] = v
		}
		if cn, ok := s.last["cookie_name"]; ok {
			s.stored["only_with_cookie"] = cn
		}
	}
	resp := map[string]any{
		"uuid": "11111111-1111-4111-a111-111111111111", "rule_id": "r1",
		"disabled": false, "action": "x",
		"action_config": map[string]any{
			"auth_user": "u", "auth_pass": "p", "fn_uuid": "f", "robot_challenge_type": "invisible",
			"custom_response_body": "b", "headers": map[string]any{}, "to": "/", "static_file_path": "/s",
		},
	}
	for k, v := range s.stored {
		if k != "action_config" {
			resp[k] = v
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func newTestRuleClient(t *testing.T) (*client.Client, *ruleEchoServer) {
	t.Helper()
	echo := &ruleEchoServer{}
	srv := httptest.NewServer(http.HandlerFunc(echo.handler))
	t.Cleanup(srv.Close)
	cfg := quantadmingo.NewConfiguration()
	cfg.Servers = quantadmingo.ServerConfigurations{{URL: srv.URL}}
	cfg.HTTPClient = srv.Client()
	return &client.Client{
		AuthContext:  context.Background(),
		Organization: "org",
		Instance:     quantadmingo.NewAPIClient(cfg),
	}, echo
}

type ruleWiringCase struct {
	name   string
	create func(ctx context.Context, c *client.Client, t *testing.T) (ruleExtraConditions, diag.Diagnostics)
	update func(ctx context.Context, c *client.Client) diag.Diagnostics
	// createUnset creates the rule with every extra condition unknown, as the
	// plan holds them when the configuration leaves them out.
	createUnset func(ctx context.Context, c *client.Client) (ruleExtraConditions, diag.Diagnostics)
}

// unknownExtraConditions sets every extra condition to unknown.
func unknownExtraConditions(c ruleExtraConditions) {
	for _, s := range []*types.String{c.Asn, c.Ua, c.TrafficSource, c.CookieName} {
		*s = types.StringUnknown()
	}
	for _, l := range []*types.List{c.AsnIs, c.AsnIsNot, c.UaIs, c.UaIsNot, c.TrafficSourceIs, c.TrafficSourceIsNot} {
		*l = types.ListUnknown(types.StringType)
	}
}

// wiringCase builds a case from a model, its condition refs and the
// resource's create and update calls.
func wiringCase[M any](name string, model *M, refs func(*M) ruleExtraConditions,
	create func(context.Context, *client.Client, *M) diag.Diagnostics,
	update func(context.Context, *client.Client, *M) diag.Diagnostics,
) ruleWiringCase {
	return ruleWiringCase{
		name: name,
		create: func(ctx context.Context, c *client.Client, t *testing.T) (ruleExtraConditions, diag.Diagnostics) {
			setTestProject(model)
			fillExtraConditions(t, refs(model))
			return refs(model), create(ctx, c, model)
		},
		update: func(ctx context.Context, c *client.Client) diag.Diagnostics {
			ec := refs(model)
			*ec.Ua = types.StringValue("ua_is_not")
			*ec.UaIsNot = types.ListValueMust(types.StringType, nil)
			return update(ctx, c, model)
		},
		createUnset: func(ctx context.Context, c *client.Client) (ruleExtraConditions, diag.Diagnostics) {
			var fresh M
			setTestProject(&fresh)
			unknownExtraConditions(refs(&fresh))
			return refs(&fresh), create(ctx, c, &fresh)
		},
	}
}

// setTestProject sets the Project field that every rule model has.
func setTestProject(model any) {
	reflect.ValueOf(model).Elem().FieldByName("Project").Set(reflect.ValueOf(types.StringValue("proj")))
}

func ruleWiringCases() []ruleWiringCase {
	auth := &resource_rule_auth.RuleAuthModel{}
	bot := &resource_rule_bot_challenge.RuleBotChallengeModel{}
	cf := &resource_rule_content_filter.RuleContentFilterModel{}
	cr := &resource_rule_custom_response.RuleCustomResponseModel{}
	fn := &resource_rule_function.RuleFunctionModel{}
	hd := &resource_rule_headers.RuleHeadersModel{}
	px := &resource_rule_proxy.RuleProxyModel{}
	rd := &resource_rule_redirect.RuleRedirectModel{}
	ss := &resource_rule_serve_static.RuleServeStaticModel{}
	return []ruleWiringCase{
		wiringCase("auth", auth, authExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_auth.RuleAuthModel) diag.Diagnostics {
				return callRuleAuthCreateAPI(ctx, &ruleAuthResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_auth.RuleAuthModel) diag.Diagnostics {
				return callRuleAuthUpdateAPI(ctx, &ruleAuthResource{client: c}, m)
			}),
		wiringCase("bot_challenge", bot, botChallengeExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_bot_challenge.RuleBotChallengeModel) diag.Diagnostics {
				return callRuleBotChallengeCreateAPI(ctx, &ruleBotChallengeResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_bot_challenge.RuleBotChallengeModel) diag.Diagnostics {
				return callRuleBotChallengeUpdateAPI(ctx, &ruleBotChallengeResource{client: c}, m)
			}),
		wiringCase("content_filter", cf, contentFilterExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_content_filter.RuleContentFilterModel) diag.Diagnostics {
				return callRuleContentFilterCreateAPI(ctx, &ruleContentFilterResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_content_filter.RuleContentFilterModel) diag.Diagnostics {
				return callRuleContentFilterUpdateAPI(ctx, &ruleContentFilterResource{client: c}, m)
			}),
		wiringCase("custom_response", cr, customResponseExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_custom_response.RuleCustomResponseModel) diag.Diagnostics {
				return callRuleCustomResponseCreateAPI(ctx, &ruleCustomResponseResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_custom_response.RuleCustomResponseModel) diag.Diagnostics {
				return callRuleCustomResponseUpdateAPI(ctx, &ruleCustomResponseResource{client: c}, m)
			}),
		wiringCase("function", fn, functionExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_function.RuleFunctionModel) diag.Diagnostics {
				return callRuleFunctionCreateAPI(ctx, &ruleFunctionResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_function.RuleFunctionModel) diag.Diagnostics {
				return callRuleFunctionUpdateAPI(ctx, &ruleFunctionResource{client: c}, m)
			}),
		wiringCase("headers", hd, headersExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_headers.RuleHeadersModel) diag.Diagnostics {
				return callRuleHeadersCreateAPI(ctx, &ruleHeadersResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_headers.RuleHeadersModel) diag.Diagnostics {
				return callRuleHeadersUpdateAPI(ctx, &ruleHeadersResource{client: c}, m)
			}),
		wiringCase("proxy", px, proxyExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_proxy.RuleProxyModel) diag.Diagnostics {
				return callRuleProxyCreateAPI(ctx, &ruleProxyResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_proxy.RuleProxyModel) diag.Diagnostics {
				return callRuleProxyUpdateAPI(ctx, &ruleProxyResource{client: c}, m)
			}),
		wiringCase("redirect", rd, redirectExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_redirect.RuleRedirectModel) diag.Diagnostics {
				return callRuleRedirectCreateAPI(ctx, &ruleRedirectResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_redirect.RuleRedirectModel) diag.Diagnostics {
				return callRuleRedirectUpdateAPI(ctx, &ruleRedirectResource{client: c}, m)
			}),
		wiringCase("serve_static", ss, serveStaticExtraConditions,
			func(ctx context.Context, c *client.Client, m *resource_rule_serve_static.RuleServeStaticModel) diag.Diagnostics {
				return callRuleServeStaticCreateAPI(ctx, &ruleServeStaticResource{client: c}, m)
			},
			func(ctx context.Context, c *client.Client, m *resource_rule_serve_static.RuleServeStaticModel) diag.Diagnostics {
				return callRuleServeStaticUpdateAPI(ctx, &ruleServeStaticResource{client: c}, m)
			}),
	}
}

func assertSentConditions(t *testing.T, body map[string]any) {
	t.Helper()
	want := map[string]string{"asn": "asn_is", "ua": "ua_is", "traffic_source": "traffic_source_is_not", "cookie_name": "session_id"}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("request %s: got %v, want %q", k, body[k], v)
		}
	}
	for _, k := range []string{"asn_is", "ua_is", "traffic_source_is_not"} {
		if l, ok := body[k].([]any); !ok || len(l) != 1 {
			t.Errorf("request %s: got %v", k, body[k])
		}
	}
}

func assertReadConditions(t *testing.T, ec ruleExtraConditions) {
	t.Helper()
	if ec.Asn.ValueString() != "asn_is" || ec.Ua.ValueString() != "ua_is" ||
		ec.TrafficSource.ValueString() != "traffic_source_is_not" || ec.CookieName.ValueString() != "session_id" {
		t.Errorf("read back: asn=%s ua=%s traffic_source=%s cookie_name=%s", *ec.Asn, *ec.Ua, *ec.TrafficSource, *ec.CookieName)
	}
	if len(ec.TrafficSourceIsNot.Elements()) != 1 {
		t.Errorf("read back traffic_source_is_not: %s", *ec.TrafficSourceIsNot)
	}
}

func TestRuleResources_ExtraConditionsWiring(t *testing.T) {
	ctx := context.Background()
	for _, tc := range ruleWiringCases() {
		t.Run(tc.name, func(t *testing.T) {
			c, echo := newTestRuleClient(t)
			ec, d := tc.create(ctx, c, t)
			if d.HasError() {
				t.Fatalf("create: %v", d)
			}
			assertSentConditions(t, echo.last)
			assertReadConditions(t, ec)

			if d := tc.update(ctx, c); d.HasError() {
				t.Fatalf("update: %v", d)
			}
			if echo.last["ua"] != "ua_is_not" {
				t.Errorf("update did not send ua: %v", echo.last["ua"])
			}
		})
		t.Run(tc.name+"/absent reads null", func(t *testing.T) {
			c, echo := newTestRuleClient(t)
			ec, d := tc.createUnset(ctx, c)
			if d.HasError() {
				t.Fatalf("create: %v", d)
			}
			for _, k := range []string{"asn", "ua", "traffic_source", "cookie_name"} {
				if _, sent := echo.last[k]; sent {
					t.Errorf("unknown %s must not be sent", k)
				}
			}
			assertNullConditions(t, ec)
		})
	}
}

func assertNullConditions(t *testing.T, ec ruleExtraConditions) {
	t.Helper()
	for name, v := range map[string]*types.String{"asn": ec.Asn, "ua": ec.Ua, "traffic_source": ec.TrafficSource, "cookie_name": ec.CookieName} {
		if !v.IsNull() {
			t.Errorf("%s: want null, got %s", name, *v)
		}
	}
	for name, l := range map[string]*types.List{"ua_is": ec.UaIs, "traffic_source_is_not": ec.TrafficSourceIsNot, "asn_is": ec.AsnIs} {
		if l.IsUnknown() {
			t.Errorf("%s: must be known after apply", name)
		}
	}
}
