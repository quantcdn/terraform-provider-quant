package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RuleBaseConfigValidator defines the common validation processes
// for each rule provider.
func RuleBaseConfigValidator() []resource.ConfigValidator {
	return []resource.ConfigValidator{}
}

// RuleBaseAttributes defines the base rule attributes for the provider
// each rule will require this selection criteria, rather than duplicating
// it for each rule we will define this once and call this in each resource
// Schema function.
func RuleBaseAttributes(ctx context.Context) map[string]schema.Attribute {
	dominDefault, _ := types.ListValueFrom(ctx, types.StringType, []string{"any"})
	return map[string]schema.Attribute{
		"organization": schema.StringAttribute{
			Optional: true,
		},
		"project": schema.StringAttribute{
			Optional: true,
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"uuid": schema.StringAttribute{
			Computed: true,
		},
		"rule_id": schema.StringAttribute{
			Computed: true,
		},
		"url": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
		},
		"domain": schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			Default:     listdefault.StaticValue(dominDefault),
			ElementType: types.StringType,
		},
		"disabled": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
		},
		"only_with_cookie": schema.BoolAttribute{
			Optional: true,
		},
		"method": schema.StringAttribute{
			Optional: true,
			Validators: []validator.String{
				stringvalidator.OneOf("any", "method_is", "method_is_not"),
			},
		},
		"method_is": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"method_is_not": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"ip": schema.StringAttribute{
			Optional: true,
			Validators: []validator.String{
				stringvalidator.OneOf("any", "ip_is", "ip_is_not"),
			},
		},
		"ip_is": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"ip_is_not": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"country": schema.StringAttribute{
			Optional: true,
			Validators: []validator.String{
				stringvalidator.OneOf("any", "country_is", "country_is_not"),
			},
		},
		"country_is": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"country_is_not": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
	}
}
