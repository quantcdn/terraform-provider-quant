package provider

import (
	"context"
	"fmt"
	"net/http"
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
	_ resource.Resource              = &ruleCustomResponse{}
	_ resource.ResourceWithConfigure = &ruleCustomResponse{}
)

// NewruleCustomResponse is a helper function to simplify the provider implementation.
func NewruleCustomResponseResource() resource.Resource {
	return &ruleCustomResponse{}
}

// ruleCustomResponse is the resource implementation.
type ruleCustomResponse struct {
	client *client.Client
}

type ruleCustomResponseModel struct {
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

	// Rule details
	CustomResponseStatusCode types.Int64  `tfsdk:"custom_response_status_code"`
	CustomResponseBody       types.String `tfsdk:"custom_response_body"`
}

// Configure adds the provider configured client to the resource.
func (r *ruleCustomResponse) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ruleCustomResponse) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_custom_response"
}

// Schema defines the schema for the resource.
func (r *ruleCustomResponse) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "A name for the rule",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "The project machine name",
				Required:            true,
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "If this rule is disabled",
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Computed:            true,
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
func (r *ruleCustomResponse) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleCustomResponseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Url.IsNull() {
		plan.Url = types.StringValue("*")
	}

	rule := openapi.NewRuleCustomResponseRequest()

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
	rule.SetUrl(plan.Url.ValueString())
	rule.SetCustomResponseBody(plan.CustomResponseBody.ValueString())
	rule.SetCustomResponseStatusCode(int32(plan.CustomResponseStatusCode.ValueInt64()))

	organization := r.client.Organization
	project := plan.Project.ValueString()

	client := r.client.Admin.RulesAPI
	res, i, err := client.OrganizationsOrganizationProjectsProjectRulesCustomResponsePost(r.client.Auth, organization, project).RuleCustomResponseRequest(*rule).Execute()
	if i.StatusCode == http.StatusForbidden {
		resp.Diagnostics.AddError(
			"Error create rule",
			"You are not authorised to make this request, please check credentials.",
		)
		return
	}

	if i.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create the rule, unexpected error "+helpers.ErrorFromAPIBody(i.Body),
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule",
			"Could not create rule, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Uuid = types.StringValue(*res.GetData().Rules[0].Uuid)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *ruleCustomResponse) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleCustomResponseModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	organization := r.client.Organization
	project := state.Project.ValueString()

	if state.Uuid.IsNull() {
		resp.Diagnostics.AddWarning(
			"Rule uuid is null",
			"The rule uuid is null and data is not able to be updated from the API",
		)
	} else {
		client := r.client.Admin.RulesAPI
		res, i, err := client.OrganizationsOrganizationProjectsProjectRulesCustomResponseRuleGet(context.Background(), organization, project, state.Uuid.ValueString()).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading header rule",
				"Could not read rule, unexpected error: "+err.Error(),
			)
		}

		if i.StatusCode != http.StatusOK {
			resp.Diagnostics.AddError(
				"Error reading header rule",
				"Could not load rule definition for "+state.Uuid.ValueString()+" "+helpers.ErrorFromAPIBody(i.Body),
			)
			return
		}
		if len(res.Data.Rules) == 0 {
			resp.Diagnostics.AddError(
				"Unkonwn UUID",
				"Could not load rule definition for "+state.Uuid.ValueString(),
			)
			return
		}

		rule := res.Data.Rules[0]
		state.Uuid = types.StringValue(*rule.Uuid)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ruleCustomResponse) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleCustomResponseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Uuid.IsNull() {
		resp.Diagnostics.AddWarning(
			"Rule uuid is null",
			"The rule uuid is null and data is not able to be updated from the API",
		)
	} else {
		rule := openapi.NewRuleCustomResponseRequest()

		rule.SetName(plan.Name.ValueString())
		rule.SetDisabled(plan.Disabled.ValueBool())
		rule.SetDomain(plan.Domain.ValueString())

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
		rule.SetUrl(plan.Url.ValueString())
		rule.SetCustomResponseBody(plan.CustomResponseBody.ValueString())
		rule.SetCustomResponseStatusCode(int32(plan.CustomResponseStatusCode.ValueInt64()))

		organization := r.client.Organization
		project := plan.Project.ValueString()

		client := r.client.Admin.RulesAPI
		res, i, err := client.OrganizationsOrganizationProjectsProjectRulesCustomResponseRulePatch(r.client.Auth, organization, project, plan.Uuid.ValueString()).RuleCustomResponseRequest(*rule).Execute()

		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating rule",
				"Could not update rule, unexpected error: "+err.Error(),
			)
		}

		if i.StatusCode != http.StatusOK {
			resp.Diagnostics.AddError(
				"Error reading header rule",
				"Could not load rule definition for "+plan.Uuid.ValueString()+" "+helpers.ErrorFromAPIBody(i.Body),
			)
			return
		}

		if len(res.Data.Rules) == 0 {
			resp.Diagnostics.AddError(
				"Unkonwn UUID",
				"Could not load rule definition for"+plan.Uuid.ValueString(),
			)
			return
		}

		r := res.Data.Rules[0]
		plan.Uuid = types.StringValue(*r.Uuid)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ruleCustomResponse) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleCustomResponseModel
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
	_, _, err := client.OrganizationsOrganizationProjectsProjectRulesCustomResponseRuleDelete(r.client.Auth, organization, project, state.Uuid.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting rule ",
			"Could not delete rule, unexpected error: "+err.Error(),
		)
		return
	}
}
