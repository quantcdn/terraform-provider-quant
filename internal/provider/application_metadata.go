package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_application"
)

// applicationMetadataFromPlan returns the configured metadata map, or nil when
// metadata is null or unknown (not managed by this configuration).
func applicationMetadataFromPlan(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]string{}
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// setApplicationMetadataState copies metadata and metadata_available from the
// API answer. Both are Optional/Computed or Computed, so they must be known
// after every apply: an absent value becomes null, never unknown.
func setApplicationMetadataState(ctx context.Context, app *quantadmingo.Application, data *resource_application.ApplicationModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if available, ok := app.GetMetadataAvailableOk(); ok {
		data.MetadataAvailable = types.BoolValue(*available)
	} else {
		data.MetadataAvailable = types.BoolNull()
	}

	if md, ok := app.GetMetadataOk(); ok && app.GetMetadataAvailable() {
		v, d := types.MapValueFrom(ctx, types.StringType, *md)
		diags.Append(d...)
		data.Metadata = v
	} else if data.Metadata.IsUnknown() {
		// Metadata could not be read: keep a configured value, never leave it unknown.
		data.Metadata = types.MapNull(types.StringType)
	}
	return diags
}

// putApplicationMetadata replaces the application's metadata with the
// configured map. It does nothing when metadata is not configured.
func putApplicationMetadata(ctx context.Context, r *applicationResource, data *resource_application.ApplicationModel) diag.Diagnostics {
	md, diags := applicationMetadataFromPlan(ctx, data.Metadata)
	if diags.HasError() || md == nil {
		return diags
	}
	_, resp, err := r.client.Instance.ApplicationsAPI.PutApplicationMetadata(r.client.AuthContext, r.getOrg(data), data.AppName.ValueString()).RequestBody(md).Execute()
	if err != nil {
		diags.AddError("Unable to update application metadata", fmt.Sprintf("Error: %s", extractAPIErrorMessage(resp, err)))
	}
	return diags
}
