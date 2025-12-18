package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_crawler"

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
	resp.Schema = resource_crawler.CrawlerResourceSchema(ctx)
}

func (r *crawlerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unepxected resource configure type",
			fmt.Sprintf("Expected *internal.Client, got: %T. Please report this issue to the provider developers", req.ProviderData),
		)
	}
	r.client = client
}

func (r *crawlerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_crawler.CrawlerModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_crawler.CrawlerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_crawler.CrawlerModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Read the current state to get the UUID and other identifiers
	var state resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Preserve the UUID and ID from the current state
	data.Uuid = state.Uuid
	data.Id = state.Id

	if resp.Diagnostics.HasError() {
		return
	}

	// Update the crawler object.
	resp.Diagnostics.Append(callCrawlerUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_crawler.CrawlerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callCrawlerDeleteAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func callCrawlerCreateAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	req := *quantadmingo.NewV2CrawlerRequestWithDefaults()

	// Required fields
	req.SetDomain(crawler.Domain.ValueString())

	// Basic optional fields
	if !crawler.Name.IsNull() && !crawler.Name.IsUnknown() {
		req.SetName(crawler.Name.ValueString())
	}

	if !crawler.BrowserMode.IsNull() && !crawler.BrowserMode.IsUnknown() {
		req.SetBrowserMode(crawler.BrowserMode.ValueBool())
	}

	// URLs - explicit list to crawl (no discovery)
	if !crawler.Urls.IsNull() && !crawler.Urls.IsUnknown() {
		urls := make([]string, 0, len(crawler.Urls.Elements()))
		diags.Append(crawler.Urls.ElementsAs(ctx, &urls, false)...)
		if !diags.HasError() {
			req.SetUrls(urls)
		}
	}

	// Start URLs - starting points for discovery crawl
	if !crawler.StartUrls.IsNull() && !crawler.StartUrls.IsUnknown() {
		startUrls := make([]string, 0, len(crawler.StartUrls.Elements()))
		diags.Append(crawler.StartUrls.ElementsAs(ctx, &startUrls, false)...)
		if !diags.HasError() {
			req.SetStartUrls(startUrls)
		}
	}

	// Exclude patterns
	if !crawler.Exclude.IsNull() && !crawler.Exclude.IsUnknown() {
		exclude := make([]string, 0, len(crawler.Exclude.Elements()))
		diags.Append(crawler.Exclude.ElementsAs(ctx, &exclude, false)...)
		if !diags.HasError() {
			req.SetExclude(exclude)
		}
	}

	// Include patterns
	if !crawler.Include.IsNull() && !crawler.Include.IsUnknown() {
		include := make([]string, 0, len(crawler.Include.Elements()))
		diags.Append(crawler.Include.ElementsAs(ctx, &include, false)...)
		if !diags.HasError() {
			req.SetInclude(include)
		}
	}

	// Headers
	if !crawler.Headers.IsNull() && !crawler.Headers.IsUnknown() {
		headers := make(map[string]string, len(crawler.Headers.Elements()))
		diags.Append(crawler.Headers.ElementsAs(ctx, &headers, false)...)
		if !diags.HasError() {
			req.SetHeaders(headers)
		}
	}

	// Webhook configuration
	if !crawler.WebhookUrl.IsNull() && !crawler.WebhookUrl.IsUnknown() {
		req.SetWebhookUrl(crawler.WebhookUrl.ValueString())
	}

	if !crawler.WebhookAuthHeader.IsNull() && !crawler.WebhookAuthHeader.IsUnknown() {
		req.SetWebhookAuthHeader(crawler.WebhookAuthHeader.ValueString())
	}

	if !crawler.WebhookExtraVars.IsNull() && !crawler.WebhookExtraVars.IsUnknown() {
		req.SetWebhookExtraVars(crawler.WebhookExtraVars.ValueString())
	}

	// Advanced settings (verified domains only)
	if !crawler.Workers.IsNull() && !crawler.Workers.IsUnknown() {
		req.SetWorkers(int32(crawler.Workers.ValueInt64()))
	}

	if !crawler.Delay.IsNull() && !crawler.Delay.IsUnknown() {
		req.SetDelay(float32(crawler.Delay.ValueFloat64()))
	}

	if !crawler.Depth.IsNull() && !crawler.Depth.IsUnknown() {
		req.SetDepth(int32(crawler.Depth.ValueInt64()))
	}

	if !crawler.MaxHits.IsNull() && !crawler.MaxHits.IsUnknown() {
		req.SetMaxHits(int32(crawler.MaxHits.ValueInt64()))
	}

	if !crawler.MaxHtml.IsNull() && !crawler.MaxHtml.IsUnknown() {
		req.SetMaxHtml(int32(crawler.MaxHtml.ValueInt64()))
	}

	if !crawler.MaxErrors.IsNull() && !crawler.MaxErrors.IsUnknown() {
		req.SetMaxErrors(int32(crawler.MaxErrors.ValueInt64()))
	}

	if !crawler.UserAgent.IsNull() && !crawler.UserAgent.IsUnknown() {
		req.SetUserAgent(crawler.UserAgent.ValueString())
	}

	// Status OK codes
	if !crawler.StatusOk.IsNull() && !crawler.StatusOk.IsUnknown() {
		statusOk := make([]int32, 0, len(crawler.StatusOk.Elements()))
		var statusOkInt64 []int64
		diags.Append(crawler.StatusOk.ElementsAs(ctx, &statusOkInt64, false)...)
		if !diags.HasError() {
			for _, v := range statusOkInt64 {
				statusOk = append(statusOk, int32(v))
			}
			req.SetStatusOk(statusOk)
		}
	}

	// Allowed domains
	if !crawler.AllowedDomains.IsNull() && !crawler.AllowedDomains.IsUnknown() {
		allowedDomains := make([]string, 0, len(crawler.AllowedDomains.Elements()))
		diags.Append(crawler.AllowedDomains.ElementsAs(ctx, &allowedDomains, false)...)
		if !diags.HasError() {
			req.SetAllowedDomains(allowedDomains)
		}
	}

	// Complex object fields - convert nested objects to API format
	if !crawler.Sitemap.IsNull() && !crawler.Sitemap.IsUnknown() {
		var sitemapEntries []resource_crawler.SitemapValue
		diags.Append(crawler.Sitemap.ElementsAs(ctx, &sitemapEntries, false)...)
		if !diags.HasError() {
			sitemapArray := make([]quantadmingo.V2CrawlerSitemapInner, len(sitemapEntries))
			for i, entry := range sitemapEntries {
				sitemapItem := quantadmingo.NewV2CrawlerSitemapInner()
				url := entry.Url.ValueString()
				recursive := entry.Recursive.ValueBool()
				sitemapItem.SetUrl(url)
				sitemapItem.SetRecursive(recursive)
				sitemapArray[i] = *sitemapItem
			}
			req.SetSitemap(sitemapArray)
		}
	}

	if !crawler.Assets.IsNull() && !crawler.Assets.IsUnknown() {
		// Assets.NetworkIntercept is a basetypes.ObjectValue - convert to NetworkInterceptValue
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

	api, httpResp, err := r.client.Instance.CrawlersAPI.CrawlersCreate(r.client.AuthContext, r.client.Organization, crawler.Project.ValueString()).V2CrawlerRequest(req).Execute()

	if err != nil {
		// Try to extract detailed error message from API response
		errorMsg := err.Error()
		if httpResp != nil && httpResp.Body != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)

			// Try to parse API error response
			var apiError struct {
				Error   bool   `json:"error"`
				Message string `json:"message"`
			}
			if jsonErr := json.Unmarshal(bodyBytes, &apiError); jsonErr == nil && apiError.Message != "" {
				errorMsg = apiError.Message
			}
		}

		diags.AddError(
			"Unable to create crawler",
			errorMsg,
		)
		return diags
	}

	// Set the UUID and ID from the API response
	crawler.Uuid = types.StringValue(api.GetUuid())
	crawler.Id = types.Int64Value(int64(api.GetId()))

	// Post-create read to populate computed fields
	return callCrawlerReadAPI(ctx, r, crawler)
}

func callCrawlerReadAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	// Validate required fields
	if crawler.Uuid.IsUnknown() || crawler.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing crawler.uuid attribute",
			"To read crawler information, uuid must be provided.",
		)
		return diags
	}

	if crawler.Project.IsNull() || crawler.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing crawler.project attribute",
			"To read crawler information, project must be provided.",
		)
		return diags
	}

	// API call with built-in rate limiting and retry logic
	api, _, err := r.client.Instance.CrawlersAPI.CrawlersRead(ctx, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString()).Execute()

	if err != nil {
		diags.AddError("Unable to read crawler", fmt.Sprintf("Error: %s", err.Error()))
		return diags
	}

	// Validate that the API returned the expected data
	if api == nil {
		diags.AddError(
			"Invalid API response",
			"The API returned a nil response when reading crawler data",
		)
		return diags
	}

	// Set all fields from the API response
	crawler.Id = types.Int64Value(int64(api.GetId()))
	crawler.ProjectId = types.Int64Value(int64(api.GetProjectId()))
	crawler.Uuid = types.StringValue(api.GetUuid())
	crawler.Name = types.StringValue(api.GetName())
	crawler.Domain = types.StringValue(api.GetDomain())
	crawler.DomainVerified = types.Int64Value(int64(api.GetDomainVerified()))
	crawler.CreatedAt = types.StringValue(api.GetCreatedAt().Format("2006-01-02T15:04:05Z07:00"))
	crawler.UpdatedAt = types.StringValue(api.GetUpdatedAt().Format("2006-01-02T15:04:05Z07:00"))

	// Set top-level fields from API response (these are returned at the root level, not in config YAML)
	if api.WebhookUrl != nil && *api.WebhookUrl != "" {
		crawler.WebhookUrl = types.StringValue(*api.WebhookUrl)
	} else {
		crawler.WebhookUrl = types.StringNull()
	}

	if api.WebhookAuthHeader != nil && *api.WebhookAuthHeader != "" {
		crawler.WebhookAuthHeader = types.StringValue(*api.WebhookAuthHeader)
	} else {
		crawler.WebhookAuthHeader = types.StringNull()
	}

	if api.WebhookExtraVars != nil && *api.WebhookExtraVars != "" {
		crawler.WebhookExtraVars = types.StringValue(*api.WebhookExtraVars)
	} else {
		crawler.WebhookExtraVars = types.StringNull()
	}

	// Note: execute_js is now only in assets.network_intercept, not at top-level

	// Set organization to the current organization
	crawler.Organization = types.StringValue(r.client.Organization)

	// Set deleted_at (null if not deleted)
	if api.DeletedAt.IsSet() && api.DeletedAt.Get() != nil {
		crawler.DeletedAt = types.StringValue(api.DeletedAt.Get().Format("2006-01-02T15:04:05Z07:00"))
	} else {
		crawler.DeletedAt = types.StringNull()
	}

	// Improved approach with better error handling and structure
	if api.Config != "" {
		crawler.Config = types.StringValue(api.GetConfig())

		// Define a structured type for the config
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

		var parsedConfig CrawlerConfig
		if err := yaml.Unmarshal([]byte(api.GetConfig()), &parsedConfig); err != nil {
			diags.AddWarning(
				"Unable to parse crawler config",
				fmt.Sprintf("Error parsing config YAML: %s. Some fields may not be set correctly.", err.Error()),
			)
		} else {
			// Set basic fields directly from the structured config
			crawler.BrowserMode = types.BoolValue(parsedConfig.Config.BrowserMode)
			// Note: execute_js is read from top-level API response above, not from config

			// Set numeric fields
			if parsedConfig.Config.Workers > 0 {
				crawler.Workers = types.Int64Value(int64(parsedConfig.Config.Workers))
			} else {
				crawler.Workers = types.Int64Null()
			}

			if parsedConfig.Config.Depth != 0 {
				crawler.Depth = types.Int64Value(int64(parsedConfig.Config.Depth))
			} else {
				crawler.Depth = types.Int64Null()
			}

			// Use top-level API field if available, otherwise use config YAML
			if api.MaxHits != nil {
				crawler.MaxHits = types.Int64Value(int64(*api.MaxHits))
			} else if parsedConfig.Config.MaxHits >= 0 {
				crawler.MaxHits = types.Int64Value(int64(parsedConfig.Config.MaxHits))
			} else {
				crawler.MaxHits = types.Int64Null()
			}

			if parsedConfig.Config.MaxHtml > 0 {
				crawler.MaxHtml = types.Int64Value(int64(parsedConfig.Config.MaxHtml))
			} else {
				crawler.MaxHtml = types.Int64Null()
			}

			// Use top-level API field if available, otherwise use config YAML
			if api.MaxErrors != nil {
				crawler.MaxErrors = types.Int64Value(int64(*api.MaxErrors))
			} else if parsedConfig.Config.MaxErrors >= 0 {
				crawler.MaxErrors = types.Int64Value(int64(parsedConfig.Config.MaxErrors))
			} else {
				crawler.MaxErrors = types.Int64Null()
			}

			if parsedConfig.Config.Delay > 0 {
				crawler.Delay = types.Float64Value(parsedConfig.Config.Delay)
			} else {
				crawler.Delay = types.Float64Null()
			}

			// Set user agent
			if parsedConfig.Config.UserAgent != "" {
				crawler.UserAgent = types.StringValue(parsedConfig.Config.UserAgent)
			} else {
				crawler.UserAgent = types.StringNull()
			}

			// Handle exclude list - preserve values from plan if API returns empty
			if len(parsedConfig.Config.Exclude) > 0 {
				excludeVals := make([]attr.Value, len(parsedConfig.Config.Exclude))
				for i, v := range parsedConfig.Config.Exclude {
					excludeVals[i] = types.StringValue(v)
				}
				crawler.Exclude = types.ListValueMust(types.StringType, excludeVals)
			} else if !crawler.Exclude.IsNull() && !crawler.Exclude.IsUnknown() {
				// If API returned empty but we had values in config, preserve them
				// Keep the existing values from the plan
			} else {
				crawler.Exclude = types.ListValueMust(types.StringType, []attr.Value{})
			}

			// Handle include list
			if len(parsedConfig.Config.Include) > 0 {
				includeVals := make([]attr.Value, len(parsedConfig.Config.Include))
				for i, v := range parsedConfig.Config.Include {
					includeVals[i] = types.StringValue(v)
				}
				crawler.Include = types.ListValueMust(types.StringType, includeVals)
			} else if !crawler.Include.IsNull() && !crawler.Include.IsUnknown() {
				// Preserve existing values
			} else {
				crawler.Include = types.ListValueMust(types.StringType, []attr.Value{})
			}

			// Handle allowed domains list
			if len(parsedConfig.Config.AllowedDomains) > 0 {
				allowedDomainsVals := make([]attr.Value, len(parsedConfig.Config.AllowedDomains))
				for i, v := range parsedConfig.Config.AllowedDomains {
					allowedDomainsVals[i] = types.StringValue(v)
				}
				crawler.AllowedDomains = types.ListValueMust(types.StringType, allowedDomainsVals)
			} else if !crawler.AllowedDomains.IsNull() && !crawler.AllowedDomains.IsUnknown() {
				// Preserve existing values
			} else {
				crawler.AllowedDomains = types.ListValueMust(types.StringType, []attr.Value{})
			}

			// Handle status_ok list
			if len(parsedConfig.Config.StatusOk) > 0 {
				statusOkVals := make([]attr.Value, len(parsedConfig.Config.StatusOk))
				for i, v := range parsedConfig.Config.StatusOk {
					statusOkVals[i] = types.Int64Value(int64(v))
				}
				crawler.StatusOk = types.ListValueMust(types.Int64Type, statusOkVals)
			} else if !crawler.StatusOk.IsNull() && !crawler.StatusOk.IsUnknown() {
				// Preserve existing values
			} else {
				crawler.StatusOk = types.ListValueMust(types.Int64Type, []attr.Value{})
			}

			// Handle headers - preserve original headers if API doesn't return them
			if len(parsedConfig.Config.Headers) > 0 {
				headersMap := make(map[string]attr.Value)
				for k, v := range parsedConfig.Config.Headers {
					headersMap[k] = types.StringValue(v)
				}
				crawler.Headers = types.MapValueMust(types.StringType, headersMap)
			} else if !crawler.Headers.IsNull() && !crawler.Headers.IsUnknown() {
				// If API returned empty but we had headers in config, preserve them
				// This handles cases where API doesn't return sensitive headers like Authorization
				// Keep the existing headers from the plan/state
			} else {
				crawler.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{})
			}

			// Initialize start_urls from start_url in config
			if len(parsedConfig.Config.StartUrl) > 0 {
				startUrlVals := make([]attr.Value, len(parsedConfig.Config.StartUrl))
				for i, v := range parsedConfig.Config.StartUrl {
					startUrlVals[i] = types.StringValue(v)
				}
				crawler.StartUrls = types.ListValueMust(types.StringType, startUrlVals)
			} else {
				crawler.StartUrls = types.ListValueMust(types.StringType, []attr.Value{})
			}

			// Note: The config YAML may not have a separate "urls" field
			// If it does, handle it here. For now, initialize as empty if not in state
			if !crawler.Urls.IsNull() && !crawler.Urls.IsUnknown() {
				// Preserve existing urls from state/plan
			} else {
				crawler.Urls = types.ListValueMust(types.StringType, []attr.Value{})
			}

			// NOTE: Webhook fields (url, auth_header, extra_vars) are read from top-level API response above,
			// not from the config YAML, so we don't parse them here

			// Handle complex object fields (sitemap, assets)
			// Convert YAML objects to Terraform nested structures
			if len(parsedConfig.Config.Sitemap) > 0 {
				sitemapVals := make([]attr.Value, len(parsedConfig.Config.Sitemap))
				sitemapEntryType := types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"url":       types.StringType,
						"recursive": types.BoolType,
					},
				}
				for i, entry := range parsedConfig.Config.Sitemap {
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
			} else if !crawler.Sitemap.IsNull() && !crawler.Sitemap.IsUnknown() {
				// Preserve existing value from state/plan
			} else {
				crawler.Sitemap = types.ListNull(types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"url":       types.StringType,
						"recursive": types.BoolType,
					},
				})
			}

			// Parse assets.network_intercept from the structured config
			if parsedConfig.Config.Assets.NetworkIntercept.Enabled || parsedConfig.Config.Assets.NetworkIntercept.Timeout > 0 {
				networkInterceptObj, _ := types.ObjectValue(
					map[string]attr.Type{
						"enabled": types.BoolType,
						"timeout": types.Int64Type,
					},
					map[string]attr.Value{
						"enabled": types.BoolValue(parsedConfig.Config.Assets.NetworkIntercept.Enabled),
						"timeout": types.Int64Value(int64(parsedConfig.Config.Assets.NetworkIntercept.Timeout)),
					},
				)

				crawler.Assets = resource_crawler.NewAssetsValueMust(
					map[string]attr.Type{
						"network_intercept": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"enabled": types.BoolType,
								"timeout": types.Int64Type,
							},
						},
						"parser": types.ObjectType{AttrTypes: map[string]attr.Type{}},
					},
					map[string]attr.Value{
						"network_intercept": networkInterceptObj,
						"parser":            types.ObjectNull(map[string]attr.Type{}),
					},
				)
			} else if !crawler.Assets.IsNull() && !crawler.Assets.IsUnknown() {
				// Preserve existing value from state/plan
			} else {
				crawler.Assets = resource_crawler.NewAssetsValueNull()
			}

			// Make sure crawler field is initialized
			crawler.Crawler = types.StringNull()
		}
	}

	// Set urls_list
	if api.UrlsList != nil {
		crawler.UrlsList = types.StringValue(*api.UrlsList)
	} else {
		crawler.UrlsList = types.StringNull()
	}

	return diags
}

func callCrawlerDeleteAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	// Validate required fields for deletion
	if crawler.Uuid.IsUnknown() || crawler.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing crawler.uuid attribute",
			"To delete crawler information the crawler uuid must be provided",
		)
		return
	}

	if crawler.Project.IsNull() || crawler.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing crawler.project attribute",
			"To delete crawler information the crawler project must be provided",
		)
		return
	}

	// Delete API call with built-in rate limiting and retry logic
	_, err := r.client.Instance.CrawlersAPI.CrawlersDelete(
		ctx,
		r.client.Organization,
		crawler.Project.ValueString(),
		crawler.Uuid.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Unable to delete crawler", fmt.Sprintf("Error: %s", err.Error()))
	}

	return diags
}

func callCrawlerUpdateAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {

	req := *quantadmingo.NewV2CrawlerRequestWithDefaults()

	// Basic fields
	if !crawler.Domain.IsNull() && !crawler.Domain.IsUnknown() {
		req.SetDomain(crawler.Domain.ValueString())
	}

	if !crawler.Name.IsNull() && !crawler.Name.IsUnknown() {
		req.SetName(crawler.Name.ValueString())
	}

	if !crawler.BrowserMode.IsNull() && !crawler.BrowserMode.IsUnknown() {
		req.SetBrowserMode(crawler.BrowserMode.ValueBool())
	}

	// URLs - explicit list to crawl (no discovery)
	if !crawler.Urls.IsNull() && !crawler.Urls.IsUnknown() {
		urls := make([]string, 0, len(crawler.Urls.Elements()))
		diags.Append(crawler.Urls.ElementsAs(ctx, &urls, false)...)
		if !diags.HasError() {
			req.SetUrls(urls)
		}
	}

	// Start URLs - starting points for discovery crawl
	if !crawler.StartUrls.IsNull() && !crawler.StartUrls.IsUnknown() {
		startUrls := make([]string, 0, len(crawler.StartUrls.Elements()))
		diags.Append(crawler.StartUrls.ElementsAs(ctx, &startUrls, false)...)
		if !diags.HasError() {
			req.SetStartUrls(startUrls)
		}
	}

	// Exclude patterns
	if !crawler.Exclude.IsNull() && !crawler.Exclude.IsUnknown() {
		exclude := make([]string, 0, len(crawler.Exclude.Elements()))
		diags.Append(crawler.Exclude.ElementsAs(ctx, &exclude, false)...)
		if !diags.HasError() {
			req.SetExclude(exclude)
		}
	}

	// Include patterns
	if !crawler.Include.IsNull() && !crawler.Include.IsUnknown() {
		include := make([]string, 0, len(crawler.Include.Elements()))
		diags.Append(crawler.Include.ElementsAs(ctx, &include, false)...)
		if !diags.HasError() {
			req.SetInclude(include)
		}
	}

	// Headers
	if !crawler.Headers.IsNull() && !crawler.Headers.IsUnknown() {
		headers := make(map[string]string, len(crawler.Headers.Elements()))
		diags.Append(crawler.Headers.ElementsAs(ctx, &headers, false)...)
		if !diags.HasError() {
			req.SetHeaders(headers)
		}
	}

	// Webhook configuration
	if !crawler.WebhookUrl.IsNull() && !crawler.WebhookUrl.IsUnknown() {
		req.SetWebhookUrl(crawler.WebhookUrl.ValueString())
	}

	if !crawler.WebhookAuthHeader.IsNull() && !crawler.WebhookAuthHeader.IsUnknown() {
		req.SetWebhookAuthHeader(crawler.WebhookAuthHeader.ValueString())
	}

	if !crawler.WebhookExtraVars.IsNull() && !crawler.WebhookExtraVars.IsUnknown() {
		req.SetWebhookExtraVars(crawler.WebhookExtraVars.ValueString())
	}

	// Advanced settings (verified domains only)
	if !crawler.Workers.IsNull() && !crawler.Workers.IsUnknown() {
		req.SetWorkers(int32(crawler.Workers.ValueInt64()))
	}

	if !crawler.Delay.IsNull() && !crawler.Delay.IsUnknown() {
		req.SetDelay(float32(crawler.Delay.ValueFloat64()))
	}

	if !crawler.Depth.IsNull() && !crawler.Depth.IsUnknown() {
		req.SetDepth(int32(crawler.Depth.ValueInt64()))
	}

	if !crawler.MaxHits.IsNull() && !crawler.MaxHits.IsUnknown() {
		req.SetMaxHits(int32(crawler.MaxHits.ValueInt64()))
	}

	if !crawler.MaxHtml.IsNull() && !crawler.MaxHtml.IsUnknown() {
		req.SetMaxHtml(int32(crawler.MaxHtml.ValueInt64()))
	}

	if !crawler.MaxErrors.IsNull() && !crawler.MaxErrors.IsUnknown() {
		req.SetMaxErrors(int32(crawler.MaxErrors.ValueInt64()))
	}

	if !crawler.UserAgent.IsNull() && !crawler.UserAgent.IsUnknown() {
		req.SetUserAgent(crawler.UserAgent.ValueString())
	}

	// Status OK codes
	if !crawler.StatusOk.IsNull() && !crawler.StatusOk.IsUnknown() {
		statusOk := make([]int32, 0, len(crawler.StatusOk.Elements()))
		var statusOkInt64 []int64
		diags.Append(crawler.StatusOk.ElementsAs(ctx, &statusOkInt64, false)...)
		if !diags.HasError() {
			for _, v := range statusOkInt64 {
				statusOk = append(statusOk, int32(v))
			}
			req.SetStatusOk(statusOk)
		}
	}

	// Allowed domains
	if !crawler.AllowedDomains.IsNull() && !crawler.AllowedDomains.IsUnknown() {
		allowedDomains := make([]string, 0, len(crawler.AllowedDomains.Elements()))
		diags.Append(crawler.AllowedDomains.ElementsAs(ctx, &allowedDomains, false)...)
		if !diags.HasError() {
			req.SetAllowedDomains(allowedDomains)
		}
	}

	// Complex object fields - convert nested objects to API format
	if !crawler.Sitemap.IsNull() && !crawler.Sitemap.IsUnknown() {
		var sitemapEntries []resource_crawler.SitemapValue
		diags.Append(crawler.Sitemap.ElementsAs(ctx, &sitemapEntries, false)...)
		if !diags.HasError() {
			sitemapArray := make([]quantadmingo.V2CrawlerSitemapInner, len(sitemapEntries))
			for i, entry := range sitemapEntries {
				sitemapItem := quantadmingo.NewV2CrawlerSitemapInner()
				url := entry.Url.ValueString()
				recursive := entry.Recursive.ValueBool()
				sitemapItem.SetUrl(url)
				sitemapItem.SetRecursive(recursive)
				sitemapArray[i] = *sitemapItem
			}
			req.SetSitemap(sitemapArray)
		}
	}

	if !crawler.Assets.IsNull() && !crawler.Assets.IsUnknown() {
		// Assets.NetworkIntercept is a basetypes.ObjectValue - convert to NetworkInterceptValue
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

	// Update API call with built-in rate limiting and retry logic
	_, _, err := r.client.Instance.CrawlersAPI.CrawlersUpdate(
		ctx,
		r.client.Organization,
		crawler.Project.ValueString(),
		crawler.Uuid.ValueString(),
	).V2CrawlerRequest(req).Execute()

	if err != nil {
		diags.AddError("Unable to update crawler", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	// Post-update read to populate computed fields
	// Note: This may cause optional+computed fields to show as "(known after apply)" in plans
	// but is required for updates to succeed without errors
	return callCrawlerReadAPI(ctx, r, crawler)
}

func (r *crawlerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If there's no state (resource is being created) or no plan (resource is being deleted), return early
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	// Get the plan and state
	var plan, state resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if domain is changing
	domainChanging := !plan.Domain.Equal(state.Domain)

	// Preserve computed fields from state, with special handling for domain_verified
	if domainChanging {
		// When domain changes, the API will reset domain_verified to 0
		// Set this expectation in the plan to avoid inconsistent result errors
		plan.DomainVerified = types.Int64Value(0)
	} else {
		// Domain not changing, preserve existing domain_verified value
		plan.DomainVerified = state.DomainVerified
	}

	plan.CreatedAt = state.CreatedAt
	// Don't preserve UpdatedAt - let it be updated by the API response
	plan.Id = state.Id
	plan.ProjectId = state.ProjectId

	// Handle force_refresh - don't preserve from state, allow it to trigger updates
	// The force_refresh field is intentionally not preserved from state to allow changes

	// Preserve exclude if it's in the plan but not in the state
	if !plan.Exclude.IsNull() && !plan.Exclude.IsUnknown() && state.Exclude.IsNull() {
		// Keep the exclude from the plan
	} else if !state.Exclude.IsNull() && !state.Exclude.IsUnknown() {
		// If both have values, prefer the plan's value (which is the default behavior)
		// But if plan is empty and state has values, use state's values
		if len(plan.Exclude.Elements()) == 0 && len(state.Exclude.Elements()) > 0 {
			plan.Exclude = state.Exclude
		}
	}

	// Set the modified plan
	resp.Plan.Set(ctx, &plan)
}

// ImportState allows importing existing crawlers by UUID and project
func (r *crawlerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: "project:uuid" or just "uuid" (assuming default project)
	parts := strings.Split(req.ID, ":")
	var project, uuid string

	if len(parts) == 2 {
		project = parts[0]
		uuid = parts[1]
	} else if len(parts) == 1 {
		project = "default" // assume default project if not specified
		uuid = parts[0]
	} else {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID should be in format 'project:uuid' or just 'uuid' for default project",
		)
		return
	}

	var data resource_crawler.CrawlerModel
	data.Project = types.StringValue(project)
	data.Uuid = types.StringValue(uuid)

	// Read the crawler to populate all fields
	diags := callCrawlerReadAPI(ctx, r, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
