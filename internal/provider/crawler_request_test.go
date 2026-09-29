package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
