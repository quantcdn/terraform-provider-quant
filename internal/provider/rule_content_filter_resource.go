package provider

import (
	"context"
	"fmt"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_rule_content_filter"
	"terraform-provider-quant/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource                = (*ruleContentFilterResource)(nil)
	_ resource.ResourceWithConfigure   = (*ruleContentFilterResource)(nil)
	_ resource.ResourceWithImportState = (*ruleContentFilterResource)(nil)
)

func NewRuleContentFilterResource() resource.Resource {
	return &ruleContentFilterResource{}
}

type ruleContentFilterResource struct {
	client *client.Client
}

func (r *ruleContentFilterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_content_filter"
}

func (r *ruleContentFilterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_rule_content_filter.RuleContentFilterResourceSchema(ctx)
}

func (r *ruleContentFilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ruleContentFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // Serialise rule modifications to avoid backend JSON races
    if r.client != nil && r.client.RulesMutex != nil {
        r.client.RulesMutex.Lock()
        defer r.client.RulesMutex.Unlock()
    }
	var data resource_rule_content_filter.RuleContentFilterModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	diags := callRuleContentFilterCreateAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = callRuleContentFilterReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleContentFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_rule_content_filter.RuleContentFilterModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	diags := callRuleContentFilterReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ruleContentFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // Serialise rule modifications to avoid backend JSON races
    if r.client != nil && r.client.RulesMutex != nil {
        r.client.RulesMutex.Lock()
        defer r.client.RulesMutex.Unlock()
    }
	var plan resource_rule_content_filter.RuleContentFilterModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state resource_rule_content_filter.RuleContentFilterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Uuid = state.Uuid
	plan.RuleId = state.RuleId

	// Update API call logic
	diags := callRuleContentFilterUpdateAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read updated state
	diags = callRuleContentFilterReadAPI(ctx, r, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ruleContentFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // Serialise rule modifications to avoid backend JSON races
    if r.client != nil && r.client.RulesMutex != nil {
        r.client.RulesMutex.Lock()
        defer r.client.RulesMutex.Unlock()
    }
	var data resource_rule_content_filter.RuleContentFilterModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	diags := callRuleContentFilterDeleteAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *ruleContentFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_rule_content_filter.RuleContentFilterModel
	var err error
	data.Project, data.RuleId, err = utils.GetRuleImportId(req.ID)

	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse import ID. Error: %s", err.Error()),
		)
		return
	}

	// Read API call logic
	diags := callRuleContentFilterReadAPI(ctx, r, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func callRuleContentFilterCreateAPI(ctx context.Context, r *ruleContentFilterResource, data *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	req := *quantadmingo.NewRuleContentFilterRequestWithDefaults()
	req.SetName(data.Name.ValueString())

	// Domain handling
	if !data.Domain.IsNull() {
		var domains []string
		if !data.Domain.IsUnknown() {
			diags.Append(data.Domain.ElementsAs(ctx, &domains, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetDomain(domains)
	}

	// URL handling
	if !data.Url.IsNull() {
		var urls []string
		if !data.Url.IsUnknown() {
			diags.Append(data.Url.ElementsAs(ctx, &urls, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetUrl(urls)
	}

	// Country handling
	if !data.Country.IsNull() {
		req.SetCountry(data.Country.ValueString())
		if data.Country.ValueString() == "country_is" && !data.CountryIs.IsNull() {
			var countryList []string
			if !data.CountryIs.IsUnknown() {
				diags.Append(data.CountryIs.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIs(countryList)
		} else if data.Country.ValueString() == "country_is_not" && !data.CountryIsNot.IsNull() {
			var countryList []string
			if !data.CountryIsNot.IsUnknown() {
				diags.Append(data.CountryIsNot.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIsNot(countryList)
		}
	}

	// IP handling
	if !data.Ip.IsNull() {
		req.SetIp(data.Ip.ValueString())
		if data.Ip.ValueString() == "ip_is" && !data.IpIs.IsNull() {
			var ipList []string
			if !data.IpIs.IsUnknown() {
				diags.Append(data.IpIs.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIs(ipList)
		} else if data.Ip.ValueString() == "ip_is_not" && !data.IpIsNot.IsNull() {
			var ipList []string
			if !data.IpIsNot.IsUnknown() {
				diags.Append(data.IpIsNot.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIsNot(ipList)
		}
	}

	// Method handling
	if !data.Method.IsNull() {
		req.SetMethod(data.Method.ValueString())
		if data.Method.ValueString() == "method_is" && !data.MethodIs.IsNull() {
			var methodList []string
			if !data.MethodIs.IsUnknown() {
				diags.Append(data.MethodIs.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIs(methodList)
		} else if data.Method.ValueString() == "method_is_not" && !data.MethodIsNot.IsNull() {
			var methodList []string
			if !data.MethodIsNot.IsUnknown() {
				diags.Append(data.MethodIsNot.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIsNot(methodList)
		}
	}

	// Content filter specific configuration
	req.SetFnUuid(data.FnUuid.ValueString())
	req.SetDisabled(data.Disabled.ValueBool())

	// Weight handling
	if !data.Weight.IsNull() && !data.Weight.IsUnknown() {
		weight := int32(data.Weight.ValueInt64())
		req.SetWeight(weight)
	}

	// Make the API call
	api, _, err := r.client.Instance.RulesContentFilterAPI.RulesContentFilterCreate(r.client.AuthContext, r.client.Organization, data.Project.ValueString()).RuleContentFilterRequest(req).Execute()
	if err != nil {
		diags.AddError(
			"Error creating rule content filter",
			fmt.Sprintf("Could not create rule content filter, unexpected error: %s", err.Error()),
		)
		return
	}

	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())

	return
}

func callRuleContentFilterUpdateAPI(ctx context.Context, r *ruleContentFilterResource, data *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to update unknown rule, please update terraform state.",
		)
		return
	}

	req := *quantadmingo.NewRuleContentFilterRequestUpdateWithDefaults()
	req.SetName(data.Name.ValueString())

	// Domain handling
	if !data.Domain.IsNull() {
		var domains []string
		if !data.Domain.IsUnknown() {
			diags.Append(data.Domain.ElementsAs(ctx, &domains, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetDomain(domains)
	}

	// URL handling
	if !data.Url.IsNull() {
		var urls []string
		if !data.Url.IsUnknown() {
			diags.Append(data.Url.ElementsAs(ctx, &urls, false)...)
			if diags.HasError() {
				return
			}
		}
		req.SetUrl(urls)
	}

	// Country handling
	if !data.Country.IsNull() {
		req.SetCountry(data.Country.ValueString())
		if data.Country.ValueString() == "country_is" {
			var countryList []string
			if !data.CountryIs.IsUnknown() {
				diags.Append(data.CountryIs.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIs(countryList)
		} else if data.Country.ValueString() == "country_is_not" {
			var countryList []string
			if !data.CountryIsNot.IsUnknown() {
				diags.Append(data.CountryIsNot.ElementsAs(ctx, &countryList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetCountryIsNot(countryList)
		}
	}

	// IP handling
	if !data.Ip.IsNull() {
		req.SetIp(data.Ip.ValueString())
		if data.Ip.ValueString() == "ip_is" {
			var ipList []string
			if !data.IpIs.IsUnknown() {
				diags.Append(data.IpIs.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIs(ipList)
		} else if data.Ip.ValueString() == "ip_is_not" {
			var ipList []string
			if !data.IpIsNot.IsUnknown() {
				diags.Append(data.IpIsNot.ElementsAs(ctx, &ipList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetIpIsNot(ipList)
		}
	}

	// Method handling
	if !data.Method.IsNull() {
		req.SetMethod(data.Method.ValueString())
		if data.Method.ValueString() == "method_is" {
			var methodList []string
			if !data.MethodIs.IsUnknown() {
				diags.Append(data.MethodIs.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIs(methodList)
		} else if data.Method.ValueString() == "method_is_not" {
			var methodList []string
			if !data.MethodIsNot.IsUnknown() {
				diags.Append(data.MethodIsNot.ElementsAs(ctx, &methodList, false)...)
				if diags.HasError() {
					return
				}
			}
			req.SetMethodIsNot(methodList)
		}
	}

	// Content filter specific configuration
	req.SetFnUuid(data.FnUuid.ValueString())
	req.SetDisabled(data.Disabled.ValueBool())

	// Weight handling
	if !data.Weight.IsNull() && !data.Weight.IsUnknown() {
		weight := int32(data.Weight.ValueInt64())
		req.SetWeight(weight)
	}

	// Make the API call
	_, _, err := r.client.Instance.RulesContentFilterAPI.RulesContentFilterUpdate(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.RuleId.ValueString(),
	).RuleContentFilterRequestUpdate(req).Execute()

	if err != nil {
		diags.AddError(
			"Error updating rule content filter",
			fmt.Sprintf("Could not update rule content filter, unexpected error: %s", err.Error()),
		)
		return
	}

	return
}

func callRuleContentFilterReadAPI(ctx context.Context, r *ruleContentFilterResource, data *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to read unknown rule, please update terraform state.",
		)
		return
	}

	api, resp, err := r.client.Instance.RulesContentFilterAPI.RulesContentFilterRead(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.RuleId.ValueString(),
	).Execute()

	if err != nil {
		// Check if it's a 404 error, which might indicate the rule was deleted
		if resp != nil && resp.StatusCode == 404 {
			diags.AddError(
				"Rule content filter not found",
				fmt.Sprintf("The rule content filter with ID %s no longer exists. It may have been deleted outside of Terraform.",
					data.RuleId.ValueString()),
			)
			return
		}

		diags.AddError(
			"Error reading rule content filter",
			fmt.Sprintf("Could not read rule content filter, unexpected error: %s", err.Error()),
		)
		return
	}

	// Get the rules action config
	actionConfig := api.GetActionConfig()

	// Set basic fields
	data.Uuid = types.StringValue(api.GetUuid())
	data.RuleId = types.StringValue(api.GetRuleId())
	data.Name = types.StringValue(api.GetName())
	data.Organization = types.StringValue(r.client.Organization)
	data.Action = types.StringValue(api.GetAction())
	data.OnlyWithCookie = types.StringNull()
	if api.Weight != nil {
		data.Weight = types.Int64Value(int64(*api.Weight))
	} else {
		data.Weight = types.Int64Value(0)
	}

	// Convert API lists to types.List - handle nil values
	domains := api.GetDomain()
	if domains == nil {
		data.Domain = types.ListNull(types.StringType)
	} else {
		domainList, diag := types.ListValueFrom(ctx, types.StringType, domains)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.Domain = domainList
	}

	urls := api.GetUrl()
	if urls == nil {
		data.Url = types.ListNull(types.StringType)
	} else {
		urlList, diag := types.ListValueFrom(ctx, types.StringType, urls)
		if diag.HasError() {
			diags.Append(diag...)
			return
		}
		data.Url = urlList
	}

	// Handle content filter specific configuration
	data.FnUuid = types.StringValue(actionConfig.GetFnUuid())
	data.Disabled = types.BoolValue(api.GetDisabled())

	// Handle country, IP, and method selection criteria
	data.Country = types.StringValue(api.GetCountry())
	if api.GetCountry() == "country_is" {
		countries := api.GetCountryIs()
		if countries == nil {
			data.CountryIs = types.ListNull(types.StringType)
		} else {
			countriesList, diag := types.ListValueFrom(ctx, types.StringType, countries)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.CountryIs = countriesList
		}
		data.CountryIsNot = types.ListNull(types.StringType)
	} else if api.GetCountry() == "country_is_not" {
		countriesNot := api.GetCountryIsNot()
		if countriesNot == nil {
			data.CountryIsNot = types.ListNull(types.StringType)
		} else {
			countriesNotList, diag := types.ListValueFrom(ctx, types.StringType, countriesNot)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.CountryIsNot = countriesNotList
		}
		data.CountryIs = types.ListNull(types.StringType)
	} else {
		data.CountryIs = types.ListNull(types.StringType)
		data.CountryIsNot = types.ListNull(types.StringType)
	}

	data.Ip = types.StringValue(api.GetIp())
	if api.GetIp() == "ip_is" {
		ips := api.GetIpIs()
		if ips == nil {
			data.IpIs = types.ListNull(types.StringType)
		} else {
			isList, diag := types.ListValueFrom(ctx, types.StringType, ips)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.IpIs = isList
		}
		data.IpIsNot = types.ListNull(types.StringType)
	} else if api.GetIp() == "ip_is_not" {
		ipsNot := api.GetIpIsNot()
		if ipsNot == nil {
			data.IpIsNot = types.ListNull(types.StringType)
		} else {
			isNotList, diag := types.ListValueFrom(ctx, types.StringType, ipsNot)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.IpIsNot = isNotList
		}
		data.IpIs = types.ListNull(types.StringType)
	} else {
		data.IpIs = types.ListNull(types.StringType)
		data.IpIsNot = types.ListNull(types.StringType)
	}

	data.Method = types.StringValue(api.GetMethod())
	if api.GetMethod() == "method_is" {
		methods := api.GetMethodIs()
		if methods == nil {
			data.MethodIs = types.ListNull(types.StringType)
		} else {
			methodIsList, diag := types.ListValueFrom(ctx, types.StringType, methods)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.MethodIs = methodIsList
		}
		data.MethodIsNot = types.ListNull(types.StringType)
	} else if api.GetMethod() == "method_is_not" {
		methodsNot := api.GetMethodIsNot()
		if methodsNot == nil {
			data.MethodIsNot = types.ListNull(types.StringType)
		} else {
			methodIsNotList, diag := types.ListValueFrom(ctx, types.StringType, methodsNot)
			if diag.HasError() {
				diags.Append(diag...)
				return
			}
			data.MethodIsNot = methodIsNotList
		}
		data.MethodIs = types.ListNull(types.StringType)
	} else {
		data.MethodIs = types.ListNull(types.StringType)
		data.MethodIsNot = types.ListNull(types.StringType)
	}

	data.Rule = types.StringNull() // Obsolete field

	return
}

func callRuleContentFilterDeleteAPI(ctx context.Context, r *ruleContentFilterResource, data *resource_rule_content_filter.RuleContentFilterModel) (diags diag.Diagnostics) {
	if data.RuleId.IsNull() || data.RuleId.IsUnknown() {
		diags.AddAttributeError(
			path.Root("rule_id"),
			"Missing rule.rule_id attribute",
			"Unable to delete unknown rule, please update terraform state.",
		)
		return
	}

	_, _, err := r.client.Instance.RulesContentFilterAPI.RulesContentFilterDelete(
		r.client.AuthContext,
		r.client.Organization,
		data.Project.ValueString(),
		data.RuleId.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError(
			"Error deleting rule content filter",
			fmt.Sprintf("Could not delete rule content filter, unexpected error: %s", err.Error()),
		)
		return
	}

	return
}
