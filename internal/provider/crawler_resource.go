package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_crawler"

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

	resp.Diagnostics.Append(callCrawlerReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_crawler.CrawlerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var state resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Preserve computed identifiers from state.
	data.Uuid = state.Uuid
	data.Id = state.Id

	if resp.Diagnostics.HasError() {
		return
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
	diags.Append(mapper.ToSDK(ctx, crawler, req)...)
	if diags.HasError() {
		return nil, diags
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

	// Assets — nested object with network_intercept sub-object.
	if !crawler.Assets.IsNull() && !crawler.Assets.IsUnknown() {
		if !crawler.Assets.NetworkIntercept.IsNull() && !crawler.Assets.NetworkIntercept.IsUnknown() {
			var networkIntercept resource_crawler.NetworkInterceptValue
			diags.Append(crawler.Assets.NetworkIntercept.As(ctx, &networkIntercept, basetypes.ObjectAsOptions{})...)
			if !diags.HasError() {
				assetsObj := quantadmingo.NewV2CrawlerAssets()
				niObj := quantadmingo.NewV2CrawlerAssetsNetworkIntercept()
				niObj.SetEnabled(networkIntercept.Enabled.ValueBool())
				niObj.SetTimeout(int32(networkIntercept.Timeout.ValueInt64()))
				assetsObj.SetNetworkIntercept(*niObj)
				req.SetAssets(*assetsObj)
			}
		}
	}

	return req, diags
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

	api, _, err := r.client.Instance.CrawlersAPI.CrawlersRead(
		r.client.AuthContext, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Unable to read crawler", fmt.Sprintf("Error: %s", err.Error()))
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
		UserAgent      string                   `yaml:"user_agent"`
		BrowserMode    bool                     `yaml:"browser_mode"`
		Workers        int                      `yaml:"workers"`
		Depth          int                      `yaml:"depth"`
		MaxHits        int                      `yaml:"max_hits"`
		MaxHtml        int                      `yaml:"max_html"`
		MaxErrors      int                      `yaml:"max_errors"`
		Cache          bool                     `yaml:"cache"`
		Delay          float64                  `yaml:"delay"`
		StatusOk       []int                    `yaml:"status_ok"`
		Quant          map[string]interface{}   `yaml:"quant"`
		StartUrl       []string                 `yaml:"start_url"`
		Headers        map[string]string        `yaml:"headers"`
		Exclude        []string                 `yaml:"exclude"`
		Include        []string                 `yaml:"include"`
		AllowedDomains []string                 `yaml:"allowed_domains"`
		Sitemap        []map[string]interface{} `yaml:"sitemap"`
		Assets         struct {
			NetworkIntercept struct {
				Enabled   bool `yaml:"enabled"`
				ExecuteJs bool `yaml:"execute_js"`
				Timeout   int  `yaml:"timeout"`
			} `yaml:"network_intercept"`
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

// parseCrawlerConfig extracts fields from the YAML config blob that are not
// present on the top-level API response object.
func parseCrawlerConfig(ctx context.Context, configYAML string, crawler *resource_crawler.CrawlerModel, api *quantadmingo.V2Crawler) (diags diag.Diagnostics) {
	var parsed CrawlerConfig
	if err := yaml.Unmarshal([]byte(configYAML), &parsed); err != nil {
		diags.AddWarning("Unable to parse crawler config",
			fmt.Sprintf("Error parsing config YAML: %s. Some fields may not be set correctly.", err.Error()))
		return
	}

	cfg := parsed.Config

	// Boolean fields.
	crawler.BrowserMode = types.BoolValue(cfg.BrowserMode)

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
	crawler.Exclude = stringListOrPreserve(cfg.Exclude, crawler.Exclude)
	crawler.Include = stringListOrPreserve(cfg.Include, crawler.Include)
	crawler.AllowedDomains = stringListOrPreserve(cfg.AllowedDomains, crawler.AllowedDomains)

	// StatusOk — int list.
	if len(cfg.StatusOk) > 0 {
		vals := make([]attr.Value, len(cfg.StatusOk))
		for i, v := range cfg.StatusOk {
			vals[i] = types.Int64Value(int64(v))
		}
		crawler.StatusOk = types.ListValueMust(types.Int64Type, vals)
	} else if crawler.StatusOk.IsNull() || crawler.StatusOk.IsUnknown() {
		crawler.StatusOk = types.ListValueMust(types.Int64Type, []attr.Value{})
	}

	// Headers — preserve from plan/state when API returns empty (sensitive headers).
	if len(cfg.Headers) > 0 {
		headersMap := make(map[string]attr.Value, len(cfg.Headers))
		for k, v := range cfg.Headers {
			headersMap[k] = types.StringValue(v)
		}
		crawler.Headers = types.MapValueMust(types.StringType, headersMap)
	} else if crawler.Headers.IsNull() || crawler.Headers.IsUnknown() {
		crawler.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	// else: preserve existing headers from plan/state

	// StartUrls — mapped from start_url in config.
	if len(cfg.StartUrl) > 0 {
		vals := make([]attr.Value, len(cfg.StartUrl))
		for i, v := range cfg.StartUrl {
			vals[i] = types.StringValue(v)
		}
		crawler.StartUrls = types.ListValueMust(types.StringType, vals)
	} else {
		crawler.StartUrls = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Urls — preserve from state; config YAML has no separate "urls" field.
	if crawler.Urls.IsNull() || crawler.Urls.IsUnknown() {
		crawler.Urls = types.ListValueMust(types.StringType, []attr.Value{})
	}

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
	} else if crawler.Sitemap.IsNull() || crawler.Sitemap.IsUnknown() {
		crawler.Sitemap = types.ListNull(sitemapEntryType)
	}

	// Assets — network_intercept nested object.
	niAttrTypes := resource_crawler.NetworkInterceptValue{}.AttributeTypes(ctx)
	parserAttrTypes := resource_crawler.ParserValue{}.AttributeTypes(ctx)
	assetsAttrTypes := resource_crawler.AssetsValue{}.AttributeTypes(ctx)

	if cfg.Assets.NetworkIntercept.Enabled || cfg.Assets.NetworkIntercept.Timeout > 0 {
		networkInterceptObj, _ := types.ObjectValue(
			niAttrTypes,
			map[string]attr.Value{
				"enabled":    types.BoolValue(cfg.Assets.NetworkIntercept.Enabled),
				"execute_js": types.BoolValue(cfg.Assets.NetworkIntercept.ExecuteJs),
				"timeout":    types.Int64Value(int64(cfg.Assets.NetworkIntercept.Timeout)),
			},
		)
		crawler.Assets = resource_crawler.NewAssetsValueMust(
			assetsAttrTypes,
			map[string]attr.Value{
				"network_intercept": networkInterceptObj,
				"parser":            types.ObjectNull(parserAttrTypes),
			},
		)
	} else if crawler.Assets.IsNull() || crawler.Assets.IsUnknown() {
		crawler.Assets = resource_crawler.NewAssetsValueNull()
	}

	return
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

	resp.Plan.Set(ctx, &plan)
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
	if !existing.IsNull() && !existing.IsUnknown() {
		return existing
	}
	return types.ListValueMust(types.StringType, []attr.Value{})
}
