package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &ruleRedirectResource{}
	_ resource.ResourceWithConfigure = &ruleRedirectResource{}
)

// NewruleRedirectResource is a helper function to simplify the provider implementation.
func NewRuleRedirectResouce() resource.Resource {
	return &ruleRedirectResource{}
}

// ruleRedirectResource is the resource implementation.
type ruleRedirectResource struct {
	client *client.Client
}

type ruleRedirectResourceModel struct {
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

	// Rule behaviour.
	Url        types.String `tfsdk:"url"`
	To         types.String `tfsdk:"to"`
	StatusCode types.Int64  `tfsdk:"status_code"`
}

// Configure adds the provider configured client to the resource.
func (r *ruleRedirectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ruleRedirectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_redirect"
}

// Schema defines the schema for the resource.
func (r *ruleRedirectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"url": schema.StringAttribute{
				MarkdownDescription: "URL to redirect from",
				Required:            true,
			},
			"to": schema.StringAttribute{
				MarkdownDescription: "Address to redirect to",
				Required:            true,
			},
			"status_code": schema.Int64Attribute{
				MarkdownDescription: "The redirect code",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(301),
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ruleRedirectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleRedirectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := r.client.Organization
	project := plan.Project.ValueString()

	rule := openapi.NewRuleRedirectRequest()

	if plan.CountryInclude.IsNull() && plan.IpInclude.IsNull() && plan.MethodInclude.IsNull() {
		resp.Diagnostics.AddError(
			"Rule criteria is missing",
			"Could not crete a rule due to missing criteria; must provide country, ip and/or method",
		)
	}

	if !plan.CountryInclude.IsNull() && !plan.CountryInclude.IsUnknown() {
		if plan.CountryInclude.ValueBool() {
			rule.SetCountryIs(helpers.FlattenToStrings(plan.Countries))
		} else {
			rule.SetCountryIsNot(helpers.FlattenToStrings(plan.Countries))
		}
	}

	if !plan.MethodInclude.IsNull() && !plan.MethodInclude.IsUnknown() {
		if plan.MethodInclude.ValueBool() {
			rule.SetMethodIs(helpers.FlattenToStrings(plan.Methods))
		} else {
			rule.SetMethodIsNot(helpers.FlattenToStrings(plan.Methods))
		}
	}

	if !plan.IpInclude.IsNull() && !plan.IpInclude.IsUnknown() {
		if plan.IpInclude.ValueBool() {
			rule.SetIpIs(helpers.FlattenToStrings(plan.Ips))
		} else {
			rule.SetIpIsNot(helpers.FlattenToStrings(plan.Ips))
		}
	}

	if plan.StatusCode.ValueInt64() != 301 || plan.StatusCode.ValueInt64() != 302 {
		resp.Diagnostics.AddError(
			"Invalid redirect status code",
			fmt.Sprintf("Redirect code must be [301, 302] %d given", plan.StatusCode.ValueInt64()),
		)
		return
	}

	rule.SetName(plan.Name.ValueString())
	rule.SetDomain(plan.Domain.ValueString())
	rule.SetUrl(plan.Url.ValueString())
	rule.SetTo(plan.To.ValueString())
	rule.SetStatusCode(int32(plan.StatusCode.ValueInt64()))

	client := r.client.Admin.RulesAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectPost(context.Background(), organization, project).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error create redirect rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
	}

	plan.Uuid = types.StringValue(*res.Data.Rules[0].Uuid)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ruleRedirectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleRedirectResourceModel
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
			"Error reading rule",
			"Could not read rule, unexpected error: "+err.Error(),
		)
	}

	// @todo — move more from Rule.config to the state.
	state.Uuid = types.StringValue(*rule.Uuid)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ruleRedirectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleRedirectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule := openapi.NewRuleRedirectRequest()

	if plan.CountryInclude.IsNull() && plan.IpInclude.IsNull() && plan.MethodInclude.IsNull() {
		resp.Diagnostics.AddError(
			"Rule criteria is missing",
			"Could not crete a rule due to missing criteria; must provide country, ip and/or method",
		)
	}

	if !plan.CountryInclude.IsNull() && !plan.CountryInclude.IsUnknown() {
		if plan.CountryInclude.ValueBool() {
			rule.SetCountryIs(helpers.FlattenToStrings(plan.Countries))
		} else {
			rule.SetCountryIsNot(helpers.FlattenToStrings(plan.Countries))
		}
	}

	if !plan.MethodInclude.IsNull() && !plan.MethodInclude.IsUnknown() {
		if plan.MethodInclude.ValueBool() {
			rule.SetMethodIs(helpers.FlattenToStrings(plan.Methods))
		} else {
			rule.SetMethodIsNot(helpers.FlattenToStrings(plan.Methods))
		}
	}

	if !plan.IpInclude.IsNull() && !plan.IpInclude.IsUnknown() {
		if plan.IpInclude.ValueBool() {
			rule.SetIpIs(helpers.FlattenToStrings(plan.Ips))
		} else {
			rule.SetIpIsNot(helpers.FlattenToStrings(plan.Ips))
		}
	}

	if plan.StatusCode.ValueInt64() != 301 || plan.StatusCode.ValueInt64() != 302 {
		resp.Diagnostics.AddError(
			"Invalid redirect status code",
			fmt.Sprintf("Redirect code must be [301, 302] %d given", plan.StatusCode.ValueInt64()),
		)
		return
	}

	rule.SetName(plan.Name.ValueString())
	rule.SetDomain(plan.Domain.ValueString())
	rule.SetUrl(plan.Url.ValueString())
	rule.SetTo(plan.To.ValueString())
	rule.SetStatusCode(int32(plan.StatusCode.ValueInt64()))

	organization := r.client.Organization
	project := plan.Project.ValueString()

	client := r.client.Admin.RulesAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectRulePatch(context.Background(), organization, project, plan.Uuid.ValueString()).RuleRedirectRequest(*rule).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not update rule, unexpected error: "+err.Error(),
		)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ruleRedirectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleRedirectResourceModel
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
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectRuleDelete(r.client.Auth, organization, project, state.Uuid.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Quant project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}
