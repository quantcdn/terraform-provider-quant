package provider

import (
	"context"
	"fmt"
	"regexp"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	RuleId         types.String `tfsdk:"rule_id"`
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
		Required: true,
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

	if resp.Diagnostics.HasError() {
		return
	}

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
	var plan ruleRedirectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state ruleRedirectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	plan.RuleId = state.RuleId

	// Update API call logic
	resp.Diagnostics.Append(callRuleRedirectUpdateAPI(ctx, r, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callRuleRedirectReadAPI(ctx, r, &plan)...)

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	data.Project, data.RuleId, err = utils.GetRuleImportId(req.ID)

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
	for _, domain := range rule.Domain.Elements() {
		if strVal, ok := domain.(types.String); ok {
			domains = append(domains, strVal.ValueString())
		}
	}
	req.SetDomain(domains)

	var urls []string
	for _, url := range rule.Url.Elements() {
		if strVal, ok := url.(types.String); ok {
			urls = append(urls, strVal.ValueString())
		}
	}
	req.SetUrl(urls)

	if !rule.Country.IsNull() {
		req.SetCountry(rule.Country.ValueString())
		var countryList []string
		if rule.Country.ValueString() == "country_is" {
			for _, country := range rule.CountryIs.Elements() {
				if strVal, ok := country.(types.String); ok {
					countryList = append(countryList, strVal.ValueString())
				}
			}
			req.SetCountryIs(countryList)
		} else if rule.Country.ValueString() == "country_is_not" {
			for _, country := range rule.CountryIsNot.Elements() {
				if strVal, ok := country.(types.String); ok {
					countryList = append(countryList, strVal.ValueString())
				}
			}
			req.SetCountryIsNot(countryList)
		}
	}

	if (!rule.Ip.IsNull()) {
		req.SetIp(rule.Ip.ValueString())
		var iplist []string
		if rule.Ip.ValueString() == "ip_is" {
			for _, ip := range rule.IpIs.Elements() {
				if strVal, ok := ip.(types.String); ok {
					iplist = append(iplist, strVal.ValueString())
				}
			}
			req.SetIpIs(iplist)
		} else if rule.Ip.ValueString() == "ip_is_not" {
			for _, ip := range rule.IpIsNot.Elements() {
				if strVal, ok := ip.(types.String); ok {
					iplist = append(iplist, strVal.ValueString())
				}
			}
			req.SetIpIsNot(iplist)
		}
	}


	if (!rule.Method.IsNull()) {
		req.SetMethod(rule.Method.ValueString())
		var methodList []string
		if rule.Method.ValueString() == "method_is" {
			for _, method := range rule.MethodIs.Elements() {
				if strVal, ok := method.(types.String); ok {
					methodList = append(methodList, strVal.ValueString())
				}
			}
			req.SetMethodIs(methodList)
		} else if rule.Method.ValueString() == "method_is_not" {
			for _, method := range rule.MethodIsNot.Elements() {
				if strVal, ok := method.(types.String); ok {
					methodList = append(methodList, strVal.ValueString())
				}
			}
			req.SetMethodIsNot(methodList)
		}
	}

	req.SetRedirectCode(rule.RedirectCode.ValueString())
	req.SetRedirectTo(rule.RedirectTo.ValueString())

	res, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectCreate(r.client.AuthContext, r.client.Organization, rule.Project.ValueString()).RuleRedirectRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to create rule", err.Error())
		return
	}

	rule.Uuid = types.StringValue(res.GetUuid())
	rule.RuleId = types.StringValue(res.GetRuleId())

	return
}


// callRuleRedirectReadAPI
func callRuleRedirectReadAPI(ctx context.Context, r *ruleRedirectResource, rule *ruleRedirectResourceModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	api, res, err := r.client.Instance.RulesRedirectAPI.RulesRedirectRead(r.client.AuthContext, r.client.Organization, rule.Project.ValueString(), rule.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to read rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
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
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to update unkown rule, please update terraform state.",
		)
		return
	}

	req := *openapi.NewRuleRedirectRequestUpdateWithDefaults()
	req.SetName(rule.Name.ValueString())

	var domains []string
	for _, domain := range rule.Domain.Elements() {
		if strVal, ok := domain.(types.String); ok {
			domains = append(domains, strVal.ValueString())
		}
	}
	req.SetDomain(domains)

	var urls []string
	for _, url := range rule.Url.Elements() {
		if strVal, ok := url.(types.String); ok {
			urls = append(urls, strVal.ValueString())
		}
	}
	req.SetUrl(urls)

	req.SetCountry(rule.Country.ValueString())
	var countryList []string
	for _, country := range rule.CountryIs.Elements() {
		if strVal, ok := country.(types.String); ok {
			countryList = append(countryList, strVal.ValueString())
		}
	}
	req.SetCountryIs(countryList)
	countryList = []string{}
	for _, country := range rule.CountryIsNot.Elements() {
		if strVal, ok := country.(types.String); ok {
			countryList = append(countryList, strVal.ValueString())
		}
	}
	req.SetCountryIsNot(countryList)

	req.SetIp(rule.Ip.ValueString())
	var iplist []string
	for _, ip := range rule.IpIs.Elements() {
		if strVal, ok := ip.(types.String); ok {
			iplist = append(iplist, strVal.ValueString())
		}
	}
	req.SetIpIs(iplist)
	iplist = []string{}
	for _, ip := range rule.IpIsNot.Elements() {
		if strVal, ok := ip.(types.String); ok {
			iplist = append(iplist, strVal.ValueString())
		}
	}
	req.SetIpIsNot(iplist)

	req.SetMethod(rule.Method.ValueString())
	var methodList []string
	for _, method := range rule.MethodIs.Elements() {
		if strVal, ok := method.(types.String); ok {
			methodList = append(methodList, strVal.ValueString())
		}
	}
	req.SetMethodIs(methodList)
	methodList = []string{}
	for _, method := range rule.MethodIsNot.Elements() {
		if strVal, ok := method.(types.String); ok {
			methodList = append(methodList, strVal.ValueString())
		}
	}
	req.SetMethodIsNot(methodList)


	req.SetRedirectCode(rule.RedirectCode.ValueString())
	req.SetRedirectTo(rule.RedirectTo.ValueString())

	_, res, err := r.client.Instance.RulesRedirectAPI.RulesRedirectUpdate(r.client.AuthContext, r.client.Organization, rule.Project.ValueString(), rule.RuleId.ValueString()).RuleRedirectRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError("Failed to update rule", err.Error())
		diags.AddError("Response", fmt.Sprintf("%v", res))
		return
	}

	return
}

// callRuleRedirectDeleteAPI calls the delete API endpoint with for a given resource.
func callRuleRedirectDeleteAPI(ctx context.Context, r *ruleRedirectResource, rule *ruleRedirectResourceModel) (diags diag.Diagnostics) {
	if rule.RuleId.IsNull() || rule.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("uuid"),
			"Missing rule.uuid attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
	}

	org := r.client.Organization
	_, _, err := r.client.Instance.RulesRedirectAPI.RulesRedirectDelete(r.client.AuthContext, org, rule.Project.ValueString(), rule.RuleId.ValueString()).Execute()

	if err != nil {
		diags.AddError("Failed to delete rule", err.Error())
		return
	}

	return
}
