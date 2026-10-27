package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_application"
)

func TestSetApplicationMetadataState_Available(t *testing.T) {
	ctx := context.Background()
	app := quantadmingo.Application{}
	app.SetMetadata(map[string]string{"team": "web"})
	app.SetMetadataAvailable(true)
	data := resource_application.ApplicationModel{Metadata: types.MapUnknown(types.StringType), MetadataAvailable: types.BoolUnknown()}

	if diags := setApplicationMetadataState(ctx, &app, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.MetadataAvailable.ValueBool() {
		t.Errorf("metadata_available = %v, want true", data.MetadataAvailable)
	}
	got := map[string]string{}
	data.Metadata.ElementsAs(ctx, &got, false)
	if got["team"] != "web" || len(got) != 1 {
		t.Errorf("metadata = %v, want map[team:web]", got)
	}
}

func TestSetApplicationMetadataState_AbsentIsNeverUnknown(t *testing.T) {
	ctx := context.Background()
	app := quantadmingo.Application{}
	data := resource_application.ApplicationModel{Metadata: types.MapUnknown(types.StringType), MetadataAvailable: types.BoolUnknown()}

	setApplicationMetadataState(ctx, &app, &data)
	if data.MetadataAvailable.IsUnknown() || !data.MetadataAvailable.IsNull() {
		t.Errorf("metadata_available = %v, want null", data.MetadataAvailable)
	}
	if data.Metadata.IsUnknown() || !data.Metadata.IsNull() {
		t.Errorf("metadata = %v, want null", data.Metadata)
	}
}

func TestSetApplicationMetadataState_UnavailableKeepsConfiguredValue(t *testing.T) {
	ctx := context.Background()
	app := quantadmingo.Application{}
	app.SetMetadataAvailable(false)
	configured, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{"team": "web"})
	data := resource_application.ApplicationModel{Metadata: configured, MetadataAvailable: types.BoolUnknown()}

	setApplicationMetadataState(ctx, &app, &data)
	if data.MetadataAvailable.ValueBool() {
		t.Errorf("metadata_available = %v, want false", data.MetadataAvailable)
	}
	if !data.Metadata.Equal(configured) {
		t.Errorf("metadata = %v, want the configured value", data.Metadata)
	}
}

func TestApplicationMetadataFromPlan(t *testing.T) {
	ctx := context.Background()
	if md, _ := applicationMetadataFromPlan(ctx, types.MapNull(types.StringType)); md != nil {
		t.Errorf("null metadata = %v, want nil", md)
	}
	if md, _ := applicationMetadataFromPlan(ctx, types.MapUnknown(types.StringType)); md != nil {
		t.Errorf("unknown metadata = %v, want nil", md)
	}
	empty, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	if md, _ := applicationMetadataFromPlan(ctx, empty); md == nil || len(md) != 0 {
		t.Errorf("empty metadata = %v, want an empty map (clears all keys)", md)
	}
}
