package provider

import (
	"context"
	"os"
	"strconv"
	"terraform-provider-quant/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*quantProvider)(nil)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &quantProvider{}
	}
}

type quantProvider struct{}

type quantProviderModel struct {
	Bearer            types.String  `tfsdk:"bearer"`
	Organization      types.String  `tfsdk:"organization"`
	BaseURL           types.String  `tfsdk:"base_url"`
	RequestsPerSecond types.Float64 `tfsdk:"requests_per_second"`
	MaxRetries        types.Int64   `tfsdk:"max_retries"`
	BaseDelayMs       types.Int64   `tfsdk:"base_delay_ms"`
	MaxDelayMs        types.Int64   `tfsdk:"max_delay_ms"`
	EnableJitter      types.Bool    `tfsdk:"enable_jitter"`
}

func (p *quantProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bearer": schema.StringAttribute{
				MarkdownDescription: "The QuantCDN API Bearer token used to authenticate requests. Can also be set via QUANTCDN_API_TOKEN environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"organization": schema.StringAttribute{
				MarkdownDescription: "The QuantCDN organization machine name. Can also be set via QUANTCDN_ORGANIZATION environment variable.",
				Optional:            true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "The base URL for the QuantCDN API. Can also be set via QUANTCDN_BASE_URL environment variable.",
				Optional:            true,
			},
			"requests_per_second": schema.Float64Attribute{
				MarkdownDescription: "Maximum number of requests per second to send to the API. Defaults to 10.0",
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of retry attempts for failed requests. Defaults to 3",
				Optional:            true,
			},
			"base_delay_ms": schema.Int64Attribute{
				MarkdownDescription: "Base delay in milliseconds for exponential backoff. Defaults to 500",
				Optional:            true,
			},
			"max_delay_ms": schema.Int64Attribute{
				MarkdownDescription: "Maximum delay in milliseconds for exponential backoff. Defaults to 30000 (30 seconds)",
				Optional:            true,
			},
			"enable_jitter": schema.BoolAttribute{
				MarkdownDescription: "Whether to add random jitter to retry delays to avoid thundering herd. Defaults to true",
				Optional:            true,
			},
		},
	}
}

func (p *quantProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config quantProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If configuration values have been provided they must be known.
	if !config.Bearer.IsNull() && config.Bearer.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("bearer"),
			"Unknown QuantCDN API bearer token",
			"The provider cannot create the QuantCDN API Client as there is an unknown configuration value for the bearer token."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the QUANTCDN_API_TOKEN environment variable.",
		)
	}
	if !config.Organization.IsNull() && config.Organization.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization"),
			"Unknown QuantCDN organization",
			"The provider cannot create the QuantCDN API Client as there is an unknown configuration value for the organization."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the QUANTCDN_ORGANIZATION environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	bearer := os.Getenv("QUANTCDN_API_TOKEN")
	organization := os.Getenv("QUANTCDN_ORGANIZATION")
	baseURL := os.Getenv("QUANTCDN_BASE_URL")

	if !config.Bearer.IsNull() {
		bearer = config.Bearer.ValueString()
	}
	if !config.Organization.IsNull() {
		organization = config.Organization.ValueString()
	}
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if bearer == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("bearer"),
			"Missing QuantCDN API bearer token",
			"The provider cannot create the HashiCups API client as there is a missing or empty value for the QuantCDN API bearer token. "+
				"Set the host value in the configuration or use the QUANTCDN_API_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if organization == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization"),
			"Missing QuantCDN organization",
			"The provider cannot create the HashiCups API client as there is a missing or empty value for the QuantCDN API organization. "+
				"Set the host value in the configuration or use the QUANTCDN_ORGANIZATION environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Build rate limiting configuration
	rateLimitConfig := client.DefaultRateLimitConfig()

	// Apply configuration overrides with environment variable fallbacks
	if !config.RequestsPerSecond.IsNull() {
		rateLimitConfig.RequestsPerSecond = config.RequestsPerSecond.ValueFloat64()
	} else if envVal := os.Getenv("QUANTCDN_REQUESTS_PER_SECOND"); envVal != "" {
		if val, err := strconv.ParseFloat(envVal, 64); err == nil {
			rateLimitConfig.RequestsPerSecond = val
		}
	}

	if !config.MaxRetries.IsNull() {
		rateLimitConfig.MaxRetries = int(config.MaxRetries.ValueInt64())
	} else if envVal := os.Getenv("QUANTCDN_MAX_RETRIES"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil {
			rateLimitConfig.MaxRetries = val
		}
	}

	if !config.BaseDelayMs.IsNull() {
		rateLimitConfig.BaseDelay = time.Duration(config.BaseDelayMs.ValueInt64()) * time.Millisecond
	} else if envVal := os.Getenv("QUANTCDN_BASE_DELAY_MS"); envVal != "" {
		if val, err := strconv.ParseInt(envVal, 10, 64); err == nil {
			rateLimitConfig.BaseDelay = time.Duration(val) * time.Millisecond
		}
	}

	if !config.MaxDelayMs.IsNull() {
		rateLimitConfig.MaxDelay = time.Duration(config.MaxDelayMs.ValueInt64()) * time.Millisecond
	} else if envVal := os.Getenv("QUANTCDN_MAX_DELAY_MS"); envVal != "" {
		if val, err := strconv.ParseInt(envVal, 10, 64); err == nil {
			rateLimitConfig.MaxDelay = time.Duration(val) * time.Millisecond
		}
	}

	if !config.EnableJitter.IsNull() {
		rateLimitConfig.EnableJitter = config.EnableJitter.ValueBool()
	} else if envVal := os.Getenv("QUANTCDN_ENABLE_JITTER"); envVal != "" {
		if val, err := strconv.ParseBool(envVal); err == nil {
			rateLimitConfig.EnableJitter = val
		}
	}

	// Create client with rate limiting configuration
	var c *client.Client
	if baseURL != "" {
		c = client.NewWithRateLimitAndBaseURL(bearer, organization, rateLimitConfig, baseURL)
	} else {
		c = client.NewWithRateLimit(bearer, organization, rateLimitConfig)
	}

	// Make the SDK client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *quantProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "quant"
}

func (p *quantProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectsDataSource,
		NewProjectDataSource,
	}
}

func (p *quantProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewDomainResource,
		NewRuleProxyResource,
		NewRuleRedirectResource,
		NewRuleCustomResponseResource,
		NewRuleContentFilterResource,
		NewCrawlerResource,
		NewCrawlerScheduleResource,
		NewHeaderResource,
	}
}
