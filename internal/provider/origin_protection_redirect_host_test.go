package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_application"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_environment"
)

// TestOriginProtectionRedirectHostRead verifies the origin_protection_config
// read maps the v4.19.0 redirect_host field, for both the application and
// environment attribute-type sets. This also guards the ObjectValueFrom call:
// OriginProtectionConfigValue now has three attributes, so the read struct must
// carry redirect_host or ObjectValueFrom errors.
func TestOriginProtectionRedirectHostRead(t *testing.T) {
	ctx := context.Background()

	var c quantadmingo.Container
	opc := quantadmingo.NewContainerOriginProtectionConfig()
	opc.SetEnabled(true)
	opc.SetRedirectHost("www.example.com")
	c.SetOriginProtectionConfig(*opc)

	cases := map[string]map[string]attr.Type{
		"application": resource_application.OriginProtectionConfigValue{}.AttributeTypes(ctx),
		"environment": resource_environment.OriginProtectionConfigValue{}.AttributeTypes(ctx),
	}
	for name, attrTypes := range cases {
		obj, diags := buildOriginProtectionConfigObject(ctx, &c, attrTypes)
		if diags.HasError() {
			t.Fatalf("%s: buildOriginProtectionConfigObject: %v", name, diags)
		}
		if obj.IsNull() {
			t.Fatalf("%s: expected non-null origin_protection_config", name)
		}
		rh, ok := obj.Attributes()["redirect_host"].(basetypes.StringValue)
		if !ok {
			t.Fatalf("%s: redirect_host missing or wrong type: %T", name, obj.Attributes()["redirect_host"])
		}
		if rh.ValueString() != "www.example.com" {
			t.Errorf("%s: redirect_host = %q, want www.example.com", name, rh.ValueString())
		}
	}
}

// TestOriginProtectionRedirectHostReadUnset confirms an unset redirect_host
// maps to a null string (not an error) and the object still builds.
func TestOriginProtectionRedirectHostReadUnset(t *testing.T) {
	ctx := context.Background()

	var c quantadmingo.Container
	opc := quantadmingo.NewContainerOriginProtectionConfig()
	opc.SetEnabled(false)
	c.SetOriginProtectionConfig(*opc)

	attrTypes := resource_application.OriginProtectionConfigValue{}.AttributeTypes(ctx)
	obj, diags := buildOriginProtectionConfigObject(ctx, &c, attrTypes)
	if diags.HasError() {
		t.Fatalf("buildOriginProtectionConfigObject: %v", diags)
	}
	rh, ok := obj.Attributes()["redirect_host"].(basetypes.StringValue)
	if !ok {
		t.Fatalf("redirect_host wrong type: %T", obj.Attributes()["redirect_host"])
	}
	if !rh.IsNull() {
		t.Errorf("expected null redirect_host when unset, got %q", rh.ValueString())
	}
}
