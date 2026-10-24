package mapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// isFreeForm reports whether t is a schemaless JSON type: interface{}, a map
// with string keys and free-form values, or a slice of free-form values.
// A types.String attribute paired with such a type holds a JSON document.
func isFreeForm(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface:
		return t.NumMethod() == 0
	case reflect.Map:
		return t.Key().Kind() == reflect.String && isFreeForm(t.Elem())
	case reflect.Slice:
		return isFreeForm(t.Elem())
	default:
		return false
	}
}

// jsonToSDK decodes a JSON string into a new value of target.
func jsonToSDK(s string, target reflect.Type, path string) (reflect.Value, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	ptr := reflect.New(target)
	if err := json.Unmarshal([]byte(s), ptr.Interface()); err != nil {
		diags.AddError(
			fmt.Sprintf("ToSDK: invalid JSON for field %q", path),
			fmt.Sprintf("the value must be a JSON document: %s", err),
		)
		return reflect.Value{}, false, diags
	}
	return ptr.Elem(), true, diags
}

// jsonFromSDK encodes a free-form SDK value as a canonical JSON string. When
// prior holds JSON with the same content, prior is returned unchanged so that
// differences in key order or whitespace do not show as drift.
func jsonFromSDK(src reflect.Value, prior types.String) (attr.Value, bool) {
	if src.IsNil() {
		return types.StringNull(), true
	}

	raw, err := json.Marshal(src.Interface())
	if err != nil {
		return nil, false
	}
	canonical, err := canonicalJSON(raw)
	if err != nil {
		return nil, false
	}

	if !prior.IsNull() && !prior.IsUnknown() {
		if existing, err := canonicalJSON([]byte(prior.ValueString())); err == nil && existing == canonical {
			return prior, true
		}
	}
	return types.StringValue(canonical), true
}

// canonicalJSON re-encodes a JSON document with object keys sorted and no
// insignificant whitespace. HTML characters are not escaped.
func canonicalJSON(raw []byte) (string, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return string(bytes.TrimSuffix(buf.Bytes(), []byte("\n"))), nil
}
