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

	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = (*crawlerResource)(nil)
	_ resource.ResourceWithConfigure = (*crawlerResource)(nil)
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
	req := *openapi.NewCrawlerRequestWithDefaults()

	req.SetBrowserMode(crawler.BrowserMode.ValueBool())
	req.SetDomain(crawler.Domain.ValueString())
	req.SetName(crawler.Name.ValueString())

	if !crawler.Urls.IsNull() {
		urls := make([]string, 0, len(crawler.Urls.Elements()))
		diags.Append(crawler.Urls.ElementsAs(ctx, &urls, false)...)
		req.SetUrls(urls)
	}

	if !crawler.Exclude.IsNull() {
		exclude := make([]string, 0, len(crawler.Exclude.Elements()))
		diags.Append(crawler.Exclude.ElementsAs(ctx, &exclude, false)...)
		req.SetExclude(exclude)
	}

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

	org := r.client.Organization
	if !crawler.Organization.IsNull() {
		org = crawler.Organization.ValueString()
	}

	api, _, err := r.client.Instance.CrawlersAPI.CrawlersRead(ctx, org, crawler.Project.ValueString(), crawler.Uuid.ValueString()).Execute()
	if err != nil {
		diags.AddError("Unable to load crawler", fmt.Sprintf("Error: %s", err.Error()))
		return diags
	}

	var config map[string]interface{}

	if api.Config != "" {
		crawler.Config = types.StringValue(api.GetConfig())
		if err := yaml.Unmarshal([]byte(api.GetConfig()), &config); err != nil {
			// Continue...
		}
	}

	crawler.Uuid = types.StringValue(api.GetUuid())
	crawler.Name = types.StringValue(api.GetName())
	crawler.Domain = types.StringValue(api.GetDomain())
	crawler.DomainVerified = types.Int64Value(int64(api.GetDomainVerified()))
	crawler.BrowserMode = types.BoolValue(config["browser_mode"].(bool))
	crawler.CreatedAt = types.StringValue(api.GetCreatedAt())
	crawler.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	if config["exclude"] != nil {
		excludeStrs := config["exclude"].([]interface{})
		excludeVals := make([]attr.Value, len(excludeStrs))
		for i, v := range excludeStrs {
			excludeVals[i] = types.StringValue(v.(string))
		}
		crawler.Exclude = types.ListValueMust(types.StringType, excludeVals)
	}

	if config["headers"] != nil {
		headers := config["headers"].(map[string]interface{})
		headersMap := make(map[string]attr.Value)
		for k, v := range headers {
			headersMap[k] = types.StringValue(v.(string))
		}
		crawler.Headers = types.MapValueMust(types.StringType, headersMap)
	}

	crawler.UrlsList = types.StringNull()

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

	org := r.client.Organization
	if !crawler.Organization.IsNull() {
		org = crawler.Organization.ValueString()
	}

	_, _, err := r.client.Instance.CrawlersAPI.CrawlersDelete(ctx, org, crawler.Project.ValueString(), crawler.Uuid.ValueString()).Execute()
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

	org := r.client.Organization
	if !crawler.Organization.IsNull() {
		org = crawler.Organization.ValueString()
	}

	req := *openapi.NewCrawlerRequestUpdateWithDefaults()

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

	_, _, err := r.client.Instance.CrawlersAPI.CrawlersUpdate(ctx, org, crawler.Project.ValueString(), crawler.Uuid.ValueString()).CrawlerRequestUpdate(req).Execute()
	if err != nil {
		diags.AddError("Unable to update crawler", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	return callCrawlerReadAPI(ctx, r, crawler)
}
