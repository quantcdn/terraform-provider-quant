package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKnownOrNull(t *testing.T) {
	if got := knownOrNull(types.StringUnknown()); !got.IsNull() {
		t.Fatalf("an unknown value must become null, got %v", got)
	}
	if got := knownOrNull(types.StringValue("Bearer abc")); got.ValueString() != "Bearer abc" {
		t.Fatalf("a known value must be kept, got %v", got)
	}
	if got := knownOrNull(types.StringNull()); !got.IsNull() {
		t.Fatalf("null must stay null, got %v", got)
	}
}

func TestUnmaskHeadersKeepsTheConfiguredValueForAMaskedHeader(t *testing.T) {
	saved := types.MapValueMust(types.StringType, map[string]attr.Value{
		"Authorization": types.StringValue("Bearer real"),
	})

	got := unmaskHeaders(map[string]string{"Authorization": maskedSecret, "X-Plain": "v"}, saved)

	if got["Authorization"].(types.String).ValueString() != "Bearer real" {
		t.Fatalf("the configured value must be kept, got %v", got["Authorization"])
	}
	if got["X-Plain"].(types.String).ValueString() != "v" {
		t.Fatalf("a plain value must come from the API, got %v", got["X-Plain"])
	}
}

func TestUnmaskHeadersNeverStoresTheMask(t *testing.T) {
	for name, saved := range map[string]types.Map{
		"null":    types.MapNull(types.StringType),
		"unknown": types.MapUnknown(types.StringType),
		"other":   types.MapValueMust(types.StringType, map[string]attr.Value{"X-Other": types.StringValue("v")}),
	} {
		got := unmaskHeaders(map[string]string{"Authorization": maskedSecret}, saved)
		if _, ok := got["Authorization"]; ok {
			t.Fatalf("%s: a masked header with no configured value must be left out, got %v", name, got)
		}
	}
}
