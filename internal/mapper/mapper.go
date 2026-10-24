// Package mapper provides reflection-based conversion between Terraform Plugin
// Framework model types and Go SDK request/response types, eliminating
// repetitive null-check-then-set boilerplate in resource CRUD handlers.
//
// ToSDK copies a model onto an SDK request through its Set{Name} methods, and
// FromSDK copies an SDK response onto a model through its Get{Name} methods.
// Field names come from the model's `tfsdk` tags: "allow_query_params" maps
// to SetAllowQueryParams / GetAllowQueryParams.
//
// # Supported attributes
//
//   - Scalars: types.String, types.Bool, types.Int64 (to any signed integer
//     setter), types.Float64.
//   - String lists: types.List of strings ↔ []string.
//   - Nested objects: a SingleNestedAttribute (a generated *Value type or a
//     plain types.Object) ↔ an SDK struct or pointer to struct. The object's
//     attributes are mapped recursively with the same rules, so nested objects,
//     lists and JSON strings inside it work too.
//   - Lists of nested objects: a ListNestedAttribute (types.List whose
//     elements are objects) ↔ an SDK slice of structs or struct pointers.
//   - JSON strings: see below.
//
// Other attribute types (maps, sets, numbers, integer lists) are skipped
// without a warning; the caller maps them by hand.
//
// # JSON-string attributes
//
// A free-form SDK field (interface{}, map[string]interface{},
// []interface{}, []map[string]interface{} and similar) has no fixed schema,
// so the Terraform schema models it as a types.String holding a JSON document.
// The mapper recognises such a field by the pairing of types, with no tag or
// option: a types.String attribute whose SDK setter or getter uses a
// free-form type is treated as JSON.
//
//   - ToSDK decodes the string with encoding/json into the setter's type.
//     Invalid JSON is reported as an error diagnostic.
//   - FromSDK encodes the getter's value in canonical form (object keys
//     sorted, no insignificant whitespace). If the model already holds a JSON
//     string that is semantically equal, that string is kept as is, so a
//     configuration written with different key order or formatting shows no
//     diff and does not trip Terraform's "inconsistent result after apply"
//     check.
//
// # Null, unknown and absent values
//
// ToSDK never calls a setter for a null or unknown value, at any depth.
//
// For nested objects, lists of nested objects and JSON strings, FromSDK sets
// the attribute to null when the SDK reports the field as absent: the getter
// returns a nil pointer, map, slice or interface, or a Has{Name} method
// returns false. Attributes of a nested object that have no SDK getter keep
// the value they had in the model (null if the model had none), so FromSDK
// should be given the plan or state model, as CRUD handlers already do. That
// prior value also supplies the element type of a list of nested objects,
// which a zero types.List does not carry.
//
// # Fields the caller maps itself
//
// Both functions accept callerHandled tfsdk tags. Those top-level fields are
// skipped entirely, so the caller's own mapping is not overwritten and does
// not produce a misleading "type mismatch" warning.
package mapper

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ToSDK maps fields from a Terraform model struct (tfModel) to an SDK request
// object (sdkReq) using reflection. For each exported field in tfModel that has
// a `tfsdk` struct tag, it looks for a corresponding Set{PascalCase} method on
// sdkReq. If the TF value is neither null nor unknown, the setter is called
// with the Go-native value.
//
// See the package documentation for the supported attribute types, including
// nested objects, lists of nested objects and JSON strings.
//
// sdkReq must be a pointer so that setter methods with pointer receivers are
// found by reflection.
//
// callerHandled names tfsdk tags that the caller maps itself. The mapper skips
// those fields without a warning. Use it for fields the mapper cannot convert,
// such as integer lists, or that need custom logic, so that the caller's own
// mapping is not overwritten or reported as a type mismatch on every apply.
func ToSDK(ctx context.Context, tfModel any, sdkReq any, callerHandled ...string) diag.Diagnostics {
	var diags diag.Diagnostics

	skip := make(map[string]struct{}, len(callerHandled))
	for _, tag := range callerHandled {
		skip[tag] = struct{}{}
	}

	modelVal := reflect.ValueOf(tfModel)
	modelType := modelVal.Type()

	// Dereference pointer if needed.
	if modelVal.Kind() == reflect.Pointer {
		modelVal = modelVal.Elem()
		modelType = modelVal.Type()
	}

	if modelVal.Kind() != reflect.Struct {
		diags.AddError("ToSDK: tfModel must be a struct or pointer to struct",
			fmt.Sprintf("got %s", modelType.Kind()))
		return diags
	}

	sdkVal := reflect.ValueOf(sdkReq)

	if sdkVal.Kind() != reflect.Pointer {
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
		if _, handled := skip[tag]; handled {
			continue
		}

		value, ok := modelVal.Field(i).Interface().(attr.Value)
		if !ok {
			// Not a Terraform value — skip silently.
			continue
		}

		diags.Append(setAttribute(ctx, sdkVal, tag, value, tag)...)
	}

	return diags
}

// setAttribute converts value and passes it to the Set{PascalCase(name)}
// method of sdkPtr. A missing setter, or one that does not take exactly one
// argument, skips the attribute silently. path names the attribute in
// diagnostics.
func setAttribute(ctx context.Context, sdkPtr reflect.Value, name string, value attr.Value, path string) diag.Diagnostics {
	setter := sdkPtr.MethodByName("Set" + snakeToPascal(name))
	if !setter.IsValid() || setter.Type().NumIn() != 1 {
		return nil
	}

	arg, ok, diags := valueToSDK(ctx, value, setter.Type().In(0), path)
	if ok {
		setter.Call([]reflect.Value{arg})
	}
	return diags
}

// valueToSDK converts a Terraform value to target, the setter's parameter
// type. ok is false when the setter must not be called: the value is null or
// unknown or of an unsupported type (no diagnostic), does not fit target (a
// warning), or is invalid JSON (an error).
func valueToSDK(ctx context.Context, value attr.Value, target reflect.Type, path string) (reflect.Value, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value.IsNull() || value.IsUnknown() {
		return reflect.Value{}, false, diags
	}

	switch v := value.(type) {
	case types.String:
		if isFreeForm(target) {
			return jsonToSDK(v.ValueString(), target, path)
		}
	case types.List:
		if elem, ok := sliceElem(target); ok && isStructTarget(elem) {
			return objectListToSDK(ctx, v, target, path)
		}
	case types.Bool, types.Int64, types.Float64:
	case basetypes.ObjectValuable:
		return objectToSDK(ctx, v, target, path)
	default:
		// Unsupported type (maps, sets, numbers, ...) — skip silently.
		return reflect.Value{}, false, diags
	}

	out, skip := convertTFValue(ctx, reflect.ValueOf(value), target)
	if skip {
		diags.Append(typeMismatch(path, target, value))
		return reflect.Value{}, false, diags
	}
	return out, true, diags
}

// typeMismatch reports a known value that does not fit the SDK type.
func typeMismatch(path string, target reflect.Type, value attr.Value) diag.Diagnostic {
	return diag.NewWarningDiagnostic(
		fmt.Sprintf("ToSDK: type mismatch for field %q", path),
		fmt.Sprintf("the SDK expects %s but the Terraform value is %T; field skipped", target, value),
	)
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
		// Guard the setter call: a string list must match the setter's
		// parameter type, or the reflective call panics.
		return convertToTarget(reflect.ValueOf(elems), targetType)

	default:
		// Unsupported type — skip silently. Nested objects are handled by
		// valueToSDK before this function is reached.
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

// FromSDK maps fields from an SDK response object (sdkResp) back to a Terraform
// model struct (tfModel) using reflection. For each exported field in tfModel
// that has a `tfsdk` struct tag, it looks for a corresponding Get{PascalCase}()
// method on sdkResp. If the getter exists and returns a supported type, the
// return value is converted to the appropriate Terraform Plugin Framework type
// and set on the model field.
//
// Supported getter return types: string, bool, int32/int/int64, float32/float64,
// []string, and the nested object, nested list and JSON forms described in the
// package documentation. Getters must take no arguments and return a single
// value.
//
// tfModel must be a pointer so that fields can be set via reflection.
//
// callerHandled names tfsdk tags that the caller maps itself. The mapper
// leaves those fields untouched.
func FromSDK(ctx context.Context, sdkResp any, tfModel any, callerHandled ...string) diag.Diagnostics {
	var diags diag.Diagnostics

	skip := make(map[string]struct{}, len(callerHandled))
	for _, tag := range callerHandled {
		skip[tag] = struct{}{}
	}

	modelVal := reflect.ValueOf(tfModel)
	if modelVal.Kind() != reflect.Pointer {
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

		if _, handled := skip[tag]; handled {
			continue
		}

		result, ok := callGetter(sdkVal, tag)
		if !ok {
			continue
		}

		fieldVal := modelVal.Field(i)

		// Nested objects, lists of nested objects and JSON strings are
		// converted using the field's current value for type information.
		if prior, isAttr := fieldVal.Interface().(attr.Value); isAttr && isStructuredResult(prior, result.Type()) {
			if v, ok := structuredFromSDK(ctx, sdkVal, tag, result, prior); ok {
				setField(fieldVal, reflect.ValueOf(v))
			}
			continue
		}

		tfVal, ok := convertToTFValue(ctx, result)
		if !ok {
			continue
		}

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
