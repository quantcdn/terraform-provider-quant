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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &quantProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &quantProvider{
			version: version,
		}
	}
}

// quantProvider is the provider implementation.
type quantProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type quantProviderModel struct {
	SecretToken  types.String `tfsdk:"secret_token"`
	Organization types.String `tfsdk:"organization"`
}

// Metadata returns the provider type name.
func (p *quantProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "quant"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *quantProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"secret_token": schema.StringAttribute{
				MarkdownDescription: "An API token scoped to the organisation.",
				Optional:            true,
			},
			"organization": schema.StringAttribute{
				MarkdownDescription: "The organization to manage.",
				Optional:            true,
			},
		},
	}
}

// Configure prepares a Quant API client for data sources and resources.
func (p *quantProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Prepare our local configuration object.
	var config quantProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if config.SecretToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_token"),
			"Unknown Quant secret token.",
			"The provider cannot create the API client. Please set statically or use the QUANT_SECRET_TOKEN environment variable.",
		)
	}

	if config.Organization.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization"),
			"Uknown Quant organization.",
			"The provider cannot create the API client. Please set statically or use the QUANT_ORGANIZATION environment variable.",
		)
	}

	secret := os.Getenv("QUANT_SECRET_TOKEN")
	organization := os.Getenv("QUANT_ORGANIZATION")

	if secret == "" {
		secret = config.SecretToken.ValueString()
	}
	if organization == "" {
		organization = config.Organization.ValueString()
	}

	// Create a new client instance.
	client := client.New(secret, organization)

	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *quantProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectsDataSource,
	}

}

// Resources defines the resources implemented in the provider.
func (p *quantProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
	}
}
