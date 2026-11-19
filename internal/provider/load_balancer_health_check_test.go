package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_application"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_environment"
)

// TestLoadBalancerHealthCheckRead maps the v4.35.0 container field into state for
// both resources, and keeps a container on the platform default as a null object.
func TestLoadBalancerHealthCheckRead(t *testing.T) {
	ctx := context.Background()

	var set quantadmingo.Container
	lb := quantadmingo.NewContainerLoadBalancerHealthCheck()
	lb.SetPath("/_quant/health")
	lb.SetIntervalSeconds(60)
	set.SetLoadBalancerHealthCheck(*lb)

	var unset quantadmingo.Container

	// The API returns null for a container on the default; that must read back as a null object.
	var explicitNull quantadmingo.Container
	if err := json.Unmarshal([]byte(`{"name":"a","imageReference":{"type":"internal","identifier":"x"},"loadBalancerHealthCheck":null}`), &explicitNull); err != nil {
		t.Fatal(err)
	}

	cases := map[string]map[string]attr.Type{
		"application": resource_application.LoadBalancerHealthCheckValue{}.AttributeTypes(ctx),
		"environment": resource_environment.LoadBalancerHealthCheckValue{}.AttributeTypes(ctx),
	}
	for name, attrTypes := range cases {
		obj, diags := buildLoadBalancerHealthCheckObject(ctx, &set, attrTypes)
		if diags.HasError() {
			t.Fatalf("%s: %v", name, diags)
		}
		if p := obj.Attributes()["path"].(basetypes.StringValue); p.ValueString() != "/_quant/health" {
			t.Errorf("%s: path = %q", name, p.ValueString())
		}
		if i := obj.Attributes()["interval_seconds"].(basetypes.Int64Value); i.ValueInt64() != 60 {
			t.Errorf("%s: interval_seconds = %d", name, i.ValueInt64())
		}

		for label, c := range map[string]*quantadmingo.Container{"unset": &unset, "json null": &explicitNull} {
			null, diags := buildLoadBalancerHealthCheckObject(ctx, c, attrTypes)
			if diags.HasError() || !null.IsNull() {
				t.Errorf("%s/%s: want a null object (null=%v, diags=%v)", name, label, null.IsNull(), diags)
			}
		}
	}
}

// TestLoadBalancerHealthCheckWrite sends only the values that are set, so an omitted
// interval takes the platform default.
func TestLoadBalancerHealthCheckWrite(t *testing.T) {
	both := buildSDKLoadBalancerHealthCheck(map[string]attr.Value{
		"path":             types.StringValue("/_quant/health"),
		"interval_seconds": types.Int64Value(60),
	})
	if got, _ := json.Marshal(both); string(got) != `{"intervalSeconds":60,"path":"/_quant/health"}` {
		t.Errorf("both set: %s", got)
	}

	pathOnly := buildSDKLoadBalancerHealthCheck(map[string]attr.Value{
		"path":             types.StringValue("/healthz"),
		"interval_seconds": types.Int64Null(),
	})
	if got, _ := json.Marshal(pathOnly); string(got) != `{"path":"/healthz"}` {
		t.Errorf("path only: %s", got)
	}
}
