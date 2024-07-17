package provider

import (
	"context"
	"regexp"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource                     = (*ruleRedirectResource)(nil)
	_ resource.ResourceWithConfigure        = (*ruleRedirectResource)(nil)
	_ resource.ResourceWithImportState      = (*ruleRedirectResource)(nil)
	_ resource.ResourceWithConfigValidators = (*ruleRedirectResource)(nil)
)

func NewRuleRedirectResource() resource.Resource {
	return &ruleRedirectResource{}
}

type ruleRedirectResource struct {
	client *client.Client
}

type ruleRedirectResourceModel struct {
	Project        types.String `tfsdk:"project"`
	Organization   types.String `tfsdk:"organization"`
	Name           types.String `tfsdk:"name"`
	Uuid           types.String `tfsdk:"uuid"`
	Url            types.List   `tfsdk:"url"`
	Domain         types.List   `tfsdk:"domain"`
	Disabled       types.Bool   `tfsdk:"disabled"`
	OnlyWithCookie types.Bool   `tfsdk:"only_with_cookie"`
	Method         types.String `tfsdk:"method"`
	MethodIs       types.List   `tfsdk:"method_is"`
	MethodIsNot    types.List   `tfsdk:"method_is_not"`
	Ip             types.String `tfsdk:"ip"`
	IpIs           types.List   `tfsdk:"ip_is"`
	IpIsNot        types.List   `tfsdk:"ip_is_not"`
	Country        types.String `tfsdk:"country"`
	CountryIs      types.List   `tfsdk:"country_is"`
	CountryIsNot   types.List   `tfsdk:"country_is_not"`

	// Rule specific details.
	RedirectTo   types.String `tfsdk:"redirect_to"`
	RedirectCode types.String `tfsdk:"redirect_code"`
}

func (r *ruleRedirectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_redirect"
}

func (r *ruleRedirectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := RuleBaseAttributes(ctx)
	attributes["redirect_to"] = schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.RegexMatches(regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`), "Must be a valid URL"),
		},
	}
	attributes["redirect_code"] = schema.StringAttribute{
		Computed: true,
		Default:  stringdefault.StaticString("301"),
		Validators: []validator.String{
			stringvalidator.OneOf("301", "302", "303"),
		},
	}

	// Set the attributes for this rule.
	resp.Schema = schema.Schema{Attributes: attributes}
}

func (r *ruleRedirectResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return RuleBaseConfigValidator()
}

func (r *ruleRedirectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			"Expected *internal.Client, got: %T. Please report this issue to the provider developers",
		)
	}
	r.client = client
}

func (r *ruleRedirectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ruleRedirectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	resp.Diagnostics.Append(callRuleRedirectCreateAPI(ctx, r, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleRedirectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ruleRedirectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callRuleRedirectReadAPI(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleRedirectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ruleRedirectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	resp.Diagnostics.Append(callRuleRedirectUpdateAPI(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleRedirectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ruleRedirectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callRuleRedirectDeleteAPI(ctx, r, &data)...)
}

func (r *ruleRedirectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ruleRedirectResourceModel
	var err error
	data.Project, data.Uuid, err = utils.GetRuleImportId(req.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			err.Error(),
		)
		return
	}
	// Read API call logic
	resp.Diagnostics.Append(callRuleRedirectReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// callRuleRedirectCreateAPI calls the API endpoint to create a rule
// resource in Quant.
func callRuleRedirectCreateAPI(ctx context.Context, r *ruleRedirectResource, rule *ruleRedirectResourceModel) (diags diag.Diagnostics) {
	req := *openapi.NewRuleRedirectRequestWithDefaults()
	req.SetName(rule.Name.ValueString())

	var domains []string
	rule.Domain.ElementsAs(ctx, domains, false)
	req.SetDomain(domains)

	var urls []string
	rule.Url.ElementsAs(ctx, urls, false)
	req.SetUrl(urls)

	req.SetCountry(rule.Country.ValueString())
	var countryList []string

	if rule.Country.ValueString() == "country_is" {
		rule.CountryIs.ElementsAs(ctx, countryList, false)
		req.SetCountryIs(countryList)
	} else if rule.Country.ValueString() == "country_is_not" {
		rule.CountryIsNot.ElementsAs(ctx, countryList, false)
		req.SetCountryIsNot(countryList)
	}

	req.SetIp(rule.Ip.ValueString())
	var iplist []string

	if rule.Ip.ValueString() == "ip_is" {
		rule.IpIs.ElementsAs(ctx, iplist, false)
		req.SetIpIs(iplist)
	} else if rule.Ip.ValueString() == "ip_is_not" {
		rule.IpIsNot.ElementsAs(ctx, iplist, false)
		req.SetIpIsNot(iplist)
	}

	req.SetMethod(rule.Method.ValueString())
	var methodList []string

	if rule.Method.ValueString() == "method_is" {
		rule.MethodIs.ElementsAs(ctx, methodList, false)
		req.SetMethodIs(methodList)
	} else if rule.Method.ValueString() == "method_is_not" {
		rule.MethodIsNot.ElementsAs(ctx, methodList, false)
		req.SetMethodIsNot(methodList)
	}

	req.SetRedirectCode(rule.RedirectCode.ValueString())
	req.SetRedirectTo(rule.RedirectTo.ValueString())

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	api, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectCreate(r.client.AuthContext, org, rule.Project.ValueString()).RuleRedirectRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	rule.Uuid = types.StringValue(api.Uuid)
	return
}

// callRuleRedirectReadAPI
func callRuleRedirectReadAPI(ctx context.Context, r *ruleRedirectResource, rule *ruleRedirectResourceModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	api, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectRead(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()
	if err != nil {
		diags.AddError("Failed to read rule", err.Error())
		return
	}

	rule.Name = types.StringValue(*api.Name)
	rule.Uuid = types.StringValue(api.Uuid)
	domains, d := types.ListValueFrom(ctx, types.StringType, api.Domain)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.Domain = domains
	urls, d := types.ListValueFrom(ctx, types.StringType, api.Url)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.Url = urls
	rule.Ip = types.StringValue(*api.Ip)
	ips, d := types.ListValueFrom(ctx, types.StringType, api.IpIs)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.IpIs = types.List(ips)
	ipIsNot, d := types.ListValueFrom(ctx, types.StringType, api.IpIsNot)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.IpIsNot = types.List(ipIsNot)
	rule.Country = types.StringValue(*api.Country)
	countries, d := types.ListValueFrom(ctx, types.StringType, api.CountryIs)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.CountryIs = types.List(countries)
	rule.CountryIsNot, d = types.ListValueFrom(ctx, types.StringType, api.CountryIsNot)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.CountryIsNot = types.List(rule.CountryIsNot)

	rule.Method = types.StringValue(*api.Method)
	methods, d := types.ListValueFrom(ctx, types.StringType, api.MethodIs)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.MethodIs = types.List(methods)
	methodIsNot, d := types.ListValueFrom(ctx, types.StringType, api.MethodIsNot)
	if d.HasError() {
		diags.Append(d...)
		return
	}
	rule.MethodIsNot = types.List(methodIsNot)

	// Rule specific fields.
	rule.RedirectCode = types.StringValue(api.ActionConfig.StatusCode)
	rule.RedirectTo = types.StringValue(api.ActionConfig.To)

	return
}

// callRuleRedirectUpdateAPI
func callRuleRedirectUpdateAPI(ctx context.Context, r *ruleRedirectResource, rule *ruleRedirectResourceModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	req := *openapi.NewRuleRedirectRequestWithDefaults()
	req.SetName(rule.Name.ValueString())

	var domains []string
	rule.Domain.ElementsAs(ctx, domains, false)
	req.SetDomain(domains)

	var urls []string
	rule.Url.ElementsAs(ctx, urls, false)
	req.SetUrl(urls)

	req.SetCountry(rule.Country.ValueString())
	var countryList []string

	if rule.Country.ValueString() == "country_is" {
		rule.CountryIs.ElementsAs(ctx, countryList, false)
		req.SetCountryIs(countryList)
	} else if rule.Country.ValueString() == "country_is_not" {
		rule.CountryIsNot.ElementsAs(ctx, countryList, false)
		req.SetCountryIsNot(countryList)
	}

	req.SetIp(rule.Ip.ValueString())
	var iplist []string

	if rule.Ip.ValueString() == "ip_is" {
		rule.IpIs.ElementsAs(ctx, iplist, false)
		req.SetIpIs(iplist)
	} else if rule.Ip.ValueString() == "ip_is_not" {
		rule.IpIsNot.ElementsAs(ctx, iplist, false)
		req.SetIpIsNot(iplist)
	}

	req.SetMethod(rule.Method.ValueString())
	var methodList []string

	if rule.Method.ValueString() == "method_is" {
		rule.MethodIs.ElementsAs(ctx, methodList, false)
		req.SetMethodIs(methodList)
	} else if rule.Method.ValueString() == "method_is_not" {
		rule.MethodIsNot.ElementsAs(ctx, methodList, false)
		req.SetMethodIsNot(methodList)
	}

	req.SetRedirectCode(rule.RedirectCode.ValueString())
	req.SetRedirectTo(rule.RedirectTo.ValueString())

	org := r.client.Organization
	if !rule.Organization.IsNull() {
		org = rule.Organization.ValueString()
	}

	_, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectUpdate(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).RuleRedirectRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule", err.Error())
		return
	}

	return
}

// callRuleRedirectDeleteAPI calls the delete API endpoint with for a given resource.
func callRuleRedirectDeleteAPI(ctx context.Context, r *ruleRedirectResource, rule *ruleRedirectResourceModel) (diags diag.Diagnostics) {
	if rule.Uuid.IsNull() || rule.Uuid.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
	}

	org := r.client.Organization

	_, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.Uuid.ValueString()).Execute()
	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
