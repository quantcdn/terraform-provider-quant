package provider

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_crawler"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.in/yaml.v3"

	quantadmingo "github.com/quantcdn/quant-admin-go"
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
	req := *quantadmingo.NewCrawlerRequestWithDefaults()

	req.SetBrowserMode(crawler.BrowserMode.ValueBool())
	req.SetDomain(crawler.Domain.ValueString())
	req.SetName(crawler.Name.ValueString())

	// Initialize exclude with an empty list by default
	exclude := make([]string, 0)

	// Only set exclude if it's provided
	if !crawler.Exclude.IsNull() {
		diags.Append(crawler.Exclude.ElementsAs(ctx, &exclude, false)...)
	}
	req.SetExclude(exclude) // Always set exclude, even if empty

	// Set headers if provided
	if !crawler.Headers.IsNull() {
		headers := make(map[string]string, len(crawler.Headers.Elements()))
		diags.Append(crawler.Headers.ElementsAs(ctx, &headers, false)...)
		req.SetHeaders(headers)
	}

	api, _, err := r.client.Instance.CrawlersAPI.CrawlersCreate(r.client.AuthContext, r.client.Organization, crawler.Project.ValueString()).CrawlerRequest(req).Execute()

	if err != nil {
		diags.AddError(
			"Unable to create crawler",
			fmt.Sprintf("Error: %s", err.Error()),
		)
		return diags
	}

	// Set the UUID and ID from the API response
	crawler.Uuid = types.StringValue(api.GetUuid())
	crawler.Id = types.Int64Value(int64(api.GetId()))

	// Use retry logic for post-create read to handle eventual consistency
	// No domain validation needed for create since we're not changing existing values
	return callCrawlerReadAPIWithRetry(ctx, r, crawler, "")
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
	crawler.CreatedAt = types.StringValue(api.GetCreatedAt())
	crawler.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	// Set organization to the current organization
	crawler.Organization = types.StringValue(r.client.Organization)

	// Set deleted_at (null if not deleted)
	if api.DeletedAt != nil {
		crawler.DeletedAt = types.StringValue(*api.DeletedAt)
	} else {
		crawler.DeletedAt = types.StringNull()
	}

	// Improved approach with better error handling and structure
	if api.Config != "" {
		crawler.Config = types.StringValue(api.GetConfig())

		// Define a structured type for the config
		type CrawlerConfig struct {
			Config struct {
				UserAgent   string                 `yaml:"user_agent"`
				BrowserMode bool                   `yaml:"browser_mode"`
				Workers     int                    `yaml:"workers"`
				Depth       int                    `yaml:"depth"`
				MaxHits     int                    `yaml:"max_hits"`
				MaxHtml     int                    `yaml:"max_html"`
				Cache       bool                   `yaml:"cache"`
				Delay       int                    `yaml:"delay"`
				StatusOk    []int                  `yaml:"status_ok"`
				Quant       map[string]interface{} `yaml:"quant"`
				StartUrl    []string               `yaml:"start_url"`
				Headers     map[string]string      `yaml:"headers"`
				Exclude     []string               `yaml:"exclude"`
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
			// Set fields directly from the structured config
			crawler.BrowserMode = types.BoolValue(parsedConfig.Config.BrowserMode)

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

			// Initialize urls from start_url in config
			if len(parsedConfig.Config.StartUrl) > 0 {
				urlVals := make([]attr.Value, len(parsedConfig.Config.StartUrl))
				for i, v := range parsedConfig.Config.StartUrl {
					urlVals[i] = types.StringValue(v)
				}
				crawler.Urls = types.ListValueMust(types.StringType, urlVals)
			} else {
				// Always set an empty list rather than null
				crawler.Urls = types.ListValueMust(types.StringType, []attr.Value{})
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
	_, _, err := r.client.Instance.CrawlersAPI.CrawlersDelete(
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

	req := *quantadmingo.NewCrawlerRequestUpdateWithDefaults()

	// Store expected values for consistency validation
	expectedDomain := ""
	if !crawler.Domain.IsNull() && !crawler.Domain.IsUnknown() {
		expectedDomain = crawler.Domain.ValueString()
		req.SetDomain(expectedDomain)
	}

	if !crawler.BrowserMode.IsNull() && !crawler.BrowserMode.IsUnknown() {
		req.SetBrowserMode(crawler.BrowserMode.ValueBool())
	}

	// Only process URLs if the list is not null
	if !crawler.Urls.IsNull() && !crawler.Urls.IsUnknown() {
		urls := make([]string, 0, len(crawler.Urls.Elements()))
		diags.Append(crawler.Urls.ElementsAs(ctx, &urls, false)...)
		if !diags.HasError() {
			req.SetUrls(urls)
		}
	}

	// Only process exclude if the list is not null
	if !crawler.Exclude.IsNull() && !crawler.Exclude.IsUnknown() {
		exclude := make([]string, 0, len(crawler.Exclude.Elements()))
		diags.Append(crawler.Exclude.ElementsAs(ctx, &exclude, false)...)
		if !diags.HasError() {
			req.SetExclude(exclude)
		}
	}

	// Only process headers if the map is not null
	if !crawler.Headers.IsNull() && !crawler.Headers.IsUnknown() {
		headers := make(map[string]string, len(crawler.Headers.Elements()))
		diags.Append(crawler.Headers.ElementsAs(ctx, &headers, false)...)
		if !diags.HasError() {
			req.SetHeaders(headers)
		}
	}

	// Update API call with built-in rate limiting and retry logic
	_, _, err := r.client.Instance.CrawlersAPI.CrawlersUpdate(
		ctx,
		r.client.Organization,
		crawler.Project.ValueString(),
		crawler.Uuid.ValueString(),
	).CrawlerRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError("Unable to update crawler", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	// Use retry logic for post-update read to handle eventual consistency
	return callCrawlerReadAPIWithRetry(ctx, r, crawler, expectedDomain)
}

// callCrawlerReadAPIWithRetry implements retry logic for eventual consistency after updates
// This follows the same patterns as our HTTP client retry logic but at the application level
func callCrawlerReadAPIWithRetry(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel, expectedDomain string) (diags diag.Diagnostics) {
	// Use the same retry configuration as our HTTP client for consistency
	config := r.client.GetRateLimitConfig()
	if config == nil {
		// Fallback to default retry configuration
		config = &client.RateLimitConfig{
			MaxRetries:   3,
			BaseDelay:    500 * time.Millisecond,
			MaxDelay:     5 * time.Second,
			EnableJitter: true,
		}
	}

	var lastDiags diag.Diagnostics

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Add delay before retry (except for first attempt)
		if attempt > 0 {
			delay := calculateRetryDelay(attempt-1, config)

			// Respect context cancellation during delay
			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				diags.AddError("Context cancelled during retry", ctx.Err().Error())
				return diags
			}
		}

		// Attempt to read the crawler
		diags = callCrawlerReadAPI(ctx, r, crawler)
		if diags.HasError() {
			// If there's an actual error (not just inconsistent data), return immediately
			return diags
		}

		// Check for consistency if we're validating domain updates
		if expectedDomain != "" {
			actualDomain := crawler.Domain.ValueString()
			if actualDomain == expectedDomain {
				// Consistency achieved, return success
				return diags
			}

			// Store the inconsistent result for potential final return
			lastDiags = diags

			// Log the inconsistency for debugging (this will appear in Terraform logs)
			if attempt < config.MaxRetries {
				// Don't log on the last attempt to avoid noise if we're going to fail anyway
				continue
			}
		} else {
			// No specific validation needed, return success
			return diags
		}
	}

	// All retries exhausted and still inconsistent
	diags.AddWarning(
		"Eventual consistency timeout",
		fmt.Sprintf("After %d attempts, the API still returns domain=%s instead of expected domain=%s. This may indicate a backend issue or longer than expected eventual consistency delay.",
			config.MaxRetries+1,
			crawler.Domain.ValueString(),
			expectedDomain,
		),
	)

	// Return the last read result - Terraform will detect the inconsistency and may retry the entire operation
	return lastDiags
}

// calculateRetryDelay calculates delay using exponential backoff with jitter
// This mirrors the logic in our HTTP client's retry mechanism
func calculateRetryDelay(attempt int, config *client.RateLimitConfig) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt)))

	// Cap at maximum delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Add jitter if enabled
	if config.EnableJitter {
		// Add up to 25% jitter to avoid thundering herd
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
		delay += jitter
	}

	return delay
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

	// Preserve computed fields from state
	plan.DomainVerified = state.DomainVerified
	plan.CreatedAt = state.CreatedAt
	// Don't preserve UpdatedAt - let it be updated by the API response
	plan.Id = state.Id
	plan.ProjectId = state.ProjectId

	// If UUID is not set in plan but exists in state, preserve it
	if plan.Uuid.IsNull() && !state.Uuid.IsNull() {
		plan.Uuid = state.Uuid
	}

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
