package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = &crawlerResource{}
	_ resource.ResourceWithConfigure = &crawlerResource{}
)

func NewCrawlerResource() resource.Resource {
	return &crawlerResource{}
}

type crawlerResource struct {
	client *client.Client
}

type crawlerResourceModel struct {
	UUID        types.String    `tfsdk:"uuid"`
	Project     types.String    `tfsdk:"uuid"`
	Name        types.String    `tfsdk:"uuid"`
	Domain      types.String    `tfsdk:"domain"`
	BrowserMode types.Bool      `tfsdk:"browser_mode"`
	UrlList     []types.String  `tfsdk:"url_list"`
	Headers     []CrawlerHeader `tfsdk:"headers"`
}

type CrawlerHeader struct {
	Header types.String `tfsdk:"header"`
	Value  types.String `tfsdk:"value"`
}

func (r *crawlerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *crawlerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crawler"
}

func (r *crawlerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier",
				Computed:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The project machine name",
				Required:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain to crawl",
				Required:            true,
			},
			"browser_mode": schema.BoolAttribute{
				MarkdownDescription: "Use a full headless browser during the crawl",
				Optional:            true,
			},
			"url_list": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of URLs to include in the crawl",
			},
			"headers": schema.SetNestedAttribute{
				MarkdownDescription: "A list of headers to add to each request",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"header": schema.StringAttribute{
							MarkdownDescription: "The header key",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "The header value",
							Required:            true,
						},
					},
				},
				Optional: true,
			},
		},
	}
}

func (r *crawlerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan crawlerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := r.client.Organization
	project := plan.Project.ValueString()
	client := r.client.Admin.CrawlersAPI

	// Prepare the resource.
	c := *openapi.NewCrawlerRequest()
	c.SetDomain(plan.Domain.ValueString())
	c.SetBrowserMode(plan.BrowserMode.ValueBool())

	var urls []string
	for _, url := range plan.UrlList {
		urls = append(urls, url.ValueString())
	}
	c.SetUrlList(urls)

	var headers map[string]interface{}
	for _, header := range plan.Headers {
		headers[header.Header.ValueString()] = header.Value.ValueString()
	}
	c.SetHeaders(headers)

	res, _, err := client.OrganizationsOrganizationProjectsProjectCrawlersPost(r.client.Auth, organization, project).CrawlerRequest(c).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating crawler",
			"Could not create crawler for project, unexpected error: "+err.Error(),
		)
		return
	}

	plan.UUID = types.StringValue(*res.Data.Crawlers[0].Uuid)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *crawlerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state crawlerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.UUID.IsNull() || state.UUID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Error reading crawler data",
			"Could not read crawler data, crawler UUID unkown",
		)
		return
	}

	organization := r.client.Organization
	project := state.Project.ValueString()

	client := r.client.Admin.CrawlersAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectCrawlersCrawlerGet(r.client.Auth, organization, project, state.UUID.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading crawler data",
			"Could not match crawler with uuid "+state.UUID.ValueString()+": "+err.Error(),
		)
		return
	}

	crawler := res.Data.Crawlers[0]

	state.Domain = types.StringValue(crawler.GetDomain())

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *crawlerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan crawlerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.UUID.IsNull() || plan.UUID.IsUnknown() {
		resp.Diagnostics.AddError(
			"Error reading crawler data",
			"Could not read crawler data, crawler UUID unkown",
		)
		return
	}

	cr := openapi.NewCrawlerRequest()
	cr.SetDomain(plan.Domain.ValueString())
	for _, header := range plan.Headers {
		cr.Headers[header.Header.ValueString()] = header.Value.ValueString()
	}
	for _, url := range plan.UrlList {
		cr.UrlList = append(cr.UrlList, url.ValueString())
	}

	cr.SetName(plan.Name.ValueString())

}

func (r *crawlerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state crawlerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.UUID.IsNull() {
		resp.Diagnostics.AddError(
			"Error Deleting Quant crawler",
			"Invalid state: crawer uuid is unknown.",
		)
		return
	}

	organization := r.client.Organization
	project := state.UUID.ValueString()

	client := r.client.Admin.CrawlersAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectCrawlersCrawlerDelete(r.client.Auth, organization, project, state.UUID.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Quant crawler",
			"Could not delete crawler, unexpected error: "+err.Error(),
		)
		return
	}

}
