package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_crawler"
)

// browserConfig builds a KNOWN BrowserConfigValue. A bare struct literal has a
// zero state, which reads as null, so the mapping would correctly skip it.
func browserConfig(attrs map[string]attr.Value) resource_crawler.BrowserConfigValue {
	return resource_crawler.NewBrowserConfigValueMust(
		resource_crawler.BrowserConfigValue{}.AttributeTypes(context.Background()), attrs,
	)
}

// The generated schema gives Terraform the browser_config surface, but the
// request mapping is hand-written: without it the provider accepts the block
// and silently never sends it. These pin that mapping.
func TestSetCrawlerBrowserConfigSendsKnownValues(t *testing.T) {
	crawler := &resource_crawler.CrawlerModel{
		BrowserConfig: browserConfig(map[string]attr.Value{
			"capture_api_responses": basetypes.NewBoolValue(true),
			"use_rendered_html":     basetypes.NewBoolValue(false),
			"wait_for_network_idle": basetypes.NewInt64Value(5000),
		}),
	}
	req := quantadmingo.NewV2CrawlerRequest("https://example.gov.au")
	if diags := setCrawlerBrowserConfig(context.Background(), crawler, req); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	bc := req.GetBrowserConfig()
	if !bc.GetCaptureApiResponses() {
		t.Error("capture_api_responses should be true")
	}
	if bc.GetUseRenderedHtml() {
		t.Error("use_rendered_html should be false")
	}
	if got := bc.GetWaitForNetworkIdle(); got != 5000 {
		t.Errorf("wait_for_network_idle = %d, want 5000", got)
	}
}

// A null block must leave browser_config off the request entirely, so an
// update that does not mention it cannot clear a value set elsewhere.
func TestSetCrawlerBrowserConfigNullSendsNothing(t *testing.T) {
	crawler := &resource_crawler.CrawlerModel{
		BrowserConfig: resource_crawler.NewBrowserConfigValueNull(),
	}
	req := quantadmingo.NewV2CrawlerRequest("https://example.gov.au")
	if diags := setCrawlerBrowserConfig(context.Background(), crawler, req); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if req.BrowserConfig != nil {
		t.Error("browser_config should be absent when the block is null")
	}
}

// Only the attributes actually set should be sent.
func TestSetCrawlerBrowserConfigPartial(t *testing.T) {
	crawler := &resource_crawler.CrawlerModel{
		BrowserConfig: browserConfig(map[string]attr.Value{
			"capture_api_responses": basetypes.NewBoolValue(true),
			"use_rendered_html":     basetypes.NewBoolNull(),
			"wait_for_network_idle": basetypes.NewInt64Null(),
		}),
	}
	req := quantadmingo.NewV2CrawlerRequest("https://example.gov.au")
	setCrawlerBrowserConfig(context.Background(), crawler, req)
	bc := req.GetBrowserConfig()
	if !bc.GetCaptureApiResponses() {
		t.Error("capture_api_responses should be true")
	}
	if bc.WaitForNetworkIdle != nil {
		t.Error("wait_for_network_idle should be absent when null")
	}
	if bc.UseRenderedHtml != nil {
		t.Error("use_rendered_html should be absent when null")
	}
}

// Drift must be visible. An earlier version returned the prior state when the
// API did not report browser_config, which hid a real failure: the production
// API accepted the field, silently dropped it, and `pulumi preview --refresh`
// reported 1 change when 116 crawlers were actually missing it.
func TestBrowserConfigReadBackReportsAbsenceAsNull(t *testing.T) {
	ctx := context.Background()
	var cfg CrawlerConfig // no browser_config echoed by the API

	prior := browserConfig(map[string]attr.Value{
		"capture_api_responses": basetypes.NewBoolValue(true),
		"use_rendered_html":     basetypes.NewBoolNull(),
		"wait_for_network_idle": basetypes.NewInt64Null(),
	})

	got := crawlerBrowserConfigFromConfig(ctx, &cfg, prior)
	if !got.IsNull() {
		t.Fatal("an absent browser_config must read back as null so drift is detectable")
	}
}

// When the API does report it, state reflects what was stored.
func TestBrowserConfigReadBackReflectsStoredValues(t *testing.T) {
	ctx := context.Background()
	var cfg CrawlerConfig
	cfg.Config.BrowserConfig.CaptureApiResponses = true
	cfg.Config.BrowserConfig.WaitForNetworkIdle = 5000

	got := crawlerBrowserConfigFromConfig(ctx, &cfg, resource_crawler.NewBrowserConfigValueNull())
	if got.IsNull() || got.IsUnknown() {
		t.Fatal("a reported browser_config must read back as known")
	}
	if !got.CaptureApiResponses.ValueBool() {
		t.Error("capture_api_responses should be true")
	}
	if got.WaitForNetworkIdle.ValueInt64() != 5000 {
		t.Errorf("wait_for_network_idle = %d, want 5000", got.WaitForNetworkIdle.ValueInt64())
	}
}
