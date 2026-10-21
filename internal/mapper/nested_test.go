package mapper

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_crawler"
)

// The nested-object tests use the generated crawler model and the real SDK
// request type, so they exercise the same shapes the resources use: a
// generated *Value object (browser_config), a generated object whose
// sub-objects are plain types.Object (assets), and a list of generated
// objects (sitemap).

// crawlerModel holds the nested attributes of resource_crawler.CrawlerModel.
type crawlerModel struct {
	Name          types.String                        `tfsdk:"name"`
	Assets        resource_crawler.AssetsValue        `tfsdk:"assets"`
	BrowserConfig resource_crawler.BrowserConfigValue `tfsdk:"browser_config"`
	Sitemap       types.List                          `tfsdk:"sitemap"`
}

func sitemapElemType(ctx context.Context) attr.Type {
	return resource_crawler.SitemapType{
		ObjectType: types.ObjectType{AttrTypes: resource_crawler.SitemapValue{}.AttributeTypes(ctx)},
	}
}

func browserConfig(ctx context.Context, capture, rendered attr.Value, idle attr.Value) resource_crawler.BrowserConfigValue {
	return resource_crawler.NewBrowserConfigValueMust(
		resource_crawler.BrowserConfigValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"capture_api_responses": capture,
			"use_rendered_html":     rendered,
			"wait_for_network_idle": idle,
		},
	)
}

func assets(ctx context.Context, enabled, executeJS, timeout, parserEnabled attr.Value) resource_crawler.AssetsValue {
	niTypes := resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx)
	parserTypes := resource_crawler.ParserValue{}.AttributeTypes(ctx)
	return resource_crawler.NewAssetsValueMust(
		resource_crawler.AssetsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"network_intercept": types.ObjectValueMust(niTypes, map[string]attr.Value{
				"enabled":    enabled,
				"execute_js": executeJS,
				"timeout":    timeout,
			}),
			"parser": types.ObjectValueMust(parserTypes, map[string]attr.Value{
				"enabled": parserEnabled,
			}),
		},
	)
}

func sitemapList(ctx context.Context, entries ...[2]attr.Value) types.List {
	elems := make([]attr.Value, len(entries))
	for i, e := range entries {
		elems[i] = resource_crawler.NewSitemapValueMust(
			resource_crawler.SitemapValue{}.AttributeTypes(ctx),
			map[string]attr.Value{"url": e[0], "recursive": e[1]},
		)
	}
	return types.ListValueMust(sitemapElemType(ctx), elems)
}

// emptyCrawlerModel returns a model with every nested attribute null and
// correctly typed, as the framework produces from a plan without them.
func emptyCrawlerModel(ctx context.Context) crawlerModel {
	return crawlerModel{
		Name:          types.StringNull(),
		Assets:        resource_crawler.NewAssetsValueNull(),
		BrowserConfig: resource_crawler.NewBrowserConfigValueNull(),
		Sitemap:       types.ListNull(sitemapElemType(ctx)),
	}
}

func TestToSDK_NestedObjects(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		model func(m *crawlerModel)
		check func(t *testing.T, req *quantadmingo.V2CrawlerRequest)
	}{
		{
			name: "generated object maps known attributes only",
			model: func(m *crawlerModel) {
				m.BrowserConfig = browserConfig(ctx, types.BoolValue(true), types.BoolNull(), types.Int64Value(1500))
			},
			check: func(t *testing.T, req *quantadmingo.V2CrawlerRequest) {
				bc, ok := req.GetBrowserConfigOk()
				if !ok {
					t.Fatal("browser_config not set")
				}
				if got := bc.GetCaptureApiResponses(); !got {
					t.Errorf("capture_api_responses = %v, want true", got)
				}
				if got := bc.GetWaitForNetworkIdle(); got != 1500 {
					t.Errorf("wait_for_network_idle = %d, want 1500", got)
				}
				if bc.HasUseRenderedHtml() {
					t.Error("use_rendered_html is null and must not be set")
				}
			},
		},
		{
			name: "object with plain types.Object sub-objects maps recursively",
			model: func(m *crawlerModel) {
				m.Assets = assets(ctx, types.BoolValue(true), types.BoolUnknown(), types.Int64Value(30), types.BoolValue(false))
			},
			check: func(t *testing.T, req *quantadmingo.V2CrawlerRequest) {
				a, ok := req.GetAssetsOk()
				if !ok {
					t.Fatal("assets not set")
				}
				ni, ok := a.GetNetworkInterceptOk()
				if !ok {
					t.Fatal("assets.network_intercept not set")
				}
				if !ni.GetEnabled() || ni.GetTimeout() != 30 {
					t.Errorf("network_intercept = {enabled:%v timeout:%d}, want {true 30}", ni.GetEnabled(), ni.GetTimeout())
				}
				if ni.HasExecuteJs() {
					t.Error("execute_js is unknown and must not be set")
				}
				p, ok := a.GetParserOk()
				if !ok || !p.HasEnabled() || p.GetEnabled() {
					t.Errorf("parser.enabled must be set to false, got %+v", p)
				}
			},
		},
		{
			name:  "null objects and list are not sent",
			model: func(m *crawlerModel) {},
			check: func(t *testing.T, req *quantadmingo.V2CrawlerRequest) {
				if req.HasAssets() || req.HasBrowserConfig() || req.HasSitemap() {
					t.Errorf("nothing must be set, got assets=%v browser_config=%v sitemap=%v",
						req.HasAssets(), req.HasBrowserConfig(), req.HasSitemap())
				}
			},
		},
		{
			name: "unknown objects and list are not sent",
			model: func(m *crawlerModel) {
				m.Assets = resource_crawler.NewAssetsValueUnknown()
				m.BrowserConfig = resource_crawler.NewBrowserConfigValueUnknown()
				m.Sitemap = types.ListUnknown(sitemapElemType(ctx))
			},
			check: func(t *testing.T, req *quantadmingo.V2CrawlerRequest) {
				if req.HasAssets() || req.HasBrowserConfig() || req.HasSitemap() {
					t.Error("unknown values must not be set")
				}
			},
		},
		{
			name: "list of objects maps to a slice of structs",
			model: func(m *crawlerModel) {
				m.Sitemap = sitemapList(ctx,
					[2]attr.Value{types.StringValue("https://a.example/sitemap.xml"), types.BoolValue(true)},
					[2]attr.Value{types.StringValue("https://b.example/sitemap.xml"), types.BoolNull()},
				)
			},
			check: func(t *testing.T, req *quantadmingo.V2CrawlerRequest) {
				got := req.GetSitemap()
				if len(got) != 2 {
					t.Fatalf("len(sitemap) = %d, want 2", len(got))
				}
				if got[0].GetUrl() != "https://a.example/sitemap.xml" || !got[0].GetRecursive() {
					t.Errorf("sitemap[0] = %+v, want url a.example and recursive true", got[0])
				}
				if got[1].GetUrl() != "https://b.example/sitemap.xml" || got[1].HasRecursive() {
					t.Errorf("sitemap[1] = %+v, want url b.example and recursive unset", got[1])
				}
			},
		},
		{
			name: "empty list maps to an empty slice",
			model: func(m *crawlerModel) {
				m.Sitemap = types.ListValueMust(sitemapElemType(ctx), []attr.Value{})
			},
			check: func(t *testing.T, req *quantadmingo.V2CrawlerRequest) {
				got, ok := req.GetSitemapOk()
				if !ok || got == nil || len(got) != 0 {
					t.Errorf("sitemap = %v (set %v), want empty and set", got, ok)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := emptyCrawlerModel(ctx)
			tt.model(&m)

			req := quantadmingo.NewV2CrawlerRequestWithDefaults()
			diags := ToSDK(ctx, &m, req)
			if diags.HasError() || len(diags.Warnings()) != 0 {
				t.Fatalf("ToSDK() diags = %v, want none", diags)
			}
			tt.check(t, req)
		})
	}
}

func TestToSDK_NestedCallerHandledIsSkipped(t *testing.T) {
	ctx := context.Background()
	m := emptyCrawlerModel(ctx)
	m.BrowserConfig = browserConfig(ctx, types.BoolValue(true), types.BoolNull(), types.Int64Null())

	req := quantadmingo.NewV2CrawlerRequestWithDefaults()
	diags := ToSDK(ctx, &m, req, "browser_config")
	if diags.HasError() {
		t.Fatalf("ToSDK() errors = %v", diags.Errors())
	}
	if req.HasBrowserConfig() {
		t.Error("caller-handled browser_config must not be set")
	}
}

// pointerStructRequest has setters that take a pointer to a struct and a
// slice of struct pointers, which some SDK types use.
type pointerStructRequest struct {
	config *quantadmingo.V2CrawlerBrowserConfig
	items  []*quantadmingo.V2CrawlerSitemapInner
	label  *string
}

func (r *pointerStructRequest) SetConfig(v *quantadmingo.V2CrawlerBrowserConfig) { r.config = v }
func (r *pointerStructRequest) SetItems(v []*quantadmingo.V2CrawlerSitemapInner) { r.items = v }
func (r *pointerStructRequest) SetLabel(v string)                                { r.label = &v }

func TestToSDK_PointerStructTargets(t *testing.T) {
	ctx := context.Background()
	type model struct {
		Config resource_crawler.BrowserConfigValue `tfsdk:"config"`
		Items  types.List                          `tfsdk:"items"`
	}
	m := model{
		Config: browserConfig(ctx, types.BoolNull(), types.BoolValue(true), types.Int64Null()),
		Items:  sitemapList(ctx, [2]attr.Value{types.StringValue("https://a.example/s.xml"), types.BoolValue(false)}),
	}

	req := &pointerStructRequest{}
	diags := ToSDK(ctx, &m, req)
	if diags.HasError() {
		t.Fatalf("ToSDK() errors = %v", diags.Errors())
	}
	if req.config == nil || !req.config.GetUseRenderedHtml() {
		t.Errorf("config = %+v, want use_rendered_html true", req.config)
	}
	if len(req.items) != 1 || req.items[0].GetUrl() != "https://a.example/s.xml" {
		t.Errorf("items = %+v, want one entry for a.example", req.items)
	}
}

func TestToSDK_NestedObjectIntoNonStructWarns(t *testing.T) {
	ctx := context.Background()
	type model struct {
		Label resource_crawler.BrowserConfigValue `tfsdk:"label"`
	}
	m := model{Label: browserConfig(ctx, types.BoolValue(true), types.BoolNull(), types.Int64Null())}

	req := &pointerStructRequest{}
	diags := ToSDK(ctx, &m, req)
	if diags.HasError() {
		t.Fatalf("ToSDK() errors = %v", diags.Errors())
	}
	if len(diags.Warnings()) != 1 {
		t.Errorf("ToSDK() warnings = %v, want 1 type mismatch", diags.Warnings())
	}
	if req.label != nil {
		t.Errorf("label = %q, want unset", *req.label)
	}
}

func TestFromSDK_NestedObjects(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		resp  func() *quantadmingo.V2CrawlerRequest
		prior func(m *crawlerModel)
		check func(t *testing.T, m crawlerModel)
	}{
		{
			name: "absent objects and list read as null",
			resp: func() *quantadmingo.V2CrawlerRequest {
				return quantadmingo.NewV2CrawlerRequestWithDefaults()
			},
			prior: func(m *crawlerModel) {
				m.BrowserConfig = browserConfig(ctx, types.BoolValue(true), types.BoolNull(), types.Int64Null())
			},
			check: func(t *testing.T, m crawlerModel) {
				if !m.BrowserConfig.IsNull() {
					t.Errorf("browser_config = %v, want null", m.BrowserConfig)
				}
				if !m.Assets.IsNull() {
					t.Errorf("assets = %v, want null", m.Assets)
				}
				if !m.Sitemap.IsNull() {
					t.Errorf("sitemap = %v, want null", m.Sitemap)
				}
			},
		},
		{
			name: "generated object reads set attributes and nulls absent ones",
			resp: func() *quantadmingo.V2CrawlerRequest {
				r := quantadmingo.NewV2CrawlerRequestWithDefaults()
				bc := quantadmingo.NewV2CrawlerBrowserConfig()
				bc.SetCaptureApiResponses(true)
				bc.SetWaitForNetworkIdle(2000)
				r.SetBrowserConfig(*bc)
				return r
			},
			prior: func(m *crawlerModel) {},
			check: func(t *testing.T, m crawlerModel) {
				want := browserConfig(ctx, types.BoolValue(true), types.BoolNull(), types.Int64Value(2000))
				if !m.BrowserConfig.Equal(want) {
					t.Errorf("browser_config = %v, want %v", m.BrowserConfig, want)
				}
			},
		},
		{
			name: "object with plain types.Object sub-objects reads recursively",
			resp: func() *quantadmingo.V2CrawlerRequest {
				r := quantadmingo.NewV2CrawlerRequestWithDefaults()
				ni := quantadmingo.NewV2CrawlerAssetsNetworkIntercept()
				ni.SetEnabled(true)
				ni.SetTimeout(45)
				a := quantadmingo.NewV2CrawlerAssets()
				a.SetNetworkIntercept(*ni)
				r.SetAssets(*a)
				return r
			},
			prior: func(m *crawlerModel) {},
			check: func(t *testing.T, m crawlerModel) {
				want := resource_crawler.NewAssetsValueMust(
					resource_crawler.AssetsValue{}.AttributeTypes(ctx),
					map[string]attr.Value{
						"network_intercept": types.ObjectValueMust(
							resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx),
							map[string]attr.Value{
								"enabled":    types.BoolValue(true),
								"execute_js": types.BoolNull(),
								"timeout":    types.Int64Value(45),
							},
						),
						"parser": types.ObjectNull(resource_crawler.ParserValue{}.AttributeTypes(ctx)),
					},
				)
				if !m.Assets.Equal(want) {
					t.Errorf("assets = %v, want %v", m.Assets, want)
				}
			},
		},
		{
			name: "list of objects reads into the prior element type",
			resp: func() *quantadmingo.V2CrawlerRequest {
				r := quantadmingo.NewV2CrawlerRequestWithDefaults()
				s := quantadmingo.NewV2CrawlerSitemapInner()
				s.SetUrl("https://a.example/sitemap.xml")
				s.SetRecursive(true)
				r.SetSitemap([]quantadmingo.V2CrawlerSitemapInner{*s})
				return r
			},
			prior: func(m *crawlerModel) {},
			check: func(t *testing.T, m crawlerModel) {
				want := sitemapList(ctx, [2]attr.Value{types.StringValue("https://a.example/sitemap.xml"), types.BoolValue(true)})
				if !m.Sitemap.Equal(want) {
					t.Errorf("sitemap = %v, want %v", m.Sitemap, want)
				}
			},
		},
		{
			name: "unknown prior object reads without unknown attributes",
			resp: func() *quantadmingo.V2CrawlerRequest {
				r := quantadmingo.NewV2CrawlerRequestWithDefaults()
				bc := quantadmingo.NewV2CrawlerBrowserConfig()
				bc.SetUseRenderedHtml(false)
				r.SetBrowserConfig(*bc)
				return r
			},
			prior: func(m *crawlerModel) {
				m.BrowserConfig = resource_crawler.NewBrowserConfigValueUnknown()
			},
			check: func(t *testing.T, m crawlerModel) {
				want := browserConfig(ctx, types.BoolNull(), types.BoolValue(false), types.Int64Null())
				if !m.BrowserConfig.Equal(want) {
					t.Errorf("browser_config = %v, want %v", m.BrowserConfig, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := emptyCrawlerModel(ctx)
			tt.prior(&m)

			diags := FromSDK(ctx, tt.resp(), &m)
			if diags.HasError() {
				t.Fatalf("FromSDK() errors = %v", diags.Errors())
			}
			tt.check(t, m)
		})
	}
}

func TestFromSDK_NestedCallerHandledIsUntouched(t *testing.T) {
	ctx := context.Background()
	m := emptyCrawlerModel(ctx)
	prior := browserConfig(ctx, types.BoolValue(true), types.BoolNull(), types.Int64Null())
	m.BrowserConfig = prior

	resp := quantadmingo.NewV2CrawlerRequestWithDefaults() // browser_config absent

	diags := FromSDK(ctx, resp, &m, "browser_config")
	if diags.HasError() {
		t.Fatalf("FromSDK() errors = %v", diags.Errors())
	}
	if !m.BrowserConfig.Equal(prior) {
		t.Errorf("browser_config = %v, want unchanged %v", m.BrowserConfig, prior)
	}
}

// TestFromSDK_ZeroObjectFieldIsLeftAlone covers a hand-written model whose
// types.Object field was never initialised: it carries no attribute types,
// so the mapper cannot build a value and must leave the field alone.
func TestFromSDK_ZeroObjectFieldIsLeftAlone(t *testing.T) {
	type model struct {
		BrowserConfig types.Object `tfsdk:"browser_config"`
	}
	resp := quantadmingo.NewV2CrawlerRequestWithDefaults()
	resp.SetBrowserConfig(*quantadmingo.NewV2CrawlerBrowserConfig())

	var m model
	diags := FromSDK(context.Background(), resp, &m)
	if diags.HasError() {
		t.Fatalf("FromSDK() errors = %v", diags.Errors())
	}
	if !m.BrowserConfig.Equal(types.Object{}) {
		t.Errorf("browser_config = %v, want zero value", m.BrowserConfig)
	}
}

// TestNestedRoundTrip sends a configuration through ToSDK and reads it back
// with FromSDK, using the configuration as the prior value. The result must
// equal the configuration, or Terraform would report a diff.
func TestNestedRoundTrip(t *testing.T) {
	ctx := context.Background()

	cfg := emptyCrawlerModel(ctx)
	cfg.Name = types.StringValue("docs")
	cfg.Assets = assets(ctx, types.BoolValue(true), types.BoolValue(false), types.Int64Value(30), types.BoolValue(true))
	cfg.BrowserConfig = browserConfig(ctx, types.BoolValue(true), types.BoolValue(false), types.Int64Value(1500))
	cfg.Sitemap = sitemapList(ctx,
		[2]attr.Value{types.StringValue("https://a.example/sitemap.xml"), types.BoolValue(true)},
		[2]attr.Value{types.StringValue("https://b.example/sitemap.xml"), types.BoolValue(false)},
	)

	req := quantadmingo.NewV2CrawlerRequestWithDefaults()
	if diags := ToSDK(ctx, &cfg, req); diags.HasError() {
		t.Fatalf("ToSDK() errors = %v", diags.Errors())
	}

	got := cfg
	if diags := FromSDK(ctx, req, &got); diags.HasError() {
		t.Fatalf("FromSDK() errors = %v", diags.Errors())
	}

	if !got.Assets.Equal(cfg.Assets) {
		t.Errorf("assets = %v, want %v", got.Assets, cfg.Assets)
	}
	if !got.BrowserConfig.Equal(cfg.BrowserConfig) {
		t.Errorf("browser_config = %v, want %v", got.BrowserConfig, cfg.BrowserConfig)
	}
	if !got.Sitemap.Equal(cfg.Sitemap) {
		t.Errorf("sitemap = %v, want %v", got.Sitemap, cfg.Sitemap)
	}
}

// ---------------------------------------------------------------------------
// JSON-string attributes
// ---------------------------------------------------------------------------

// jsonSDK is a test double with the free-form field shapes the SDK generates
// for schemaless objects.
type jsonSDK struct {
	metadata    map[string]interface{}
	metadataSet bool
	items       []map[string]interface{}
	toolChoice  interface{}
}

func (r *jsonSDK) SetMetadata(v map[string]interface{}) { r.metadata = v; r.metadataSet = true }
func (r *jsonSDK) GetMetadata() map[string]interface{}  { return r.metadata }
func (r *jsonSDK) HasMetadata() bool                    { return r.metadataSet }
func (r *jsonSDK) SetItems(v []map[string]interface{})  { r.items = v }
func (r *jsonSDK) GetItems() []map[string]interface{}   { return r.items }
func (r *jsonSDK) SetToolChoice(v interface{})          { r.toolChoice = v }
func (r *jsonSDK) GetToolChoice() interface{}           { return r.toolChoice }

type jsonModel struct {
	Metadata   types.String `tfsdk:"metadata"`
	Items      types.String `tfsdk:"items"`
	ToolChoice types.String `tfsdk:"tool_choice"`
}

func nullJSONModel() jsonModel {
	return jsonModel{Metadata: types.StringNull(), Items: types.StringNull(), ToolChoice: types.StringNull()}
}

func TestToSDK_JSONString(t *testing.T) {
	tests := []struct {
		name      string
		model     jsonModel
		want      jsonSDK
		wantError bool
	}{
		{
			name:  "object decodes into a map",
			model: jsonModel{Metadata: types.StringValue(`{"b": 1, "a": {"x": [true, null]}}`), Items: types.StringNull(), ToolChoice: types.StringNull()},
			want: jsonSDK{
				metadata:    map[string]interface{}{"a": map[string]interface{}{"x": []interface{}{true, nil}}, "b": float64(1)},
				metadataSet: true,
			},
		},
		{
			name:  "array decodes into a slice of maps",
			model: jsonModel{Metadata: types.StringNull(), Items: types.StringValue(`[{"id": "a"}, {"id": "b"}]`), ToolChoice: types.StringNull()},
			want:  jsonSDK{items: []map[string]interface{}{{"id": "a"}, {"id": "b"}}},
		},
		{
			name:  "scalar JSON decodes into interface{}",
			model: jsonModel{Metadata: types.StringNull(), Items: types.StringNull(), ToolChoice: types.StringValue(`"auto"`)},
			want:  jsonSDK{toolChoice: "auto"},
		},
		{
			name:  "null and unknown are not sent",
			model: jsonModel{Metadata: types.StringNull(), Items: types.StringUnknown(), ToolChoice: types.StringNull()},
			want:  jsonSDK{},
		},
		{
			name:      "invalid JSON is an error",
			model:     jsonModel{Metadata: types.StringValue(`{not json`), Items: types.StringNull(), ToolChoice: types.StringNull()},
			want:      jsonSDK{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &jsonSDK{}
			diags := ToSDK(context.Background(), tt.model, got)
			if diags.HasError() != tt.wantError {
				t.Fatalf("ToSDK() errors = %v, want error %v", diags.Errors(), tt.wantError)
			}
			if tt.wantError && !strings.Contains(diags.Errors()[0].Summary(), `"metadata"`) {
				t.Errorf("error summary = %q, want it to name the field", diags.Errors()[0].Summary())
			}
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("ToSDK() request = %#v, want %#v", *got, tt.want)
			}
		})
	}
}

func TestFromSDK_JSONString(t *testing.T) {
	tests := []struct {
		name  string
		resp  jsonSDK
		prior jsonModel
		want  jsonModel
	}{
		{
			name:  "no prior value reads canonical JSON",
			resp:  jsonSDK{metadata: map[string]interface{}{"b": 1, "a": map[string]interface{}{"y": 2, "x": "<1>"}}, metadataSet: true},
			prior: nullJSONModel(),
			want:  jsonModel{Metadata: types.StringValue(`{"a":{"x":"<1>","y":2},"b":1}`), Items: types.StringNull(), ToolChoice: types.StringNull()},
		},
		{
			name:  "equivalent prior JSON is kept as written",
			resp:  jsonSDK{metadata: map[string]interface{}{"a": 1, "b": []interface{}{"x"}}, metadataSet: true},
			prior: jsonModel{Metadata: types.StringValue("{\n  \"b\": [\"x\"],\n  \"a\": 1.0\n}"), Items: types.StringNull(), ToolChoice: types.StringNull()},
			want:  jsonModel{Metadata: types.StringValue("{\n  \"b\": [\"x\"],\n  \"a\": 1.0\n}"), Items: types.StringNull(), ToolChoice: types.StringNull()},
		},
		{
			name:  "changed content replaces the prior JSON",
			resp:  jsonSDK{metadata: map[string]interface{}{"a": 2}, metadataSet: true},
			prior: jsonModel{Metadata: types.StringValue(`{"a": 1}`), Items: types.StringNull(), ToolChoice: types.StringNull()},
			want:  jsonModel{Metadata: types.StringValue(`{"a":2}`), Items: types.StringNull(), ToolChoice: types.StringNull()},
		},
		{
			name:  "field reported absent reads as null",
			resp:  jsonSDK{metadata: map[string]interface{}{"a": 1}},
			prior: jsonModel{Metadata: types.StringValue(`{"a": 1}`), Items: types.StringNull(), ToolChoice: types.StringNull()},
			want:  nullJSONModel(),
		},
		{
			name:  "nil slice and nil interface read as null",
			resp:  jsonSDK{},
			prior: jsonModel{Metadata: types.StringNull(), Items: types.StringValue(`[]`), ToolChoice: types.StringValue(`"auto"`)},
			want:  nullJSONModel(),
		},
		{
			name:  "slice and scalar values read as JSON",
			resp:  jsonSDK{items: []map[string]interface{}{{"id": "a"}}, toolChoice: "auto"},
			prior: nullJSONModel(),
			want:  jsonModel{Metadata: types.StringNull(), Items: types.StringValue(`[{"id":"a"}]`), ToolChoice: types.StringValue(`"auto"`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prior
			resp := tt.resp
			diags := FromSDK(context.Background(), &resp, &got)
			if diags.HasError() {
				t.Fatalf("FromSDK() errors = %v", diags.Errors())
			}
			if got != tt.want {
				t.Errorf("FromSDK() model = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSONRoundTripKeyOrder sends JSON through ToSDK and back through FromSDK
// with the configuration as the prior value. Key order and whitespace differ
// from the canonical form, and the read must still produce no diff.
func TestJSONRoundTripKeyOrder(t *testing.T) {
	ctx := context.Background()
	cfg := nullJSONModel()
	cfg.Metadata = types.StringValue(`{ "zeta": {"b": 2, "a": 1}, "alpha": [3, 2, 1] }`)

	req := &jsonSDK{}
	if diags := ToSDK(ctx, cfg, req); diags.HasError() {
		t.Fatalf("ToSDK() errors = %v", diags.Errors())
	}

	got := cfg
	if diags := FromSDK(ctx, req, &got); diags.HasError() {
		t.Fatalf("FromSDK() errors = %v", diags.Errors())
	}
	if !got.Metadata.Equal(cfg.Metadata) {
		t.Errorf("metadata = %s, want unchanged %s", got.Metadata, cfg.Metadata)
	}
}

// jsonInObjectSDK nests a free-form field inside an SDK struct.
type jsonInObjectSDK struct {
	Settings *jsonSettings
}

type jsonSettings struct {
	Schema map[string]interface{}
}

func (s *jsonSettings) SetSchema(v map[string]interface{}) { s.Schema = v }
func (s *jsonSettings) GetSchema() map[string]interface{}  { return s.Schema }

func (r *jsonInObjectSDK) SetSettings(v jsonSettings) { r.Settings = &v }
func (r *jsonInObjectSDK) GetSettings() jsonSettings {
	if r.Settings == nil {
		return jsonSettings{}
	}
	return *r.Settings
}
func (r *jsonInObjectSDK) HasSettings() bool { return r.Settings != nil }

// TestJSONInsideNestedObjectRoundTrip covers a JSON string attribute inside a
// nested object, which uses the prior nested value to keep its formatting.
func TestJSONInsideNestedObjectRoundTrip(t *testing.T) {
	ctx := context.Background()
	attrTypes := map[string]attr.Type{"schema": types.StringType}
	type model struct {
		Settings types.Object `tfsdk:"settings"`
	}

	cfg := model{Settings: types.ObjectValueMust(attrTypes, map[string]attr.Value{
		"schema": types.StringValue(`{"type": "object", "required": ["a"]}`),
	})}

	req := &jsonInObjectSDK{}
	if diags := ToSDK(ctx, cfg, req); diags.HasError() {
		t.Fatalf("ToSDK() errors = %v", diags.Errors())
	}
	if req.Settings == nil || req.Settings.Schema["type"] != "object" {
		t.Fatalf("settings = %+v, want schema decoded", req.Settings)
	}

	got := cfg
	if diags := FromSDK(ctx, req, &got); diags.HasError() {
		t.Fatalf("FromSDK() errors = %v", diags.Errors())
	}
	if !got.Settings.Equal(cfg.Settings) {
		t.Errorf("settings = %v, want unchanged %v", got.Settings, cfg.Settings)
	}

	// With no prior value the canonical form is read.
	fresh := model{Settings: types.ObjectNull(attrTypes)}
	if diags := FromSDK(ctx, req, &fresh); diags.HasError() {
		t.Fatalf("FromSDK() errors = %v", diags.Errors())
	}
	want := types.ObjectValueMust(attrTypes, map[string]attr.Value{
		"schema": types.StringValue(`{"required":["a"],"type":"object"}`),
	})
	if !fresh.Settings.Equal(want) {
		t.Errorf("settings = %v, want %v", fresh.Settings, want)
	}
}
