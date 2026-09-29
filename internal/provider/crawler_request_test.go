package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_crawler"
)

// crawlerModelWithAssets builds the model the framework produces for a plan
// that sets the assets block. The nested values are plain basetypes.ObjectValue
// because AssetsValue stores its attributes as basetypes.ObjectValue.
func crawlerModelWithAssets(ctx context.Context) *resource_crawler.CrawlerModel {
	niAttrTypes := resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx)
	parserAttrTypes := resource_crawler.ParserValue{}.AttributeTypes(ctx)

	networkIntercept := types.ObjectValueMust(niAttrTypes, map[string]attr.Value{
		"enabled":    types.BoolValue(true),
		"execute_js": types.BoolValue(false),
		"timeout":    types.Int64Value(30),
	})
	parser := types.ObjectValueMust(parserAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(true),
	})

	return &resource_crawler.CrawlerModel{
		Name:    types.StringValue("test"),
		Domain:  types.StringValue("https://example.com"),
		Project: types.StringValue("default"),
		Assets: resource_crawler.NewAssetsValueMust(
			resource_crawler.AssetsValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"network_intercept": networkIntercept,
				"parser":            parser,
			},
		),
	}
}

// TestBuildCrawlerRequest_Assets reproduces the production failure: reading the
// network_intercept object into the generated NetworkInterceptValue raises a
// framework Value Conversion Error, because the object's type is a plain
// basetypes.ObjectType whose ValueType is basetypes.ObjectValue.
func TestBuildCrawlerRequest_Assets(t *testing.T) {
	ctx := context.Background()

	req, diags := buildCrawlerRequest(ctx, crawlerModelWithAssets(ctx))

	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}
	if len(diags.Warnings()) != 0 {
		t.Errorf("unexpected warning diagnostics: %v", diags.Warnings())
	}

	assets, ok := req.GetAssetsOk()
	if !ok || assets == nil {
		t.Fatal("assets were not set on the request")
	}

	ni, ok := assets.GetNetworkInterceptOk()
	if !ok || ni == nil {
		t.Fatal("network_intercept was not set on the request")
	}
	if got := ni.GetEnabled(); got != true {
		t.Errorf("network_intercept.enabled = %v, want true", got)
	}
	if got := ni.GetExecuteJs(); got != false {
		t.Errorf("network_intercept.execute_js = %v, want false", got)
	}
	if got := ni.GetTimeout(); got != 30 {
		t.Errorf("network_intercept.timeout = %d, want 30", got)
	}

	parser, ok := assets.GetParserOk()
	if !ok || parser == nil {
		t.Fatal("parser was not set on the request")
	}
	if got := parser.GetEnabled(); got != true {
		t.Errorf("parser.enabled = %v, want true", got)
	}
}

// TestBuildCrawlerRequest_AssetsParserOnly covers an assets block that sets the
// parser but leaves network_intercept null.
func TestBuildCrawlerRequest_AssetsParserOnly(t *testing.T) {
	ctx := context.Background()

	crawler := crawlerModelWithAssets(ctx)
	crawler.Assets = resource_crawler.NewAssetsValueMust(
		resource_crawler.AssetsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"network_intercept": types.ObjectNull(resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx)),
			"parser": types.ObjectValueMust(
				resource_crawler.ParserValue{}.AttributeTypes(ctx),
				map[string]attr.Value{"enabled": types.BoolValue(true)},
			),
		},
	)

	req, diags := buildCrawlerRequest(ctx, crawler)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}

	assets, ok := req.GetAssetsOk()
	if !ok || assets == nil {
		t.Fatal("assets were not set on the request")
	}
	if _, ok := assets.GetNetworkInterceptOk(); ok {
		t.Error("network_intercept must stay unset when it is null in the plan")
	}
	if parser, ok := assets.GetParserOk(); !ok || parser.GetEnabled() != true {
		t.Error("parser.enabled was not set on the request")
	}
}

// TestBuildCrawlerRequest_NoAssets pins the behaviour of the overwhelming
// majority of managed crawlers: no assets block, no assets on the request and
// no diagnostics.
func TestBuildCrawlerRequest_NoAssets(t *testing.T) {
	ctx := context.Background()

	crawler := &resource_crawler.CrawlerModel{
		Name:    types.StringValue("test"),
		Domain:  types.StringValue("https://example.com"),
		Project: types.StringValue("default"),
		Assets:  resource_crawler.NewAssetsValueNull(),
	}

	req, diags := buildCrawlerRequest(ctx, crawler)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}
	if len(diags.Warnings()) != 0 {
		t.Errorf("unexpected warning diagnostics: %v", diags.Warnings())
	}
	if _, ok := req.GetAssetsOk(); ok {
		t.Error("assets must stay unset when the crawler configures none")
	}
	if got := req.GetName(); got != "test" {
		t.Errorf("name = %q, want %q", got, "test")
	}
	if got := req.GetDomain(); got != "https://example.com" {
		t.Errorf("domain = %q, want %q", got, "https://example.com")
	}
}

// TestBuildCrawlerRequest_SitemapAndStatusOkReachTheSDK proves the two
// "field skipped" warnings from mapper.ToSDK are benign: buildCrawlerRequest
// sets both fields itself, after the mapper pass.
func TestBuildCrawlerRequest_SitemapAndStatusOkReachTheSDK(t *testing.T) {
	ctx := context.Background()

	sitemapElemType := resource_crawler.SitemapType{
		ObjectType: types.ObjectType{
			AttrTypes: resource_crawler.SitemapValue{}.AttributeTypes(ctx),
		},
	}

	crawler := &resource_crawler.CrawlerModel{
		Name:    types.StringValue("test"),
		Domain:  types.StringValue("https://example.com"),
		Project: types.StringValue("default"),
		Assets:  resource_crawler.NewAssetsValueNull(),
		StatusOk: types.ListValueMust(types.Int64Type, []attr.Value{
			types.Int64Value(200),
			types.Int64Value(301),
		}),
		Sitemap: types.ListValueMust(sitemapElemType, []attr.Value{
			resource_crawler.NewSitemapValueMust(
				resource_crawler.SitemapValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"url":       types.StringValue("https://example.com/sitemap.xml"),
					"recursive": types.BoolValue(true),
				},
			),
		}),
	}

	req, diags := buildCrawlerRequest(ctx, crawler)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}
	if len(diags.Warnings()) != 0 {
		t.Errorf("unexpected warning diagnostics: %v", diags.Warnings())
	}

	statusOk := req.GetStatusOk()
	if len(statusOk) != 2 || statusOk[0] != 200 || statusOk[1] != 301 {
		t.Errorf("status_ok = %v, want [200 301]", statusOk)
	}

	sitemap := req.GetSitemap()
	if len(sitemap) != 1 {
		t.Fatalf("sitemap length = %d, want 1", len(sitemap))
	}
	if got := sitemap[0].GetUrl(); got != "https://example.com/sitemap.xml" {
		t.Errorf("sitemap[0].url = %q", got)
	}
	if got := sitemap[0].GetRecursive(); got != true {
		t.Errorf("sitemap[0].recursive = %v, want true", got)
	}
}

// TestCrawlerAssetsFromConfig_ParserRoundTrips pins the read path: a parser
// block reported by the API must land back in state. Without it the value
// stays null and Terraform reports an inconsistent result after apply.
func TestCrawlerAssetsFromConfig_ParserRoundTrips(t *testing.T) {
	ctx := context.Background()

	var cfg CrawlerConfig
	cfg.Config.Assets.NetworkIntercept.Enabled = true
	cfg.Config.Assets.NetworkIntercept.Timeout = 30
	cfg.Config.Assets.Parser.Enabled = true

	got := crawlerAssetsFromConfig(ctx, &cfg, resource_crawler.NewAssetsValueNull())

	if got.IsNull() || got.IsUnknown() {
		t.Fatal("assets must be known when the config reports them")
	}
	if got.Parser.IsNull() || got.Parser.IsUnknown() {
		t.Fatal("parser must be known when the config enables it")
	}
	enabled, ok := got.Parser.Attributes()["enabled"].(types.Bool)
	if !ok || !enabled.ValueBool() {
		t.Errorf("parser.enabled = %v, want true", got.Parser.Attributes()["enabled"])
	}
	ni := got.NetworkIntercept.Attributes()
	if v, _ := ni["enabled"].(types.Bool); !v.ValueBool() {
		t.Error("network_intercept.enabled = false, want true")
	}
	if v, _ := ni["timeout"].(types.Int64); v.ValueInt64() != 30 {
		t.Errorf("network_intercept.timeout = %v, want 30", ni["timeout"])
	}
}

// TestCrawlerAssetsFromConfig_NoAssets pins the no-assets case: the value stays
// null and no assets object appears in state.
func TestCrawlerAssetsFromConfig_NoAssets(t *testing.T) {
	ctx := context.Background()

	var cfg CrawlerConfig
	got := crawlerAssetsFromConfig(ctx, &cfg, resource_crawler.NewAssetsValueNull())

	if !got.IsNull() {
		t.Errorf("assets = %v, want null", got)
	}
}

// crawlerConfigYAMLWithEmptyLists is the shape the API stores for a crawler
// whose start_url, include and allowed_domains lists are empty. The backend
// writes an empty mapping, not an empty sequence.
const crawlerConfigYAMLWithEmptyLists = `config:
    user_agent: 'test-agent'
    browser_mode: true
    workers: 3
    depth: -1
    delay: 4
    status_ok: [200]
    start_url: {  }
    include: {  }
    allowed_domains: {  }
    exclude:
        - /admin
    sitemap:
        - recursive: true
          url: /sitemap.xml
    headers: {  }
domain: 'https://example.com'
headers: {  }
`

// TestParseCrawlerConfig_EmptyListsAreMappings covers the config the API
// actually stores. The empty mappings must not discard the rest of the parse:
// every other field has to reach the model, and no warning may be raised.
func TestParseCrawlerConfig_EmptyListsAreMappings(t *testing.T) {
	ctx := context.Background()

	crawler := &resource_crawler.CrawlerModel{Assets: resource_crawler.NewAssetsValueNull()}
	api := &quantadmingo.V2Crawler{}

	diags := parseCrawlerConfig(ctx, crawlerConfigYAMLWithEmptyLists, crawler, api)

	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}
	if len(diags.Warnings()) != 0 {
		t.Errorf("unexpected warning diagnostics: %v", diags.Warnings())
	}

	// Fields that share the config blob with the empty lists.
	if !crawler.BrowserMode.ValueBool() {
		t.Error("browser_mode was not read back")
	}
	if got := crawler.Workers.ValueInt64(); got != 3 {
		t.Errorf("workers = %d, want 3", got)
	}
	if got := crawler.UserAgent.ValueString(); got != "test-agent" {
		t.Errorf("user_agent = %q, want %q", got, "test-agent")
	}

	// The control: a non-empty list already round-tripped before this fix.
	if got := len(crawler.Exclude.Elements()); got != 1 {
		t.Errorf("exclude length = %d, want 1", got)
	}

	// The values the empty mappings used to take down with them.
	if got := len(crawler.StatusOk.Elements()); got != 1 {
		t.Errorf("status_ok length = %d, want 1", got)
	}
	if got := len(crawler.Sitemap.Elements()); got != 1 {
		t.Errorf("sitemap length = %d, want 1", got)
	}

	// The empty lists themselves must not look like configured values.
	if !crawler.Include.IsNull() && len(crawler.Include.Elements()) != 0 {
		t.Errorf("include = %v, want empty or null", crawler.Include)
	}
	if !crawler.AllowedDomains.IsNull() && len(crawler.AllowedDomains.Elements()) != 0 {
		t.Errorf("allowed_domains = %v, want empty or null", crawler.AllowedDomains)
	}
}

// BLOCKER A. The config the API stores reports no start_url, include or
// allowed_domains. A planned value for those, and for urls and headers, must
// survive the read; the parse must not replace it with an empty collection.
func TestParseCrawlerConfig_PlannedListValuesSurvive(t *testing.T) {
	ctx := context.Background()

	crawler := &resource_crawler.CrawlerModel{
		Assets: resource_crawler.NewAssetsValueNull(),
		StartUrls: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("https://example.com/a"),
		}),
		Urls: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("https://example.com/b"),
		}),
		Include: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("/keep"),
		}),
		AllowedDomains: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("example.org"),
		}),
		Headers: types.MapValueMust(types.StringType, map[string]attr.Value{
			"x-token": types.StringValue("secret"),
		}),
	}

	diags := parseCrawlerConfig(ctx, crawlerConfigYAMLWithEmptyLists, crawler, &quantadmingo.V2Crawler{})
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}

	assertStringList(t, "start_urls", crawler.StartUrls, "https://example.com/a")
	assertStringList(t, "urls", crawler.Urls, "https://example.com/b")
	assertStringList(t, "include", crawler.Include, "/keep")
	assertStringList(t, "allowed_domains", crawler.AllowedDomains, "example.org")

	if got := len(crawler.Headers.Elements()); got != 1 {
		t.Errorf("headers were wiped: %v", crawler.Headers)
	}
}

func assertStringList(t *testing.T, name string, list types.List, want string) {
	t.Helper()
	elems := list.Elements()
	if len(elems) != 1 {
		t.Errorf("%s = %v, want one element %q", name, list, want)
		return
	}
	if got, _ := elems[0].(types.String); got.ValueString() != want {
		t.Errorf("%s[0] = %v, want %q", name, elems[0], want)
	}
}

// BLOCKER A. A planned null must come back null, not an empty collection. An
// unknown value must still resolve, because the framework forbids leaving one
// unknown after apply.
func TestParseCrawlerConfig_NullStaysNullUnknownResolves(t *testing.T) {
	ctx := context.Background()

	crawler := &resource_crawler.CrawlerModel{
		Assets:    resource_crawler.NewAssetsValueNull(),
		StartUrls: types.ListNull(types.StringType),
		Include:   types.ListUnknown(types.StringType),
	}

	diags := parseCrawlerConfig(ctx, crawlerConfigYAMLWithEmptyLists, crawler, &quantadmingo.V2Crawler{})
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}

	if !crawler.StartUrls.IsNull() {
		t.Errorf("start_urls = %v, want null", crawler.StartUrls)
	}
	if crawler.Include.IsUnknown() {
		t.Error("include must not stay unknown after the read")
	}
	if crawler.Include.IsNull() || len(crawler.Include.Elements()) != 0 {
		t.Errorf("include = %v, want an empty list", crawler.Include)
	}
}

// BLOCKER B. The backend echoes network_intercept but not assets.parser. The
// planned parser must be preserved, or Terraform reports an inconsistent
// result and the Pulumi bridge shows a perpetual diff.
func TestCrawlerAssetsFromConfig_PreservesPlannedParserWhenNotEchoed(t *testing.T) {
	ctx := context.Background()

	var cfg CrawlerConfig
	cfg.Config.Assets.NetworkIntercept.Enabled = true
	cfg.Config.Assets.NetworkIntercept.Timeout = 30
	// The backend does not echo assets.parser.

	planned := resource_crawler.NewAssetsValueMust(
		resource_crawler.AssetsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"network_intercept": types.ObjectNull(resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx)),
			"parser": types.ObjectValueMust(
				resource_crawler.ParserValue{}.AttributeTypes(ctx),
				map[string]attr.Value{"enabled": types.BoolValue(true)},
			),
		},
	)

	got := crawlerAssetsFromConfig(ctx, &cfg, planned)

	if got.Parser.IsNull() || got.Parser.IsUnknown() {
		t.Fatalf("planned parser was dropped: %v", got.Parser)
	}
	if v, _ := got.Parser.Attributes()["enabled"].(types.Bool); !v.ValueBool() {
		t.Errorf("parser.enabled = %v, want true", got.Parser.Attributes()["enabled"])
	}
	if got.NetworkIntercept.IsNull() {
		t.Error("network_intercept from the config was dropped")
	}
}

// BLOCKER B. execute_js alone marks the network_intercept block as present.
func TestCrawlerAssetsFromConfig_ExecuteJsAloneCounts(t *testing.T) {
	ctx := context.Background()

	var cfg CrawlerConfig
	cfg.Config.Assets.NetworkIntercept.ExecuteJs = true

	got := crawlerAssetsFromConfig(ctx, &cfg, resource_crawler.NewAssetsValueNull())

	if got.IsNull() {
		t.Fatal("assets = null, want the network_intercept block")
	}
	if v, _ := got.NetworkIntercept.Attributes()["execute_js"].(types.Bool); !v.ValueBool() {
		t.Errorf("execute_js = %v, want true", got.NetworkIntercept.Attributes()["execute_js"])
	}
}

// BLOCKER C. A document-level type error leaves the whole Config struct at its
// zero value. The provider must warn and leave state alone rather than
// overwrite it with defaults.
func TestParseCrawlerConfig_DocumentLevelTypeErrorIsNotApplied(t *testing.T) {
	ctx := context.Background()

	crawler := &resource_crawler.CrawlerModel{
		Assets:      resource_crawler.NewAssetsValueNull(),
		BrowserMode: types.BoolValue(true),
		Workers:     types.Int64Value(3),
		StartUrls: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("https://example.com/a"),
		}),
	}

	diags := parseCrawlerConfig(ctx, "config: 'a string not a map'\n", crawler, &quantadmingo.V2Crawler{})

	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}
	if len(diags.Warnings()) == 0 {
		t.Error("expected a warning for an unparseable config")
	}
	if !crawler.BrowserMode.ValueBool() {
		t.Error("browser_mode was overwritten with a default")
	}
	if crawler.Workers.ValueInt64() != 3 {
		t.Errorf("workers = %v, was overwritten with a default", crawler.Workers)
	}
	if len(crawler.StartUrls.Elements()) != 1 {
		t.Errorf("start_urls = %v, was overwritten with a default", crawler.StartUrls)
	}
}
