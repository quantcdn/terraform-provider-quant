package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &ruleHeaders{}
	_ resource.ResourceWithConfigure = &ruleHeaders{}
)

// NewruleHeaders is a helper function to simplify the provider implementation.
func NewRuleHeadersResource() resource.Resource {
	return &ruleHeaders{}
}

// ruleHeaders is the resource implementation.
type ruleHeaders struct {
	client *client.Client
}

type ruleHeadersModel struct {
	Name     types.String `tfsdk:"name"`
	Uuid     types.String `tfsdk:"uuid"`
	Project  types.String `tfsdk:"project"`
	Disabled types.Bool   `tfsdk:"disabled"`

	Domain types.String `tfsdk:"domain"`
	Url    types.String `tfsdk:"url"`

	// Rule selection.
	CountryInclude types.Bool     `tfsdk:"country_include"`
	Countries      []types.String `tfsdk:"countries"`
	MethodInclude  types.Bool     `tfsdk:"method_include"`
	Methods        []types.String `tfsdk:"methods"`
	IpInclude      types.Bool     `tfsdk:"ip_include"`
	Ips            []types.String `tfsdk:"ips"`
	OnlyWithCookie types.String   `tfsdk:"only_with_cookie"`

	Headers map[string]interface{} `tfsdk:"headeres"`
}

// Configure adds the provider configured client to the resource.
func (r *ruleHeaders) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ruleHeaders) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_authentication"
}

// Schema defines the schema for the resource.
func (r *ruleHeaders) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ruleHeaders) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleHeadersModel
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

	proxy := openapi.NewRuleProxyRequest()

	proxy.SetName(plan.Name.ValueString())
	proxy.SetDisabled(plan.Disabled.ValueBool())
	proxy.SetDomain(plan.Domain.ValueString())

	if !plan.CountryInclude.IsNull() && !plan.CountryInclude.IsUnknown() {
		if plan.CountryInclude.ValueBool() {
			proxy.SetCountryIs(helpers.FlattenToStrings(plan.Countries))
		} else {
			proxy.SetCountryIsNot(helpers.FlattenToStrings(plan.Countries))
		}
	}

	if !plan.MethodInclude.IsNull() && !plan.MethodInclude.IsUnknown() {
		if plan.MethodInclude.ValueBool() {
			proxy.SetMethodIs(helpers.FlattenToStrings(plan.Methods))
		} else {
			proxy.SetMethodIsNot(helpers.FlattenToStrings(plan.Methods))
		}
	}

	if !plan.IpInclude.IsNull() && !plan.IpInclude.IsUnknown() {
		if plan.IpInclude.ValueBool() {
			proxy.SetIpIs(helpers.FlattenToStrings(plan.Ips))
		} else {
			proxy.SetIpIsNot(helpers.FlattenToStrings(plan.Ips))
		}
	}

	// Rule behaviour.
	proxy.SetUrl(plan.Url.ValueString())

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ruleHeaders) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleHeadersModel
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
func (r *ruleHeaders) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleHeadersModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ruleHeaders) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleHeadersModel
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
