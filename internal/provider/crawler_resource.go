package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_crawler"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"gopkg.in/yaml.v3"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*crawlerResource)(nil)
	_ resource.ResourceWithConfigure   = (*crawlerResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*crawlerResource)(nil)
	_ resource.ResourceWithImportState = (*crawlerResource)(nil)
)

func NewCrawlerResource() resource.Resource {
	return &crawlerResource{}
}

type crawlerResource struct {
	client *client.Client
}

func (r *crawlerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crawler"
}

func (r *crawlerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_crawler.CrawlerResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	resp.Schema = s
}

func (r *crawlerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unepxected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}
	r.client = client
}

func (r *crawlerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_crawler.CrawlerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_crawler.CrawlerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readDiags, gone := stripNotFound(callCrawlerReadAPI(ctx, r, &data))
	resp.Diagnostics.Append(readDiags...)
	if gone {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Start from current state — this ensures all computed and complex fields
	// are resolved. Then apply planned changes on top.
	var data resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	var plan resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Apply user-changeable fields from plan onto the state-initialized model.
	data.Domain = plan.Domain
	data.Name = plan.Name
	data.BrowserMode = plan.BrowserMode
	data.Tracking = plan.Tracking
	data.Workers = plan.Workers
	data.Delay = plan.Delay
	data.Depth = plan.Depth
	data.MaxHits = plan.MaxHits
	data.MaxHtml = plan.MaxHtml
	data.MaxErrors = plan.MaxErrors
	data.UserAgent = plan.UserAgent
	data.WebhookUrl = plan.WebhookUrl
	data.WebhookAuthHeader = plan.WebhookAuthHeader
	data.WebhookExtraVars = plan.WebhookExtraVars
	data.Project = plan.Project
	// List/map fields: use plan values if set, otherwise keep state
	if !plan.Urls.IsUnknown() {
		data.Urls = plan.Urls
	}
	if !plan.StartUrls.IsUnknown() {
		data.StartUrls = plan.StartUrls
	}
	if !plan.Exclude.IsUnknown() {
		data.Exclude = plan.Exclude
	}
	if !plan.Include.IsUnknown() {
		data.Include = plan.Include
	}
	if !plan.AllowedDomains.IsUnknown() {
		data.AllowedDomains = plan.AllowedDomains
	}
	if !plan.Headers.IsUnknown() {
		data.Headers = plan.Headers
	}
	if !plan.Sitemap.IsUnknown() {
		data.Sitemap = plan.Sitemap
	}
	if !plan.StatusOk.IsUnknown() {
		data.StatusOk = plan.StatusOk
	}
	if !plan.Assets.IsUnknown() {
		data.Assets = plan.Assets
	}

	resp.Diagnostics.Append(callCrawlerUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_crawler.CrawlerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerDeleteAPI(ctx, r, &data)...)
}

// ---------------------------------------------------------------------------
// buildCrawlerRequest maps model fields to SDK request using mapper for simple
// fields and manual handling for complex types (maps, int lists, nested objects).
// ---------------------------------------------------------------------------
func buildCrawlerRequest(ctx context.Context, crawler *resource_crawler.CrawlerModel) (*quantadmingo.V2CrawlerRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	req := quantadmingo.NewV2CrawlerRequestWithDefaults()

	// mapper.ToSDK handles: Name, Domain, BrowserMode, WebhookUrl,
	// WebhookAuthHeader, WebhookExtraVars, Workers, Delay, Depth, MaxHits,
	// MaxHtml, MaxErrors, UserAgent, Urls, StartUrls, Exclude, Include,
	// AllowedDomains (all string/bool/int/float/string-list fields).
	// sitemap and status_ok are named as caller-handled: the mapper only maps
	// string lists, and this function sets both fields itself further down.
	diags.Append(mapper.ToSDK(ctx, crawler, req, "sitemap", "status_ok")...)
	if diags.HasError() {
		return nil, diags
	}

	// Tracking — set explicitly rather than through mapper.ToSDK, which skips a
	// field silently when the SDK has no setter for it.
	if !crawler.Tracking.IsNull() && !crawler.Tracking.IsUnknown() {
		req.SetTracking(crawler.Tracking.ValueBool())
	}

	// Headers — map[string]string, not supported by mapper.
	if !crawler.Headers.IsNull() && !crawler.Headers.IsUnknown() {
		headers := make(map[string]string, len(crawler.Headers.Elements()))
		diags.Append(crawler.Headers.ElementsAs(ctx, &headers, false)...)
		if !diags.HasError() {
			req.SetHeaders(headers)
		}
	}

	// StatusOk — []int32, mapper only handles []string lists.
	if !crawler.StatusOk.IsNull() && !crawler.StatusOk.IsUnknown() {
		var statusOkInt64 []int64
		diags.Append(crawler.StatusOk.ElementsAs(ctx, &statusOkInt64, false)...)
		if !diags.HasError() {
			statusOk := make([]int32, len(statusOkInt64))
			for i, v := range statusOkInt64 {
				statusOk[i] = int32(v)
			}
			req.SetStatusOk(statusOk)
		}
	}

	// Sitemap — []V2CrawlerSitemapInner nested objects.
	if !crawler.Sitemap.IsNull() && !crawler.Sitemap.IsUnknown() {
		var sitemapEntries []resource_crawler.SitemapValue
		diags.Append(crawler.Sitemap.ElementsAs(ctx, &sitemapEntries, false)...)
		if !diags.HasError() {
			sitemapArray := make([]quantadmingo.V2CrawlerSitemapInner, len(sitemapEntries))
			for i, entry := range sitemapEntries {
				sitemapItem := quantadmingo.NewV2CrawlerSitemapInner()
				sitemapItem.SetUrl(entry.Url.ValueString())
				sitemapItem.SetRecursive(entry.Recursive.ValueBool())
				sitemapArray[i] = *sitemapItem
			}
			req.SetSitemap(sitemapArray)
		}
	}

	// Assets — nested object with network_intercept and parser sub-objects.
	diags.Append(setCrawlerAssets(ctx, crawler, req)...)

	// Browser config — only meaningful in browser mode.
	diags.Append(setCrawlerBrowserConfig(ctx, crawler, req)...)

	return req, diags
}

// setCrawlerBrowserConfig maps the browser_config block onto the request.
//
// These keys only take effect in browser mode: the crawler reads them on the
// network-interception path, which is installed only when browser_mode is on.
// capture_api_responses is the one that matters — it makes a crawl store
// XHR/fetch responses as files, which is what lets a static copy serve a site
// whose navigation or content is rendered client-side from a JSON endpoint.
// Without it those endpoints 404 on the static copy and the page renders with
// no navigation at all.
//
// Unlike AssetsValue, BrowserConfigValue exposes its attributes directly, so
// no ObjectValue conversion is needed.
func setCrawlerBrowserConfig(ctx context.Context, crawler *resource_crawler.CrawlerModel, req *quantadmingo.V2CrawlerRequest) diag.Diagnostics {
	var diags diag.Diagnostics

	bc := crawler.BrowserConfig
	if bc.IsNull() || bc.IsUnknown() {
		return diags
	}

	out := quantadmingo.NewV2CrawlerBrowserConfig()
	set := false

	if isKnown(bc.CaptureApiResponses) {
		out.SetCaptureApiResponses(bc.CaptureApiResponses.ValueBool())
		set = true
	}
	if isKnown(bc.UseRenderedHtml) {
		out.SetUseRenderedHtml(bc.UseRenderedHtml.ValueBool())
		set = true
	}
	if isKnown(bc.WaitForNetworkIdle) {
		// The schema models this as int64; the API takes int32 milliseconds.
		out.SetWaitForNetworkIdle(int32(bc.WaitForNetworkIdle.ValueInt64()))
		set = true
	}

	if set {
		req.SetBrowserConfig(*out)
	}
	return diags
}

// setCrawlerAssets maps the assets block onto the request. AssetsValue stores
// its sub-objects as plain basetypes.ObjectValue, so the attributes are read
// directly. ObjectValue.As into a generated NetworkInterceptValue raises a
// framework Value Conversion Error and must not be used here.
func setCrawlerAssets(ctx context.Context, crawler *resource_crawler.CrawlerModel, req *quantadmingo.V2CrawlerRequest) diag.Diagnostics {
	var diags diag.Diagnostics

	if crawler.Assets.IsNull() || crawler.Assets.IsUnknown() {
		return diags
	}

	assetsObj := quantadmingo.NewV2CrawlerAssets()
	set := false

	niObj, d := buildNetworkIntercept(ctx, crawler.Assets.NetworkIntercept)
	diags.Append(d...)
	if niObj != nil {
		assetsObj.SetNetworkIntercept(*niObj)
		set = true
	}

	parserObj, d := buildAssetsParser(ctx, crawler.Assets.Parser)
	diags.Append(d...)
	if parserObj != nil {
		assetsObj.SetParser(*parserObj)
		set = true
	}

	if set && !diags.HasError() {
		req.SetAssets(*assetsObj)
	}
	return diags
}

// buildNetworkIntercept converts the network_intercept object. It returns nil
// when the object is absent.
func buildNetworkIntercept(ctx context.Context, obj basetypes.ObjectValue) (*quantadmingo.V2CrawlerAssetsNetworkIntercept, diag.Diagnostics) {
	var diags diag.Diagnostics

	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}

	value, d := resource_crawler.NewNetworkInterceptValue(
		resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx), obj.Attributes(),
	)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	out := quantadmingo.NewV2CrawlerAssetsNetworkIntercept()
	if isKnown(value.Enabled) {
		out.SetEnabled(value.Enabled.ValueBool())
	}
	if isKnown(value.ExecuteJs) {
		out.SetExecuteJs(value.ExecuteJs.ValueBool())
	}
	if isKnown(value.Timeout) {
		out.SetTimeout(int32(value.Timeout.ValueInt64()))
	}
	return out, diags
}

// buildAssetsParser converts the parser object. It returns nil when the object
// is absent.
func buildAssetsParser(ctx context.Context, obj basetypes.ObjectValue) (*quantadmingo.V2CrawlerAssetsParser, diag.Diagnostics) {
	var diags diag.Diagnostics

	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}

	value, d := resource_crawler.NewParserValue(
		resource_crawler.ParserValue{}.AttributeTypes(ctx), obj.Attributes(),
	)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	out := quantadmingo.NewV2CrawlerAssetsParser()
	if isKnown(value.Enabled) {
		out.SetEnabled(value.Enabled.ValueBool())
	}
	return out, diags
}

// isKnown reports whether an attribute carries a usable value.
func isKnown(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------
func callCrawlerCreateAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	req, d := buildCrawlerRequest(ctx, crawler)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	api, httpResp, err := r.client.Instance.CrawlersAPI.CrawlersCreate(
		r.client.AuthContext, r.client.Organization, crawler.Project.ValueString(),
	).V2CrawlerRequest(*req).Execute()

	if err != nil {
		errorMsg := err.Error()
		if httpResp != nil && httpResp.Body != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			var apiError struct {
				Error   bool   `json:"error"`
				Message string `json:"message"`
			}
			if jsonErr := json.Unmarshal(bodyBytes, &apiError); jsonErr == nil && apiError.Message != "" {
				errorMsg = apiError.Message
			}
		}
		diags.AddError("Unable to create crawler", errorMsg)
		return
	}

	crawler.Uuid = types.StringValue(api.GetUuid())
	crawler.Id = types.Int64Value(int64(api.GetId()))

	// Post-create read to populate computed fields.
	return callCrawlerReadAPI(ctx, r, crawler)
}

// ---------------------------------------------------------------------------
// Read — uses mapper.FromSDK for simple fields, then YAML config parsing for
// fields that only appear in the config blob.
// ---------------------------------------------------------------------------
func callCrawlerReadAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	if crawler.Uuid.IsUnknown() || crawler.Uuid.IsNull() {
		diags.AddAttributeError(path.Root("uuid"), "Missing crawler.uuid attribute",
			"To read crawler information, uuid must be provided.")
		return
	}
	if crawler.Project.IsNull() || crawler.Project.IsUnknown() {
		diags.AddAttributeError(path.Root("project"), "Missing crawler.project attribute",
			"To read crawler information, project must be provided.")
		return
	}

	api, httpResp, err := r.client.Instance.CrawlersAPI.CrawlersRead(
		r.client.AuthContext, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.Append(readFailure(httpResp, "Unable to read crawler", fmt.Sprintf("Error: %s", err.Error())))
		return
	}
	if api == nil {
		diags.AddError("Invalid API response", "The API returned a nil response when reading crawler data")
		return
	}

	// Save plan/state values for fields that mapper.FromSDK would overwrite
	// with empty API defaults. These are handled canonically by YAML config
	// parsing or preserved from plan/state.
	savedUrls := crawler.Urls
	savedStartUrls := crawler.StartUrls
	savedExclude := crawler.Exclude
	savedInclude := crawler.Include
	savedAllowedDomains := crawler.AllowedDomains
	savedHeaders := crawler.Headers

	// mapper.FromSDK handles: Id, Name, Domain, DomainVerified, Uuid,
	// ProjectId, BrowserMode, Workers, Delay, Depth, MaxHits, MaxHtml,
	// MaxErrors, UserAgent, Config, UrlsList, WebhookUrl, WebhookAuthHeader,
	// WebhookExtraVars.
	diags.Append(mapper.FromSDK(ctx, api, crawler)...)

	// Restore saved values — YAML config parsing provides the canonical source.
	crawler.Urls = savedUrls
	crawler.StartUrls = savedStartUrls
	crawler.Exclude = savedExclude
	crawler.Include = savedInclude
	crawler.AllowedDomains = savedAllowedDomains
	crawler.Headers = savedHeaders

	// Timestamps — time.Time not supported by mapper.
	crawler.CreatedAt = types.StringValue(api.GetCreatedAt().Format("2006-01-02T15:04:05Z07:00"))
	crawler.UpdatedAt = types.StringValue(api.GetUpdatedAt().Format("2006-01-02T15:04:05Z07:00"))

	// Webhook fields — set to null when API returns empty strings.
	if api.WebhookUrl == nil || *api.WebhookUrl == "" {
		crawler.WebhookUrl = types.StringNull()
	}
	if api.WebhookAuthHeader == nil || *api.WebhookAuthHeader == "" {
		crawler.WebhookAuthHeader = types.StringNull()
	}
	if api.WebhookExtraVars == nil || *api.WebhookExtraVars == "" {
		crawler.WebhookExtraVars = types.StringNull()
	}

	crawler.Organization = types.StringValue(r.client.Organization)

	if api.DeletedAt.IsSet() && api.DeletedAt.Get() != nil {
		crawler.DeletedAt = types.StringValue(api.DeletedAt.Get().Format("2006-01-02T15:04:05Z07:00"))
	} else {
		crawler.DeletedAt = types.StringNull()
	}

	// UrlsList nullable field.
	if api.UrlsList != nil {
		crawler.UrlsList = types.StringValue(*api.UrlsList)
	} else {
		crawler.UrlsList = types.StringNull()
	}

	// ------------------------------------------------------------------
	// Parse YAML config for fields not on the top-level API response.
	// ------------------------------------------------------------------
	if api.Config != "" {
		crawler.Config = types.StringValue(api.GetConfig())
		diags.Append(parseCrawlerConfig(ctx, api.GetConfig(), crawler, api)...)
	}

	crawler.Crawler = types.StringNull()
	return
}

// CrawlerConfig is the structured representation of the YAML config blob.
type CrawlerConfig struct {
	Config struct {
		Cloud struct {
			Tracking struct {
				Enabled bool `yaml:"enabled"`
			} `yaml:"tracking"`
		} `yaml:"cloud"`
		UserAgent      string                           `yaml:"user_agent"`
		BrowserMode    bool                             `yaml:"browser_mode"`
		Workers        int                              `yaml:"workers"`
		Depth          int                              `yaml:"depth"`
		MaxHits        int                              `yaml:"max_hits"`
		MaxHtml        int                              `yaml:"max_html"`
		MaxErrors      int                              `yaml:"max_errors"`
		Cache          bool                             `yaml:"cache"`
		Delay          float64                          `yaml:"delay"`
		StatusOk       yamlList[int]                    `yaml:"status_ok"`
		Quant          map[string]interface{}           `yaml:"quant"`
		StartUrl       yamlList[string]                 `yaml:"start_url"`
		Headers        map[string]string                `yaml:"headers"`
		Exclude        yamlList[string]                 `yaml:"exclude"`
		Include        yamlList[string]                 `yaml:"include"`
		AllowedDomains yamlList[string]                 `yaml:"allowed_domains"`
		Sitemap        yamlList[map[string]interface{}] `yaml:"sitemap"`
		Assets         struct {
			NetworkIntercept struct {
				Enabled   bool `yaml:"enabled"`
				ExecuteJs bool `yaml:"execute_js"`
				Timeout   int  `yaml:"timeout"`
			} `yaml:"network_intercept"`
			Parser struct {
				Enabled bool `yaml:"enabled"`
			} `yaml:"parser"`
		} `yaml:"assets"`
		Webhook struct {
			Url        string `yaml:"url"`
			AuthHeader string `yaml:"auth_header"`
			ExtraVars  string `yaml:"extra_vars"`
		} `yaml:"webhook"`
	}
	Domain  string            `yaml:"domain"`
	Headers map[string]string `yaml:"headers"`
}

// plannedSubObject returns the planned value of an assets sub-block, or a null
// object when there is no usable planned value to keep.
func plannedSubObject(assets resource_crawler.AssetsValue, sub basetypes.ObjectValue, attrTypes map[string]attr.Type) basetypes.ObjectValue {
	// A zero-value object carries no attribute types, which happens on import.
	// Returning it verbatim would put an untyped object into state.
	if assets.IsNull() || assets.IsUnknown() || sub.IsUnknown() ||
		len(sub.AttributeTypes(context.Background())) == 0 {
		return types.ObjectNull(attrTypes)
	}
	return sub
}

// yamlList decodes a YAML sequence into a slice, and also accepts the empty
// mapping the API writes for an empty list field. A PHP empty array serialises
// as "{  }", which a plain slice field cannot decode.
type yamlList[T any] []T

func (l *yamlList[T]) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode && len(node.Content) == 0 {
		*l = nil
		return nil
	}
	var out []T
	if err := node.Decode(&out); err != nil {
		return err
	}
	*l = out
	return nil
}

// parseCrawlerConfig extracts fields from the YAML config blob that are not
// present on the top-level API response object.
func parseCrawlerConfig(ctx context.Context, configYAML string, crawler *resource_crawler.CrawlerModel, api *quantadmingo.V2Crawler) (diags diag.Diagnostics) {
	// yamlList already absorbs the empty mappings the API writes for empty list
	// fields. Any error that still reaches here may be document level, which
	// leaves the whole struct at its zero value, so the parse is discarded
	// rather than applied over good state.
	var parsed CrawlerConfig
	if err := yaml.Unmarshal([]byte(configYAML), &parsed); err != nil {
		diags.AddWarning("Unable to parse crawler config",
			fmt.Sprintf("Error parsing config YAML: %s. Some fields may not be set correctly.", err.Error()))
		return
	}

	cfg := parsed.Config

	// Boolean fields.
	crawler.BrowserMode = types.BoolValue(cfg.BrowserMode)
	crawler.Tracking = types.BoolValue(cfg.Cloud.Tracking.Enabled)

	// Numeric fields — null when zero/absent.
	crawler.Workers = nullableInt64(int64(cfg.Workers), cfg.Workers > 0)
	crawler.Depth = nullableInt64(int64(cfg.Depth), cfg.Depth != 0)
	crawler.Delay = nullableFloat64(cfg.Delay, cfg.Delay > 0)
	crawler.MaxHtml = nullableInt64(int64(cfg.MaxHtml), cfg.MaxHtml > 0)

	// MaxHits/MaxErrors: prefer top-level API field, fallback to config.
	if api.MaxHits != nil {
		crawler.MaxHits = types.Int64Value(int64(*api.MaxHits))
	} else {
		crawler.MaxHits = nullableInt64(int64(cfg.MaxHits), cfg.MaxHits >= 0)
	}
	if api.MaxErrors != nil {
		crawler.MaxErrors = types.Int64Value(int64(*api.MaxErrors))
	} else {
		crawler.MaxErrors = nullableInt64(int64(cfg.MaxErrors), cfg.MaxErrors >= 0)
	}

	// String fields.
	if cfg.UserAgent != "" {
		crawler.UserAgent = types.StringValue(cfg.UserAgent)
	} else {
		crawler.UserAgent = types.StringNull()
	}

	// String list fields — preserve plan values when API returns empty.
	crawler.Exclude = stringListOrPreserve([]string(cfg.Exclude), crawler.Exclude)
	crawler.Include = stringListOrPreserve([]string(cfg.Include), crawler.Include)
	crawler.AllowedDomains = stringListOrPreserve([]string(cfg.AllowedDomains), crawler.AllowedDomains)

	// StatusOk — int list.
	if len(cfg.StatusOk) > 0 {
		vals := make([]attr.Value, len(cfg.StatusOk))
		for i, v := range cfg.StatusOk {
			vals[i] = types.Int64Value(int64(v))
		}
		crawler.StatusOk = types.ListValueMust(types.Int64Type, vals)
	} else if crawler.StatusOk.IsUnknown() || crawler.StatusOk.ElementType(ctx) == nil {
		crawler.StatusOk = types.ListValueMust(types.Int64Type, []attr.Value{})
	}

	// Headers — preserve from plan/state when API returns empty (sensitive headers).
	if len(cfg.Headers) > 0 {
		headersMap := make(map[string]attr.Value, len(cfg.Headers))
		for k, v := range cfg.Headers {
			headersMap[k] = types.StringValue(v)
		}
		crawler.Headers = types.MapValueMust(types.StringType, headersMap)
	} else if crawler.Headers.IsUnknown() || crawler.Headers.ElementType(ctx) == nil {
		crawler.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	// else: preserve existing headers from plan/state

	// StartUrls — mapped from start_url in config. The config reports an empty
	// list for a crawler that has none, so a planned value must be preserved.
	crawler.StartUrls = stringListOrPreserve([]string(cfg.StartUrl), crawler.StartUrls)

	// Urls — preserve from plan/state; config YAML has no separate "urls" field.
	crawler.Urls = stringListOrPreserve(nil, crawler.Urls)

	// Sitemap — nested objects.
	sitemapEntryType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"url":       types.StringType,
			"recursive": types.BoolType,
		},
	}
	if len(cfg.Sitemap) > 0 {
		sitemapVals := make([]attr.Value, len(cfg.Sitemap))
		for i, entry := range cfg.Sitemap {
			url := ""
			recursive := false
			if urlVal, ok := entry["url"].(string); ok {
				url = urlVal
			}
			if recursiveVal, ok := entry["recursive"].(bool); ok {
				recursive = recursiveVal
			}
			objVal, _ := types.ObjectValue(
				sitemapEntryType.AttrTypes,
				map[string]attr.Value{
					"url":       types.StringValue(url),
					"recursive": types.BoolValue(recursive),
				},
			)
			sitemapVals[i] = objVal
		}
		crawler.Sitemap, _ = types.ListValue(sitemapEntryType, sitemapVals)
	} else if crawler.Sitemap.IsNull() || crawler.Sitemap.IsUnknown() ||
		crawler.Sitemap.ElementType(ctx) == nil {
		crawler.Sitemap = types.ListNull(sitemapEntryType)
	}

	// Assets — network_intercept and parser nested objects.
	crawler.Assets = crawlerAssetsFromConfig(ctx, &parsed, crawler.Assets)

	return
}

// crawlerAssetsFromConfig rebuilds the assets value from the crawler config the
// API returned. current is the planned value; it is kept when the config
// reports no assets, so that a crawler without assets is left untouched.
func crawlerAssetsFromConfig(ctx context.Context, cfg *CrawlerConfig, current resource_crawler.AssetsValue) resource_crawler.AssetsValue {
	niAttrTypes := resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx)
	parserAttrTypes := resource_crawler.ParserValue{}.AttributeTypes(ctx)

	hasNetworkIntercept := cfg.Config.Assets.NetworkIntercept.Enabled ||
		cfg.Config.Assets.NetworkIntercept.ExecuteJs ||
		cfg.Config.Assets.NetworkIntercept.Timeout > 0
	hasParser := cfg.Config.Assets.Parser.Enabled

	if !hasNetworkIntercept && !hasParser {
		if current.IsNull() || current.IsUnknown() {
			return resource_crawler.NewAssetsValueNull()
		}
		return current
	}

	// The backend may echo one sub-block and not the other. A sub-block the
	// config does not report keeps its planned value, or Terraform reports an
	// inconsistent result after apply.
	networkIntercept := plannedSubObject(current, current.NetworkIntercept, niAttrTypes)
	if hasNetworkIntercept {
		networkIntercept = types.ObjectValueMust(niAttrTypes, map[string]attr.Value{
			"enabled":    types.BoolValue(cfg.Config.Assets.NetworkIntercept.Enabled),
			"execute_js": types.BoolValue(cfg.Config.Assets.NetworkIntercept.ExecuteJs),
			"timeout":    types.Int64Value(int64(cfg.Config.Assets.NetworkIntercept.Timeout)),
		})
	}

	parser := plannedSubObject(current, current.Parser, parserAttrTypes)
	if hasParser {
		parser = types.ObjectValueMust(parserAttrTypes, map[string]attr.Value{
			"enabled": types.BoolValue(true),
		})
	}

	return resource_crawler.NewAssetsValueMust(
		resource_crawler.AssetsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"network_intercept": networkIntercept,
			"parser":            parser,
		},
	)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------
func callCrawlerDeleteAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	if crawler.Uuid.IsUnknown() || crawler.Uuid.IsNull() {
		diags.AddAttributeError(path.Root("uuid"), "Missing crawler.uuid attribute",
			"To delete crawler information the crawler uuid must be provided")
		return
	}
	if crawler.Project.IsNull() || crawler.Project.IsUnknown() {
		diags.AddAttributeError(path.Root("project"), "Missing crawler.project attribute",
			"To delete crawler information the crawler project must be provided")
		return
	}

	_, err := r.client.Instance.CrawlersAPI.CrawlersDelete(
		r.client.AuthContext, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Unable to delete crawler", fmt.Sprintf("Error: %s", err.Error()))
	}
	return
}

// ---------------------------------------------------------------------------
// Update — reuses buildCrawlerRequest for the shared ToSDK + manual logic.
// ---------------------------------------------------------------------------
func callCrawlerUpdateAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	req, d := buildCrawlerRequest(ctx, crawler)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	_, _, err := r.client.Instance.CrawlersAPI.CrawlersUpdate(
		r.client.AuthContext, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString(),
	).V2CrawlerRequest(*req).Execute()

	if err != nil {
		diags.AddError("Unable to update crawler", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	return callCrawlerReadAPI(ctx, r, crawler)
}

// ---------------------------------------------------------------------------
// ModifyPlan
// ---------------------------------------------------------------------------
func (r *crawlerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainChanging := !plan.Domain.Equal(state.Domain)
	if domainChanging {
		plan.DomainVerified = types.Int64Value(0)
	} else {
		plan.DomainVerified = state.DomainVerified
	}

	plan.CreatedAt = state.CreatedAt
	plan.Id = state.Id
	plan.ProjectId = state.ProjectId

	if !plan.Exclude.IsNull() && !plan.Exclude.IsUnknown() && state.Exclude.IsNull() {
		// Keep the exclude from the plan.
	} else if !state.Exclude.IsNull() && !state.Exclude.IsUnknown() {
		if len(plan.Exclude.Elements()) == 0 && len(state.Exclude.Elements()) > 0 {
			plan.Exclude = state.Exclude
		}
	}

	// Changing any other attribute makes the framework re-plan unconfigured
	// computed attributes as unknown. assets cannot take an attribute plan
	// modifier (its generated type rejects the conversion), so keep the prior
	// value here; otherwise the Pulumi bridge reports a perpetual update.
	plan.Assets = keepPriorAssetsIfUnknown(plan.Assets, state.Assets)

	resp.Plan.Set(ctx, &plan)
}

// keepPriorAssetsIfUnknown returns the prior assets when the planned value is
// unknown, mirroring UseStateForUnknown for the generated assets type.
func keepPriorAssetsIfUnknown(planned, prior resource_crawler.AssetsValue) resource_crawler.AssetsValue {
	if planned.IsUnknown() {
		return prior
	}
	return planned
}

// ---------------------------------------------------------------------------
// ImportState
// ---------------------------------------------------------------------------
func (r *crawlerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	var project, uuid string

	if len(parts) == 2 {
		project = parts[0]
		uuid = parts[1]
	} else if len(parts) == 1 {
		project = "default"
		uuid = parts[0]
	} else {
		resp.Diagnostics.AddError("Invalid Import ID",
			"Import ID should be in format 'project:uuid' or just 'uuid' for default project")
		return
	}

	var data resource_crawler.CrawlerModel
	data.Project = types.StringValue(project)
	data.Uuid = types.StringValue(uuid)

	diags := callCrawlerReadAPI(ctx, r, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// nullableInt64 returns a typed Int64 value or null based on condition.
func nullableInt64(val int64, present bool) types.Int64 {
	if present {
		return types.Int64Value(val)
	}
	return types.Int64Null()
}

// nullableFloat64 returns a typed Float64 value or null based on condition.
func nullableFloat64(val float64, present bool) types.Float64 {
	if present {
		return types.Float64Value(val)
	}
	return types.Float64Null()
}

// stringListOrPreserve converts a Go string slice to a TF list. If the slice
// is empty, it preserves the existing TF value (for plan/state drift avoidance)
// or returns an empty list if the existing value is null/unknown.
func stringListOrPreserve(vals []string, existing types.List) types.List {
	if len(vals) > 0 {
		attrVals := make([]attr.Value, len(vals))
		for i, v := range vals {
			attrVals[i] = types.StringValue(v)
		}
		return types.ListValueMust(types.StringType, attrVals)
	}
	// An unknown value must resolve: the framework forbids leaving one unknown
	// after apply. A zero-value list carries no element type, which happens on
	// import, where there is no prior plan or state; returning it verbatim puts
	// an untyped list into state and State.Set rejects it. Otherwise a planned
	// null, or a planned value, is returned unchanged so that the read never
	// invents a list the user did not ask for.
	if existing.IsUnknown() || existing.ElementType(context.Background()) == nil {
		return types.ListValueMust(types.StringType, []attr.Value{})
	}
	return existing
}
