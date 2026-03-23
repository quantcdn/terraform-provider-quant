package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// parseAPIError extracts a human-readable error message from an API error
// response body. Returns the message string if the body contains a JSON object
// with an "error" and "message" field, otherwise returns empty string.
func parseAPIError(httpResp *http.Response) string {
	if httpResp == nil || httpResp.Body == nil {
		return ""
	}
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ""
	}
	var apiError struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	if jsonErr := json.Unmarshal(bodyBytes, &apiError); jsonErr == nil && apiError.Message != "" {
		return apiError.Message
	}
	return ""
}

// extractStringList converts a types.List with string elements to a []string
// for use in SDK calls. Returns nil and diagnostics if the conversion fails.
func extractStringList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var result []string
	diags := list.ElementsAs(ctx, &result, false)
	return result, diags
}

// setConditionalList handles one conditional list triplet (e.g., country +
// country_is + country_is_not) in the Read direction (API response -> TF
// model). If the API value pointer is nil or empty, the model selector is set
// to null and both lists are set to empty. Otherwise the selector is set and
// the appropriate list is populated from the API response.
func setConditionalList(
	ctx context.Context,
	apiValue *string, apiIs []string, apiIsNot []string,
	modelValue *types.String, modelIs *types.List, modelIsNot *types.List,
) diag.Diagnostics {
	var diags diag.Diagnostics
	emptyList, d := types.ListValueFrom(ctx, types.StringType, []string{})
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	if apiValue == nil || *apiValue == "" {
		*modelValue = types.StringNull()
		*modelIs = emptyList
		*modelIsNot = emptyList
		return diags
	}

	*modelValue = types.StringValue(*apiValue)

	if len(apiIs) > 0 {
		isList, d := types.ListValueFrom(ctx, types.StringType, apiIs)
		diags.Append(d...)
		*modelIs = isList
	} else {
		*modelIs = emptyList
	}

	if len(apiIsNot) > 0 {
		isNotList, d := types.ListValueFrom(ctx, types.StringType, apiIsNot)
		diags.Append(d...)
		*modelIsNot = isNotList
	} else {
		*modelIsNot = emptyList
	}

	return diags
}

// buildConditionalList handles one conditional list triplet in the Write
// direction (TF model -> SDK request). If the model selector is null/unknown,
// nothing is set. Otherwise it calls the selector setter and then extracts and
// sets the appropriate list (is or is_not) based on the selector value.
func buildConditionalList(
	ctx context.Context,
	modelValue types.String, modelIs types.List, modelIsNot types.List,
	setValue func(string), setIs func([]string), setIsNot func([]string),
) diag.Diagnostics {
	var diags diag.Diagnostics

	if modelValue.IsNull() || modelValue.IsUnknown() {
		return diags
	}

	setValue(modelValue.ValueString())

	isList, d := extractStringList(ctx, modelIs)
	diags.Append(d...)
	if len(isList) > 0 {
		setIs(isList)
	}

	isNotList, d := extractStringList(ctx, modelIsNot)
	diags.Append(d...)
	if len(isNotList) > 0 {
		setIsNot(isNotList)
	}

	return diags
}

// setConditionalListsFromAPI is a convenience wrapper that applies
// setConditionalList for all three standard conditional triplets (country, ip,
// method) in one call.
func setConditionalListsFromAPI(
	ctx context.Context,
	// Country API fields
	apiCountry *string, apiCountryIs []string, apiCountryIsNot []string,
	// IP API fields
	apiIp *string, apiIpIs []string, apiIpIsNot []string,
	// Method API fields
	apiMethod *string, apiMethodIs []string, apiMethodIsNot []string,
	// Country model fields
	modelCountry *types.String, modelCountryIs *types.List, modelCountryIsNot *types.List,
	// IP model fields
	modelIp *types.String, modelIpIs *types.List, modelIpIsNot *types.List,
	// Method model fields
	modelMethod *types.String, modelMethodIs *types.List, modelMethodIsNot *types.List,
) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(setConditionalList(ctx,
		apiCountry, apiCountryIs, apiCountryIsNot,
		modelCountry, modelCountryIs, modelCountryIsNot,
	)...)

	diags.Append(setConditionalList(ctx,
		apiIp, apiIpIs, apiIpIsNot,
		modelIp, modelIpIs, modelIpIsNot,
	)...)

	diags.Append(setConditionalList(ctx,
		apiMethod, apiMethodIs, apiMethodIsNot,
		modelMethod, modelMethodIs, modelMethodIsNot,
	)...)

	return diags
}

// addUseStateForUnknown adds UseStateForUnknown plan modifiers to all Computed
// attributes in the schema. Without these, every plan cycle treats unconfigured
// Optional+Computed fields as Unknown, causing perpetual drift.
func addUseStateForUnknown(attrs map[string]schema.Attribute) {
	for name, attr := range attrs {
		switch a := attr.(type) {
		case schema.StringAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.BoolAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.Int64Attribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.ListAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, listplanmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.MapAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, mapplanmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.Float64Attribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, float64planmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.SingleNestedAttribute:
			if a.Computed {
				addUseStateForUnknown(a.Attributes)
				attrs[name] = a
			}
		case schema.ListNestedAttribute:
			if a.Computed {
				addUseStateForUnknown(a.NestedObject.Attributes)
				a.PlanModifiers = append(a.PlanModifiers, listplanmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		case schema.ObjectAttribute:
			if a.Computed {
				a.PlanModifiers = append(a.PlanModifiers, objectplanmodifier.UseStateForUnknown())
				attrs[name] = a
			}
		}
	}
}

// clearStringPlanModifiers strips all plan modifiers from the named string
// attributes. Use this after addUseStateForUnknown to exempt volatile computed
// fields (e.g. updated_at, crawler_schedule) that change on every update.
func clearStringPlanModifiers(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		if attr, ok := attrs[name]; ok {
			if sa, ok := attr.(schema.StringAttribute); ok {
				sa.PlanModifiers = nil
				attrs[name] = sa
			}
		}
	}
}

// buildConditionalListsForRequest is a convenience wrapper that applies
// buildConditionalList for all three standard conditional triplets (country,
// ip, method) in one call.
func buildConditionalListsForRequest(
	ctx context.Context,
	// Country model fields
	modelCountry types.String, modelCountryIs types.List, modelCountryIsNot types.List,
	// IP model fields
	modelIp types.String, modelIpIs types.List, modelIpIsNot types.List,
	// Method model fields
	modelMethod types.String, modelMethodIs types.List, modelMethodIsNot types.List,
	// Country setters
	setCountry func(string), setCountryIs func([]string), setCountryIsNot func([]string),
	// IP setters
	setIp func(string), setIpIs func([]string), setIpIsNot func([]string),
	// Method setters
	setMethod func(string), setMethodIs func([]string), setMethodIsNot func([]string),
) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(buildConditionalList(ctx,
		modelCountry, modelCountryIs, modelCountryIsNot,
		setCountry, setCountryIs, setCountryIsNot,
	)...)

	diags.Append(buildConditionalList(ctx,
		modelIp, modelIpIs, modelIpIsNot,
		setIp, setIpIs, setIpIsNot,
	)...)

	diags.Append(buildConditionalList(ctx,
		modelMethod, modelMethodIs, modelMethodIsNot,
		setMethod, setMethodIs, setMethodIsNot,
	)...)

	return diags
}
