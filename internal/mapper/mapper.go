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
// Supported TF types: types.String, types.Bool, types.Int64, types.Float64.
// types.List and nested objects are silently skipped (handled per-resource).
//
// sdkReq must be a pointer so that setter methods with pointer receivers are
// found by reflection.
func ToSDK(_ context.Context, tfModel any, sdkReq any) diag.Diagnostics {
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

		args, skip := convertTFValue(fieldVal, setter.Type().In(0))
		if skip {
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
func convertTFValue(fieldVal reflect.Value, targetType reflect.Type) (reflect.Value, bool) {
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

	default:
		// Unsupported type (types.List, nested objects, etc.) — skip silently.
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
