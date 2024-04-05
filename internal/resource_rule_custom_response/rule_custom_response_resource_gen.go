// Code generated by terraform-plugin-framework-generator DO NOT EDIT.

package resource_rule_custom_response

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func RuleCustomResponseResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"auth_pass": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"auth_user": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"cache_lifetime": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"config": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{},
				CustomType: ConfigType{
					ObjectType: types.ObjectType{
						AttrTypes: ConfigValue{}.AttributeTypes(ctx),
					},
				},
				Computed: true,
			},
			"country": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"country_is",
						"country_is_not",
					),
				},
			},
			"country_is": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"country_is_not": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"custom_response_body": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"custom_response_status_code": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"disable_ssl_verify": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"disabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"domain": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"host": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"ip": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"ip_is",
						"ip_is_not",
					),
				},
			},
			"ip_is": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"ip_is_not": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"method": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"method_is",
						"method_is_not",
					),
				},
			},
			"method_is": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"method_is_not": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"only_proxy_404": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"only_with_cookie": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"organization": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Organization machine name",
				MarkdownDescription: "Organization machine name",
			},
			"project": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Project machine name",
				MarkdownDescription: "Project machine name",
			},
			"rule": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"strip_headers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"to": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"url": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"uuid": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

type RuleCustomResponseModel struct {
	AuthPass                 types.String `tfsdk:"auth_pass"`
	AuthUser                 types.String `tfsdk:"auth_user"`
	CacheLifetime            types.Int64  `tfsdk:"cache_lifetime"`
	Config                   ConfigValue  `tfsdk:"config"`
	Country                  types.String `tfsdk:"country"`
	CountryIs                types.List   `tfsdk:"country_is"`
	CountryIsNot             types.List   `tfsdk:"country_is_not"`
	CustomResponseBody       types.String `tfsdk:"custom_response_body"`
	CustomResponseStatusCode types.Int64  `tfsdk:"custom_response_status_code"`
	DisableSslVerify         types.Bool   `tfsdk:"disable_ssl_verify"`
	Disabled                 types.Bool   `tfsdk:"disabled"`
	Domain                   types.String `tfsdk:"domain"`
	Host                     types.String `tfsdk:"host"`
	Ip                       types.String `tfsdk:"ip"`
	IpIs                     types.List   `tfsdk:"ip_is"`
	IpIsNot                  types.List   `tfsdk:"ip_is_not"`
	Method                   types.String `tfsdk:"method"`
	MethodIs                 types.List   `tfsdk:"method_is"`
	MethodIsNot              types.List   `tfsdk:"method_is_not"`
	Name                     types.String `tfsdk:"name"`
	OnlyProxy404             types.Bool   `tfsdk:"only_proxy_404"`
	OnlyWithCookie           types.String `tfsdk:"only_with_cookie"`
	Organization             types.String `tfsdk:"organization"`
	Project                  types.String `tfsdk:"project"`
	Rule                     types.String `tfsdk:"rule"`
	StripHeaders             types.List   `tfsdk:"strip_headers"`
	To                       types.String `tfsdk:"to"`
	Url                      types.String `tfsdk:"url"`
	Uuid                     types.String `tfsdk:"uuid"`
}

var _ basetypes.ObjectTypable = ConfigType{}

type ConfigType struct {
	basetypes.ObjectType
}

func (t ConfigType) Equal(o attr.Type) bool {
	other, ok := o.(ConfigType)

	if !ok {
		return false
	}

	return t.ObjectType.Equal(other.ObjectType)
}

func (t ConfigType) String() string {
	return "ConfigType"
}

func (t ConfigType) ValueFromObject(ctx context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	if diags.HasError() {
		return nil, diags
	}

	return ConfigValue{
		state: attr.ValueStateKnown,
	}, diags
}

func NewConfigValueNull() ConfigValue {
	return ConfigValue{
		state: attr.ValueStateNull,
	}
}

func NewConfigValueUnknown() ConfigValue {
	return ConfigValue{
		state: attr.ValueStateUnknown,
	}
}

func NewConfigValue(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) (ConfigValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/521
	ctx := context.Background()

	for name, attributeType := range attributeTypes {
		attribute, ok := attributes[name]

		if !ok {
			diags.AddError(
				"Missing ConfigValue Attribute Value",
				"While creating a ConfigValue value, a missing attribute value was detected. "+
					"A ConfigValue must contain values for all attributes, even if null or unknown. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("ConfigValue Attribute Name (%s) Expected Type: %s", name, attributeType.String()),
			)

			continue
		}

		if !attributeType.Equal(attribute.Type(ctx)) {
			diags.AddError(
				"Invalid ConfigValue Attribute Type",
				"While creating a ConfigValue value, an invalid attribute value was detected. "+
					"A ConfigValue must use a matching attribute type for the value. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("ConfigValue Attribute Name (%s) Expected Type: %s\n", name, attributeType.String())+
					fmt.Sprintf("ConfigValue Attribute Name (%s) Given Type: %s", name, attribute.Type(ctx)),
			)
		}
	}

	for name := range attributes {
		_, ok := attributeTypes[name]

		if !ok {
			diags.AddError(
				"Extra ConfigValue Attribute Value",
				"While creating a ConfigValue value, an extra attribute value was detected. "+
					"A ConfigValue must not contain values beyond the expected attribute types. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("Extra ConfigValue Attribute Name: %s", name),
			)
		}
	}

	if diags.HasError() {
		return NewConfigValueUnknown(), diags
	}

	if diags.HasError() {
		return NewConfigValueUnknown(), diags
	}

	return ConfigValue{
		state: attr.ValueStateKnown,
	}, diags
}

func NewConfigValueMust(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) ConfigValue {
	object, diags := NewConfigValue(attributeTypes, attributes)

	if diags.HasError() {
		// This could potentially be added to the diag package.
		diagsStrings := make([]string, 0, len(diags))

		for _, diagnostic := range diags {
			diagsStrings = append(diagsStrings, fmt.Sprintf(
				"%s | %s | %s",
				diagnostic.Severity(),
				diagnostic.Summary(),
				diagnostic.Detail()))
		}

		panic("NewConfigValueMust received error(s): " + strings.Join(diagsStrings, "\n"))
	}

	return object
}

func (t ConfigType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if in.Type() == nil {
		return NewConfigValueNull(), nil
	}

	if !in.Type().Equal(t.TerraformType(ctx)) {
		return nil, fmt.Errorf("expected %s, got %s", t.TerraformType(ctx), in.Type())
	}

	if !in.IsKnown() {
		return NewConfigValueUnknown(), nil
	}

	if in.IsNull() {
		return NewConfigValueNull(), nil
	}

	attributes := map[string]attr.Value{}

	val := map[string]tftypes.Value{}

	err := in.As(&val)

	if err != nil {
		return nil, err
	}

	for k, v := range val {
		a, err := t.AttrTypes[k].ValueFromTerraform(ctx, v)

		if err != nil {
			return nil, err
		}

		attributes[k] = a
	}

	return NewConfigValueMust(ConfigValue{}.AttributeTypes(ctx), attributes), nil
}

func (t ConfigType) ValueType(ctx context.Context) attr.Value {
	return ConfigValue{}
}

var _ basetypes.ObjectValuable = ConfigValue{}

type ConfigValue struct {
	state attr.ValueState
}

func (v ConfigValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	attrTypes := make(map[string]tftypes.Type, 0)

	objectType := tftypes.Object{AttributeTypes: attrTypes}

	switch v.state {
	case attr.ValueStateKnown:
		vals := make(map[string]tftypes.Value, 0)

		if err := tftypes.ValidateValue(objectType, vals); err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		return tftypes.NewValue(objectType, vals), nil
	case attr.ValueStateNull:
		return tftypes.NewValue(objectType, nil), nil
	case attr.ValueStateUnknown:
		return tftypes.NewValue(objectType, tftypes.UnknownValue), nil
	default:
		panic(fmt.Sprintf("unhandled Object state in ToTerraformValue: %s", v.state))
	}
}

func (v ConfigValue) IsNull() bool {
	return v.state == attr.ValueStateNull
}

func (v ConfigValue) IsUnknown() bool {
	return v.state == attr.ValueStateUnknown
}

func (v ConfigValue) String() string {
	return "ConfigValue"
}

func (v ConfigValue) ToObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	objVal, diags := types.ObjectValue(
		map[string]attr.Type{},
		map[string]attr.Value{})

	return objVal, diags
}

func (v ConfigValue) Equal(o attr.Value) bool {
	other, ok := o.(ConfigValue)

	if !ok {
		return false
	}

	if v.state != other.state {
		return false
	}

	if v.state != attr.ValueStateKnown {
		return true
	}

	return true
}

func (v ConfigValue) Type(ctx context.Context) attr.Type {
	return ConfigType{
		basetypes.ObjectType{
			AttrTypes: v.AttributeTypes(ctx),
		},
	}
}

func (v ConfigValue) AttributeTypes(ctx context.Context) map[string]attr.Type {
	return map[string]attr.Type{}
}
