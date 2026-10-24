package mapper

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---------------------------------------------------------------------------
// ToSDK: nested objects and lists of nested objects
// ---------------------------------------------------------------------------

// sliceElem returns the element type of a slice type.
func sliceElem(t reflect.Type) (reflect.Type, bool) {
	if t.Kind() != reflect.Slice {
		return nil, false
	}
	return t.Elem(), true
}

// structOf returns the struct type behind t, which is either a struct or a
// pointer to a struct.
func structOf(t reflect.Type) (reflect.Type, bool) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	return t, true
}

// isStructTarget reports whether t is a struct or a pointer to a struct.
func isStructTarget(t reflect.Type) bool {
	_, ok := structOf(t)
	return ok
}

// objectToSDK builds an SDK struct of type target (a struct or pointer to
// struct) from a nested object, mapping each attribute through the struct's
// setters.
func objectToSDK(ctx context.Context, obj basetypes.ObjectValuable, target reflect.Type, path string) (reflect.Value, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	st, ok := structOf(target)
	if !ok {
		diags.Append(typeMismatch(path, target, obj))
		return reflect.Value{}, false, diags
	}

	objVal, d := obj.ToObjectValue(ctx)
	diags.Append(d...)
	if diags.HasError() {
		return reflect.Value{}, false, diags
	}

	ptr := reflect.New(st)
	attrs := objVal.Attributes()
	for _, name := range sortedKeys(attrs) {
		diags.Append(setAttribute(ctx, ptr, name, attrs[name], path+"."+name)...)
	}

	if target.Kind() == reflect.Pointer {
		return ptr, true, diags
	}
	return ptr.Elem(), true, diags
}

// objectListToSDK builds an SDK slice of structs (or struct pointers) from a
// list of nested objects. Null and unknown elements are left out.
func objectListToSDK(ctx context.Context, list types.List, target reflect.Type, path string) (reflect.Value, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	elems := list.Elements()
	out := reflect.MakeSlice(target, 0, len(elems))
	for i, elem := range elems {
		elemPath := fmt.Sprintf("%s[%d]", path, i)

		if elem.IsNull() || elem.IsUnknown() {
			continue
		}

		obj, ok := elem.(basetypes.ObjectValuable)
		if !ok {
			diags.Append(typeMismatch(elemPath, target.Elem(), elem))
			return reflect.Value{}, false, diags
		}

		v, ok, d := objectToSDK(ctx, obj, target.Elem(), elemPath)
		diags.Append(d...)
		if !ok {
			return reflect.Value{}, false, diags
		}
		out = reflect.Append(out, v)
	}

	return out, true, diags
}

// sortedKeys returns the keys of attrs in sorted order, so setters are called
// in a stable order.
func sortedKeys(attrs map[string]attr.Value) []string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// FromSDK: nested objects, lists of nested objects and JSON strings
// ---------------------------------------------------------------------------

// callGetter calls Get{PascalCase(name)} on sdkPtr. ok is false when there is
// no such method or it is not a zero-argument, single-result getter.
func callGetter(sdkPtr reflect.Value, name string) (reflect.Value, bool) {
	getter := sdkPtr.MethodByName("Get" + snakeToPascal(name))
	if !getter.IsValid() || getter.Type().NumIn() != 0 || getter.Type().NumOut() != 1 {
		return reflect.Value{}, false
	}
	return getter.Call(nil)[0], true
}

// reportedAbsent reports whether sdkPtr has a Has{PascalCase(name)} method
// that returns false.
func reportedAbsent(sdkPtr reflect.Value, name string) bool {
	has := sdkPtr.MethodByName("Has" + snakeToPascal(name))
	if !has.IsValid() || has.Type().NumIn() != 0 || has.Type().NumOut() != 1 ||
		has.Type().Out(0).Kind() != reflect.Bool {
		return false
	}
	return !has.Call(nil)[0].Bool()
}

// isStructuredResult reports whether a top-level model field and a getter's
// result type pair up as a nested object, a list of nested objects or a JSON
// string. Other pairs keep the scalar conversion in convertToTFValue.
func isStructuredResult(prior attr.Value, result reflect.Type) bool {
	switch prior.(type) {
	case types.String:
		return isFreeForm(result)
	case types.List:
		elem, ok := sliceElem(result)
		return ok && isStructTarget(elem)
	case types.Bool, types.Int64, types.Float64:
		return false
	case basetypes.ObjectValuable:
		return true
	default:
		return false
	}
}

// structuredFromSDK converts a top-level getter result for a nested object,
// list of nested objects or JSON string field, honouring Has{Name}.
func structuredFromSDK(ctx context.Context, sdkPtr reflect.Value, name string, result reflect.Value, prior attr.Value) (attr.Value, bool) {
	if reportedAbsent(sdkPtr, name) {
		return nullOf(ctx, prior.Type(ctx))
	}
	return valueFromSDK(ctx, result, prior)
}

// valueFromSDK converts an SDK value to a Terraform value of the same type as
// prior, the attribute's current value. prior supplies the attribute type
// (including the element type of lists and the attribute types of objects)
// and, for JSON strings, the formatting to keep when the content is
// unchanged. ok is false when the value cannot be converted, in which case
// the caller leaves the attribute as it was.
func valueFromSDK(ctx context.Context, src reflect.Value, prior attr.Value) (attr.Value, bool) {
	if p, ok := prior.(types.String); ok && isFreeForm(src.Type()) {
		return jsonFromSDK(src, p)
	}

	for src.Kind() == reflect.Pointer || src.Kind() == reflect.Interface {
		if src.IsNil() {
			return nullOf(ctx, prior.Type(ctx))
		}
		src = src.Elem()
	}

	switch p := prior.(type) {
	case types.String:
		if src.Kind() == reflect.String {
			return types.StringValue(src.String()), true
		}
	case types.Bool:
		if src.Kind() == reflect.Bool {
			return types.BoolValue(src.Bool()), true
		}
	case types.Int64:
		switch src.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return types.Int64Value(src.Int()), true
		}
	case types.Float64:
		switch src.Kind() {
		case reflect.Float32, reflect.Float64:
			return types.Float64Value(src.Float()), true
		}
	case types.List:
		return listFromSDK(ctx, src, p)
	case basetypes.ObjectValuable:
		if src.Kind() == reflect.Struct {
			return objectFromSDK(ctx, src, p)
		}
	}

	return nil, false
}

// objectFromSDK builds a nested object of prior's type from an SDK struct.
// Attributes without a getter keep their prior value, or null when there is
// none. Attributes the SDK reports as absent through Has{Name} become null.
func objectFromSDK(ctx context.Context, src reflect.Value, prior basetypes.ObjectValuable) (attr.Value, bool) {
	priorObj, d := prior.ToObjectValue(ctx)
	if d.HasError() {
		return nil, false
	}
	attrTypes := priorObj.AttributeTypes(ctx)
	if len(attrTypes) == 0 {
		// A zero types.Object carries no attribute types to build from.
		return nil, false
	}
	priorAttrs := priorObj.Attributes()

	// Getters on SDK structs use pointer receivers.
	ptr := reflect.New(src.Type())
	ptr.Elem().Set(src)

	attrs := make(map[string]attr.Value, len(attrTypes))
	for name, t := range attrTypes {
		current, ok := priorAttrs[name]
		if !ok || current.IsUnknown() {
			if current, ok = nullOf(ctx, t); !ok {
				return nil, false
			}
		}
		attrs[name] = current

		result, ok := callGetter(ptr, name)
		if !ok {
			continue
		}
		if reportedAbsent(ptr, name) {
			if v, ok := nullOf(ctx, t); ok {
				attrs[name] = v
			}
			continue
		}
		if v, ok := valueFromSDK(ctx, result, current); ok {
			attrs[name] = v
		}
	}

	obj, d := types.ObjectValue(attrTypes, attrs)
	if d.HasError() {
		return nil, false
	}

	typable, ok := prior.Type(ctx).(basetypes.ObjectTypable)
	if !ok {
		return nil, false
	}
	v, d := typable.ValueFromObject(ctx, obj)
	if d.HasError() {
		return nil, false
	}
	return v, true
}

// listFromSDK builds a list of prior's element type from an SDK slice. A nil
// slice becomes a null list. Elements are matched to prior elements by index
// so JSON strings inside them keep their formatting when unchanged.
func listFromSDK(ctx context.Context, src reflect.Value, prior types.List) (attr.Value, bool) {
	if src.Kind() != reflect.Slice {
		return nil, false
	}

	elemType := prior.ElementType(ctx)
	if elemType == nil {
		// A zero types.List carries no element type to build from.
		return nil, false
	}

	if src.IsNil() {
		return types.ListNull(elemType), true
	}

	priorElems := prior.Elements()
	elems := make([]attr.Value, src.Len())
	for i := range elems {
		var current attr.Value
		if i < len(priorElems) && !priorElems[i].IsUnknown() {
			current = priorElems[i]
		} else {
			var ok bool
			if current, ok = nullOf(ctx, elemType); !ok {
				return nil, false
			}
		}

		v, ok := valueFromSDK(ctx, src.Index(i), current)
		if !ok {
			return nil, false
		}
		elems[i] = v
	}

	list, d := types.ListValue(elemType, elems)
	if d.HasError() {
		return nil, false
	}
	return list, true
}

// nullOf returns the null value of t, including custom types such as the
// generated nested object types.
func nullOf(ctx context.Context, t attr.Type) (attr.Value, bool) {
	v, err := t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), nil))
	if err != nil {
		return nil, false
	}
	return v, true
}

// setField assigns v to field when the types match.
func setField(field, v reflect.Value) {
	if field.CanSet() && v.IsValid() && v.Type().AssignableTo(field.Type()) {
		field.Set(v)
	}
}
