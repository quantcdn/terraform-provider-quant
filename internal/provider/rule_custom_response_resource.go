package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &rulecustomResponse{}
	_ resource.ResourceWithConfigure = &rulecustomResponse{}
)

// NewrulecustomResponse is a helper function to simplify the provider implementation.
func NewRulecustomResponseResource() resource.Resource {
	return &rulecustomResponse{}
}

// rulecustomResponse is the resource implementation.
type rulecustomResponse struct {
	client *client.Client
}

type rulecustomResponseModel struct {
	Name     types.String `tfsdk:"name"`
	Uuid     types.String `tfsdk:"uuid"`
	Project  types.String `tfsdk:"project"`
	Disabled types.Bool   `tfsdk:"disabled"`

	Domain types.String `tfsdk:"domain"`

	// Rule selection.
	CountryInclude types.Bool     `tfsdk:"country_include"`
	Countries      []types.String `tfsdk:"countries"`
	MethodInclude  types.Bool     `tfsdk:"method_include"`
	Methods        []types.String `tfsdk:"methods"`
	IpInclude      types.Bool     `tfsdk:"ip_include"`
	Ips            []types.String `tfsdk:"ips"`
	OnlyWithCookie types.String   `tfsdk:"only_with_cookie"`

	// Rule details
	CustomResponseStatusCode types.Int64  `tfsdk:"custom_response_status_code"`
	CustomResponseBody       types.String `tfsdk:"custom_response_body"`
}

// Configure adds the provider configured client to the resource.
func (r *rulecustomResponse) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *hashicups.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *rulecustomResponse) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_custom_response"
}

// Schema defines the schema for the resource.
func (r *rulecustomResponse) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "A name for the rule",
				Optional:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The project machine name",
				Required:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "If this rule is disabled",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain the rule applies to",
				Optional:            true,
			},
			"countries": schema.ListAttribute{
				MarkdownDescription: "A list of countries",
				Optional:            true,
			},
			"country_include": schema.BoolAttribute{
				MarkdownDescription: "Include the country list",
				Optional:            true,
			},
			"methods": schema.ListAttribute{
				MarkdownDescription: "A list of HTTP methods",
				Optional:            true,
			},
			"method_include": schema.BoolAttribute{
				MarkdownDescription: "Include the methods",
				Optional:            true,
			},
			"ips": schema.ListAttribute{
				MarkdownDescription: "A list of IP addresses",
				Optional:            true,
			},
			"ip_include": schema.BoolAttribute{
				MarkdownDescription: "Include hte IP addresses",
				Optional:            true,
			},
			"only_with_cookie": schema.StringAttribute{
				MarkdownDescription: "Apply rule only if the cookie is present",
				Optional:            true,
			},
			"custom_response_status_code": schema.Int64Attribute{
				MarkdownDescription: "HTTP status code for the response",
				Required:            true,
			},
			"custom_response_body": schema.StringAttribute{
				MarkdownDescription: "Response body",
				Required:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *rulecustomResponse) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rulecustomResponseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CountryInclude.IsNull() && plan.IpInclude.IsNull() && plan.MethodInclude.IsNull() {
		resp.Diagnostics.AddError(
			"Rule criteria is missing",
			"Could not crete a rule due to missing criteria; must provide country, ip and/or method",
		)
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *rulecustomResponse) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rulecustomResponseModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := r.client.Organization
	project := state.Project.ValueString()

	client := r.client.Admin.RulesAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectRuleGet(context.Background(), organization, project, state.Uuid.ValueString()).Execute()
	rule := res.Data.Rules[0]

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating HashiCups Order",
			"Could not update order, unexpected error: "+err.Error(),
		)
	}

	// @todo — move more from Rule.config to the state.
	state.Uuid = types.StringValue(*rule.Uuid)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *rulecustomResponse) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rulecustomResponseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *rulecustomResponse) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rulecustomResponseModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Project.IsNull() {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Invalid state: project machine name is unknown.",
		)
		return
	}

	organization := r.client.Organization
	project := state.Project.ValueString()

	client := r.client.Admin.RulesAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesAuthRuleDelete(r.client.Auth, organization, project, state.Uuid.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}
