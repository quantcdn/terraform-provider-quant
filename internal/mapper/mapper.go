// Package mapper provides reflection-based conversion between Terraform Plugin
// Framework model types and Go SDK request types, eliminating repetitive
// null-check-then-set boilerplate in resource CRUD handlers.
package mapper

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ToSDK maps fields from a Terraform model struct (tfModel) to an SDK request
// object (sdkReq) using reflection. For each exported field in tfModel that has
// a `tfsdk` struct tag, it looks for a corresponding Set{PascalCase} method on
// sdkReq. If the TF value is neither null nor unknown, the setter is called
// with the Go-native value.
//
// Supported TF types: types.String, types.Bool, types.Int64, types.Float64,
// types.List (string elements only). Nested objects are silently skipped
// (handled per-resource).
//
// sdkReq must be a pointer so that setter methods with pointer receivers are
// found by reflection.
func ToSDK(ctx context.Context, tfModel any, sdkReq any) diag.Diagnostics {
	var diags diag.Diagnostics

	modelVal := reflect.ValueOf(tfModel)
	modelType := modelVal.Type()

	// Dereference pointer if needed.
	if modelVal.Kind() == reflect.Ptr {
		modelVal = modelVal.Elem()
		modelType = modelVal.Type()
	}

	if modelVal.Kind() != reflect.Struct {
		diags.AddError("ToSDK: tfModel must be a struct or pointer to struct",
			fmt.Sprintf("got %s", modelType.Kind()))
		return diags
	}

	sdkVal := reflect.ValueOf(sdkReq)

	if sdkVal.Kind() != reflect.Ptr {
		diags.AddError("ToSDK: sdkReq must be a pointer",
			fmt.Sprintf("got %s", sdkVal.Kind()))
		return diags
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		fieldVal := modelVal.Field(i)
		setterName := "Set" + snakeToPascal(tag)

		setter := sdkVal.MethodByName(setterName)
		if !setter.IsValid() {
			// No matching setter — skip silently.
			continue
		}

		// Ensure the setter takes exactly one argument.
		if setter.Type().NumIn() != 1 {
			continue
		}

		args, skip := convertTFValue(ctx, fieldVal, setter.Type().In(0))
		if skip {
			if setter.IsValid() && !isNullOrUnknown(fieldVal) {
				diags.AddWarning(
					fmt.Sprintf("ToSDK: type mismatch for field %q", tag),
					fmt.Sprintf("setter %s expects %s but TF value is %s — field skipped",
						setterName, setter.Type().In(0), fieldVal.Type()),
				)
			}
			continue
		}

		setter.Call([]reflect.Value{args})
	}

	return diags
}

// convertTFValue extracts the Go-native value from a Terraform Plugin Framework
// attribute value and converts it to match the setter's expected parameter type.
// Returns the converted reflect.Value and a bool indicating whether the field
// should be skipped (true = skip).
func convertTFValue(ctx context.Context, fieldVal reflect.Value, targetType reflect.Type) (reflect.Value, bool) {
	iface := fieldVal.Interface()

	switch v := iface.(type) {
	case types.String:
		if v.IsNull() || v.IsUnknown() {
			return reflect.Value{}, true
		}
		return convertToTarget(reflect.ValueOf(v.ValueString()), targetType)

	case types.Bool:
		if v.IsNull() || v.IsUnknown() {
			return reflect.Value{}, true
		}
		return convertToTarget(reflect.ValueOf(v.ValueBool()), targetType)

	case types.Int64:
		if v.IsNull() || v.IsUnknown() {
			return reflect.Value{}, true
		}
		return convertInt64ToTarget(v.ValueInt64(), targetType)

	case types.Float64:
		if v.IsNull() || v.IsUnknown() {
			return reflect.Value{}, true
		}
		return convertToTarget(reflect.ValueOf(v.ValueFloat64()), targetType)

	case types.List:
		if v.IsNull() || v.IsUnknown() {
			return reflect.Value{}, true
		}
		var elems []string
		diags := v.ElementsAs(ctx, &elems, false)
		if diags.HasError() {
			return reflect.Value{}, true
		}
		return reflect.ValueOf(elems), false

	default:
		// Unsupported type (nested objects, etc.) — skip silently.
		return reflect.Value{}, true
	}
}

// convertInt64ToTarget converts an int64 value to the setter's expected integer
// type. SDK setters often expect int32 even though TF uses int64 internally.
func convertInt64ToTarget(val int64, targetType reflect.Type) (reflect.Value, bool) {
	switch targetType.Kind() {
	case reflect.Int32:
		return reflect.ValueOf(int32(val)), false
	case reflect.Int64:
		return reflect.ValueOf(val), false
	case reflect.Int:
		return reflect.ValueOf(int(val)), false
	case reflect.Int16:
		return reflect.ValueOf(int16(val)), false
	case reflect.Int8:
		return reflect.ValueOf(int8(val)), false
	default:
		return reflect.Value{}, true
	}
}

// convertToTarget returns the value as-is if it already matches the target type,
// or skips if there's a type mismatch.
func convertToTarget(val reflect.Value, targetType reflect.Type) (reflect.Value, bool) {
	if val.Type() == targetType {
		return val, false
	}
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType), false
	}
	return reflect.Value{}, true
}

// isNullOrUnknown reports whether a Terraform attribute value is null or unknown.
// This is used to distinguish benign skips (null/unknown) from type mismatch skips.
func isNullOrUnknown(fieldVal reflect.Value) bool {
	iface := fieldVal.Interface()
	switch v := iface.(type) {
	case types.String:
		return v.IsNull() || v.IsUnknown()
	case types.Bool:
		return v.IsNull() || v.IsUnknown()
	case types.Int64:
		return v.IsNull() || v.IsUnknown()
	case types.Float64:
		return v.IsNull() || v.IsUnknown()
	case types.List:
		return v.IsNull() || v.IsUnknown()
	default:
		// Unsupported types are always treated as "null-like" — no warning.
		return true
	}
}

// FromSDK maps fields from an SDK response object (sdkResp) back to a Terraform
// model struct (tfModel) using reflection. For each exported field in tfModel
// that has a `tfsdk` struct tag, it looks for a corresponding Get{PascalCase}()
// method on sdkResp. If the getter exists and returns a supported type, the
// return value is converted to the appropriate Terraform Plugin Framework type
// and set on the model field.
//
// Supported getter return types: string, bool, int32/int/int64, float32/float64.
// Getters must take no arguments and return a single value.
//
// tfModel must be a pointer so that fields can be set via reflection.
func FromSDK(ctx context.Context, sdkResp any, tfModel any) diag.Diagnostics {
	var diags diag.Diagnostics

	modelVal := reflect.ValueOf(tfModel)
	if modelVal.Kind() != reflect.Ptr {
		diags.AddError("FromSDK: tfModel must be a pointer",
			fmt.Sprintf("got %s", modelVal.Kind()))
		return diags
	}

	modelVal = modelVal.Elem()
	modelType := modelVal.Type()

	if modelVal.Kind() != reflect.Struct {
		diags.AddError("FromSDK: tfModel must be a pointer to struct",
			fmt.Sprintf("got pointer to %s", modelType.Kind()))
		return diags
	}

	sdkVal := reflect.ValueOf(sdkResp)

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}

		getterName := "Get" + snakeToPascal(tag)
		getter := sdkVal.MethodByName(getterName)
		if !getter.IsValid() {
			// No matching getter — skip silently.
			continue
		}

		// Ensure the getter takes no arguments and returns exactly one value.
		if getter.Type().NumIn() != 0 || getter.Type().NumOut() != 1 {
			continue
		}

		result := getter.Call(nil)[0]
		tfVal, ok := convertToTFValue(ctx, result)
		if !ok {
			continue
		}

		fieldVal := modelVal.Field(i)
		if fieldVal.CanSet() {
			fieldVal.Set(tfVal)
		}
	}

	return diags
}

// listValueFromFunc is the function used to convert []string to types.List.
// It defaults to types.ListValueFrom and can be overridden in tests to simulate
// errors.
var listValueFromFunc = types.ListValueFrom

// convertToTFValue converts a Go-native value returned by an SDK getter into
// the corresponding Terraform Plugin Framework attribute value. Returns the
// converted reflect.Value and a bool indicating success (false = skip).
func convertToTFValue(ctx context.Context, val reflect.Value) (reflect.Value, bool) {
	switch val.Kind() {
	case reflect.String:
		return reflect.ValueOf(types.StringValue(val.String())), true

	case reflect.Bool:
		return reflect.ValueOf(types.BoolValue(val.Bool())), true

	case reflect.Int32, reflect.Int, reflect.Int64:
		return reflect.ValueOf(types.Int64Value(val.Int())), true

	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(types.Float64Value(val.Float())), true

	case reflect.Slice:
		// Handle []string → types.List conversion.
		if val.Type().Elem().Kind() == reflect.String {
			strs := make([]string, val.Len())
			for i := 0; i < val.Len(); i++ {
				strs[i] = val.Index(i).String()
			}
			listVal, diags := listValueFromFunc(ctx, types.StringType, strs)
			if diags.HasError() {
				return reflect.Value{}, false
			}
			return reflect.ValueOf(listVal), true
		}
		return reflect.Value{}, false

	default:
		// Unsupported return type — skip silently.
		return reflect.Value{}, false
	}
}

// snakeToPascal converts a snake_case string to PascalCase.
// Example: "allow_query_params" → "AllowQueryParams"
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}
