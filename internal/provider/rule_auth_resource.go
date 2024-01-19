package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &ruleAuth{}
	_ resource.ResourceWithConfigure = &ruleAuth{}
)

// NewruleAuth is a helper function to simplify the provider implementation.
func NewRuleAuthResource() resource.Resource {
	return &ruleAuth{}
}

// ruleAuth is the resource implementation.
type ruleAuth struct {
	client *client.Client
}

type ruleAuthModel struct {
	Name     types.String `tfsdk:"name"`
	Uuid     types.String `tfsdk:"uuid"`
	Project  types.String `tfsdk:"project"`
	Disabled types.Bool   `tfsdk:"disabled"`

	Domain   types.String `tfsdk:"domain"`
	Url      types.String `tfsdk:"url"`
	AuthUser types.String `tfsdk:"auth_user"`
	AuthPass types.String `tfsdk:"auth_pass"`

	// Rule selection.
	CountryInclude types.Bool     `tfsdk:"country_include"`
	Countries      []types.String `tfsdk:"countries"`
	MethodInclude  types.Bool     `tfsdk:"method_include"`
	Methods        []types.String `tfsdk:"methods"`
	IpInclude      types.Bool     `tfsdk:"ip_include"`
	Ips            []types.String `tfsdk:"ips"`
	OnlyWithCookie types.String   `tfsdk:"only_with_cookie"`
}

// Configure adds the provider configured client to the resource.
func (r *ruleAuth) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ruleAuth) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_authentication"
}

// Schema defines the schema for the resource.
func (r *ruleAuth) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The rules UUID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain the rule applies to",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("any"),
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL to apply to",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/*"),
			},
			"countries": schema.ListAttribute{
				MarkdownDescription: "A list of countries",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"country_include": schema.BoolAttribute{
				MarkdownDescription: "Include the country list",
				Optional:            true,
			},
			"methods": schema.ListAttribute{
				MarkdownDescription: "A list of HTTP methods",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"method_include": schema.BoolAttribute{
				MarkdownDescription: "Include the methods",
				Optional:            true,
			},
			"ips": schema.ListAttribute{
				MarkdownDescription: "A list of IP addresses",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"ip_include": schema.BoolAttribute{
				MarkdownDescription: "Include hte IP addresses",
				Optional:            true,
			},
			"only_with_cookie": schema.StringAttribute{
				MarkdownDescription: "Apply rule only if the cookie is present",
				Optional:            true,
			},
			"auth_user": schema.StringAttribute{
				MarkdownDescription: "HTTP authentication username",
				Required:            true,
			},
			"auth_pass": schema.StringAttribute{
				MarkdownDescription: "HTTP authentication password",
				Required:            true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ruleAuth) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleAuthModel
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

	rule := openapi.NewRuleAuthRequest()

	if plan.Url.IsNull() {
		plan.Url = types.StringValue("*")
	}

	if plan.CountryInclude.IsNull() {
		rule.SetCountry("any")
	}

	if plan.MethodInclude.IsNull() {
		rule.SetMethod("any")
	}

	if plan.IpInclude.IsNull() {
		rule.SetIp("any")
	}

	rule.SetName(plan.Name.ValueString())
	rule.SetDisabled(plan.Disabled.ValueBool())
	rule.SetDomain(plan.Domain.ValueString())
	rule.SetUrl(plan.Url.ValueString())

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

	// Rule behaviour.
	rule.SetAuthUser(plan.AuthUser.ValueString())
	rule.SetAuthPass(plan.AuthUser.ValueString())

	organization := r.client.Organization
	project := plan.Project.ValueString()

	client := r.client.Admin.RulesAPI
	res, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectPost(context.Background(), organization, project).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
	}

	plan.Uuid = types.StringValue(*res.GetData().Rules[0].Uuid)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ruleAuth) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleAuthModel
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
func (r *ruleAuth) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleAuthModel
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

	rule := openapi.NewRuleAuthRequest()

	if plan.Url.IsNull() {
		plan.Url = types.StringValue("*")
	}

	if plan.CountryInclude.IsNull() {
		rule.SetCountry("any")
	}

	if plan.MethodInclude.IsNull() {
		rule.SetMethod("any")
	}

	if plan.IpInclude.IsNull() {
		rule.SetIp("any")
	}

	rule.SetName(plan.Name.ValueString())
	rule.SetDisabled(plan.Disabled.ValueBool())
	rule.SetDomain(plan.Domain.ValueString())
	rule.SetUrl(plan.Url.ValueString())

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

	// Rule behaviour.
	rule.SetAuthUser(plan.AuthUser.ValueString())
	rule.SetAuthPass(plan.AuthUser.ValueString())

	organization := r.client.Organization
	project := plan.Project.ValueString()

	client := r.client.Admin.RulesAPI
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesRedirectRulePatch(context.Background(), organization, project, plan.Uuid.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule",
			"Could not update rule, unexpected error: "+err.Error(),
		)
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ruleAuth) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleAuthModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Project.IsNull() {
		resp.Diagnostics.AddError(
			"Error deleting rule",
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
			"Error deleting rule",
			"Could not delete rule, unexpected error: "+err.Error(),
		)
		return
	}
}
