package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_crawler"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.in/yaml.v3"

	quantadmingo "github.com/quantcdn/quant-admin-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = (*crawlerResource)(nil)
	_ resource.ResourceWithConfigure = (*crawlerResource)(nil)
	_ resource.ResourceWithModifyPlan = (*crawlerResource)(nil)
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

	// Read the current state to get the UUID
	var state resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Preserve the UUID from the current state
	data.Uuid = state.Uuid

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

	crawler.Uuid = types.StringValue(api.GetUuid())

	return callCrawlerReadAPI(ctx, r, crawler)
}

func callCrawlerReadAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	// Add debug logging for client configuration
	bearerPreview := "not_set"
	if len(r.client.Bearer) > 10 {
		bearerPreview = r.client.Bearer[:10] + "..."
	}

	tflog.Debug(ctx, "Checking client configuration", map[string]interface{}{
		"has_auth_context": r.client.AuthContext != nil,
		"organization":     r.client.Organization,
		"has_bearer":      r.client.Bearer != "",
		"bearer_preview":  bearerPreview,
	})

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

	tflog.Debug(ctx, "Reading crawler", map[string]interface{}{
		"organization": r.client.Organization,
		"project":      crawler.Project.ValueString(),
		"uuid":         crawler.Uuid.ValueString(),
	})

	// Create the request and execute it
	api, resp, err := r.client.Instance.CrawlersAPI.CrawlersRead(ctx, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString()).Execute()

	if resp != nil {
		// Create a map of headers for logging
		headers := make(map[string]string)
		for k, v := range resp.Request.Header {
			if len(v) > 0 {
				headers[k] = v[0]
				// Redact most of the token value for security
				if k == "Authorization" {
					if len(v[0]) > 10 {
						headers[k] = v[0][:10] + "..."
					}
				}
			}
		}

		tflog.Debug(ctx, "Crawler read API response", map[string]interface{}{
			"url":         resp.Request.URL.String(),
			"method":      resp.Request.Method,
			"status_code": resp.StatusCode,
			"headers":     headers,
		})
	}

	if err != nil {
		diags.AddError(
			"Unable to load crawler", 
			fmt.Sprintf("Error: %s\nURL: %s\nStatus: %d", 
				err.Error(), 
				resp.Request.URL.String(), 
				resp.StatusCode,
			),
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

	// Parse and set config-related fields
	var config map[string]interface{}
	if api.Config != "" {
		crawler.Config = types.StringValue(api.GetConfig())
		if err := yaml.Unmarshal([]byte(api.GetConfig()), &config); err != nil {
			diags.AddWarning(
				"Unable to parse crawler config",
				fmt.Sprintf("Error parsing config YAML: %s. Some fields may not be set correctly.", err.Error()),
			)
		}
	}

	// Set browser_mode from config
	if browserMode, ok := config["browser_mode"]; ok && browserMode != nil {
		crawler.BrowserMode = types.BoolValue(browserMode.(bool))
	} else {
		crawler.BrowserMode = types.BoolValue(false)
	}

	// Handle exclude list consistently
	excludeVals := make([]attr.Value, 0)
	if config != nil {
		if excludeList, ok := config["exclude"]; ok && excludeList != nil {
			excludeStrs := excludeList.([]interface{})
			excludeVals = make([]attr.Value, len(excludeStrs))
			for i, v := range excludeStrs {
				if strVal, ok := v.(string); ok {
					excludeVals[i] = types.StringValue(strVal)
				}
			}
		}
	}

	// If we have no excludes from config but there are excludes in the plan/state, keep those
	if len(excludeVals) == 0 && !crawler.Exclude.IsNull() {
		var currentExcludes []string
		diags.Append(crawler.Exclude.ElementsAs(ctx, &currentExcludes, false)...)
		if len(currentExcludes) > 0 {
			excludeVals = make([]attr.Value, len(currentExcludes))
			for i, v := range currentExcludes {
				excludeVals[i] = types.StringValue(v)
			}
		}
	}

	crawler.Exclude = types.ListValueMust(types.StringType, excludeVals)

	// Set headers from config
	if config["headers"] != nil {
		headers := config["headers"].(map[string]interface{})
		headersMap := make(map[string]attr.Value)
		for k, v := range headers {
			headersMap[k] = types.StringValue(v.(string))
		}
		crawler.Headers = types.MapValueMust(types.StringType, headersMap)
	} else {
		crawler.Headers = types.MapNull(types.StringType)
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
	if crawler.Uuid.IsUnknown() || crawler.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing crawler.uuid attribute",
			"To read crawler information the crawler uuid must be provided",
		)
		return
	}

	if crawler.Project.IsNull() || crawler.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing crawler.project attribute",
			"To read crawler information the crawler project must be provided",
		)
		return
	}

	_, _, err := r.client.Instance.CrawlersAPI.CrawlersDelete(ctx, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString()).Execute()
	if err != nil {
		diags.AddError("Unable to delete crawler", fmt.Sprintf("Error: %s", err.Error()))
	}
	return diags
}

func callCrawlerUpdateAPI(ctx context.Context, r *crawlerResource, crawler *resource_crawler.CrawlerModel) (diags diag.Diagnostics) {
	if crawler.Uuid.IsUnknown() || crawler.Uuid.IsNull() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing crawler.uuid attribute",
			"To read crawler information the crawler uuid must be provided",
		)
		return
	}

	if crawler.Project.IsNull() || crawler.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing crawler.project attribute",
			"To read crawler information the crawler project must be provided",
		)
		return
	}

	req := *quantadmingo.NewCrawlerRequestUpdateWithDefaults()

	req.SetDomain(crawler.Domain.ValueString())
	req.SetBrowserMode(crawler.BrowserMode.ValueBool())

	urls := make([]string, 0, len(crawler.Urls.Elements()))
	diags.Append(crawler.Urls.ElementsAs(ctx, &urls, false)...)
	req.SetUrls(urls)

	exclude := make([]string, 0, len(crawler.Exclude.Elements()))
	diags.Append(crawler.Exclude.ElementsAs(ctx, &exclude, false)...)
	req.SetExclude(exclude)

	headers := make(map[string]string, len(crawler.Headers.Elements()))
	diags.Append(crawler.Headers.ElementsAs(ctx, &headers, false)...)
	req.SetHeaders(headers)

	_, _, err := r.client.Instance.CrawlersAPI.CrawlersUpdate(ctx, r.client.Organization, crawler.Project.ValueString(), crawler.Uuid.ValueString()).CrawlerRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError("Unable to update crawler", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	return callCrawlerReadAPI(ctx, r, crawler)
}

func (r *crawlerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If there's no state (resource is being created), return early
	if req.State.Raw.IsNull() {
		return
	}

	// Get the plan and state
	var plan, state resource_crawler.CrawlerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep the domain_verified value from the state
	plan.DomainVerified = state.DomainVerified

	// Set the modified plan
	resp.Plan.Set(ctx, &plan)
}
