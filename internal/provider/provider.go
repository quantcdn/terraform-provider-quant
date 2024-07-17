package provider

import (
	"context"
	"os"
	"terraform-provider-quant/internal/client"

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
	Bearer types.String `tfsdk:"bearer"`
	Organization types.String `tfsdk:"organization"`
}

func (p *quantProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bearer": schema.StringAttribute{
				MarkdownDescription: "The API Bearer token",
				Optional: true,
			},
			"organization": schema.StringAttribute{
				MarkdownDescription: "Organization machine name",
				Optional: true,
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
	if config.Bearer.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("bearer"),
			"Unknown QuantCDN API bearer token",
			"The provider cannot create the QuantCDN API Client as there is an unnknown configuration value for the bearer token."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the QUANTCDN_API_TOKEN environment variable.",
		)
	}
	if config.Organization.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization"),
			"Unknown QuantCDN organization",
			"The provider cannot create the QuantCDN API Client as there is an unnknown configuration value for the organization."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the QUANTCDN_ORGANIZATION environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	bearer := os.Getenv("QUANTCDN_API_TOKEN")
	organization := os.Getenv("QUANTCDN_ORGANIZATION")

	if !config.Bearer.IsNull() {
		bearer = config.Bearer.ValueString()
	}
	if !config.Organization.IsNull() {
		organization = config.Organization.ValueString()
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

	c := client.New(bearer, organization)

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
	}
}

func (p *quantProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewHeaderResource,
		NewRuleProxyResource,
		NewRuleRedirectResource,
	}
}
