package provider

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_crawler"

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
	_ resource.ResourceWithModifyPlan = (*crawlerResource)(nil)
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
	req.SetExclude(exclude)  // Always set exclude, even if empty

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
			
			// Handle headers
			if len(parsedConfig.Config.Headers) > 0 {
				headersMap := make(map[string]attr.Value)
				for k, v := range parsedConfig.Config.Headers {
					headersMap[k] = types.StringValue(v)
				}
				crawler.Headers = types.MapValueMust(types.StringType, headersMap)
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

	// Only set fields that are not null or unknown
	if !crawler.Domain.IsNull() && !crawler.Domain.IsUnknown() {
		req.SetDomain(crawler.Domain.ValueString())
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

	// Preserve computed fields from state
	plan.DomainVerified = state.DomainVerified
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = state.UpdatedAt
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


