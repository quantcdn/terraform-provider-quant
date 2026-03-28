package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"github.com/quantcdn/terraform-provider-quant/internal/client"
	"github.com/quantcdn/terraform-provider-quant/internal/resource_volume"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*volumeResource)(nil)
	_ resource.ResourceWithConfigure   = (*volumeResource)(nil)
	_ resource.ResourceWithImportState = (*volumeResource)(nil)
)

func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

type volumeResource struct {
	client *client.Client
}

func (r *volumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_volume.VolumeResourceSchema(ctx)
}

func (r *volumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_volume.VolumeModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVolumeCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_volume.VolumeModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVolumeReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The V3 Volumes API does not have an Update endpoint.
	// Changes require destroy + create (ForceNew).
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"The QuantCloud Volumes API does not support in-place updates. Changes require destroying and recreating the volume.",
	)
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_volume.VolumeModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVolumeDeleteAPI(ctx, r, &data)...)
}

func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "application/environment/volume_id"
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'application/environment/volume_id'.",
		)
		return
	}

	var data resource_volume.VolumeModel
	data.Application = types.StringValue(parts[0])
	data.Environment = types.StringValue(parts[1])
	data.VolumeId = types.StringValue(parts[2])

	resp.Diagnostics.Append(callVolumeReadAPI(ctx, &volumeResource{client: r.client}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) getOrg(data *resource_volume.VolumeModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func callVolumeCreateAPI(ctx context.Context, r *volumeResource, data *resource_volume.VolumeModel) (diags diag.Diagnostics) {
	if data.VolumeName.IsNull() || data.VolumeName.IsUnknown() {
		diags.AddAttributeError(path.Root("volume_name"), "Missing volume_name", "Cannot create a volume without volume_name.")
		return
	}

	sdkReq := quantadmingo.NewCreateVolumeRequest(data.VolumeName.ValueString())

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		sdkReq.Description = *quantadmingo.NewNullableString(strPtr(data.Description.ValueString()))
	}
	if !data.RootDirectory.IsNull() && !data.RootDirectory.IsUnknown() {
		sdkReq.RootDirectory = *quantadmingo.NewNullableString(strPtr(data.RootDirectory.ValueString()))
	}

	org := r.getOrg(data)
	vol, resp, err := r.client.Instance.VolumesAPI.CreateVolume(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString()).CreateVolumeRequest(*sdkReq).Execute()

	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				diags.AddError("Authentication Failed", "Please check your API token.")
				return
			case http.StatusForbidden:
				diags.AddError("Authorization Failed", extractAPIErrorMessage(resp, err))
				return
			case http.StatusBadRequest:
				diags.AddError("Invalid Volume Configuration", extractAPIErrorMessage(resp, err))
				return
			}
		}
		diags.AddError("Unable to Create Volume", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	// Map response to model
	mapVolumeResponse(vol, data)
	data.Organization = types.StringValue(org)

	return
}

func callVolumeReadAPI(ctx context.Context, r *volumeResource, data *resource_volume.VolumeModel) (diags diag.Diagnostics) {
	if data.VolumeId.IsNull() || data.VolumeId.IsUnknown() {
		diags.AddError("Unable to read volume", "The volume_id is unknown.")
		return
	}

	org := r.getOrg(data)
	vol, resp, err := r.client.Instance.VolumesAPI.GetVolume(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString(), data.VolumeId.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			diags.AddError("Volume Not Found", fmt.Sprintf("Volume '%s' not found.", data.VolumeId.ValueString()))
			return
		}
		var errMsg string
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				errMsg = string(body)
			}
		}
		if errMsg == "" {
			errMsg = err.Error()
		}
		diags.AddError("Unable to read volume", fmt.Sprintf("Error: %s", errMsg))
		return
	}

	mapVolumeResponse(vol, data)
	data.Organization = types.StringValue(org)

	return
}

func callVolumeDeleteAPI(ctx context.Context, r *volumeResource, data *resource_volume.VolumeModel) (diags diag.Diagnostics) {
	if data.VolumeName.IsNull() || data.VolumeName.IsUnknown() {
		diags.AddAttributeError(path.Root("volume_name"), "Missing volume_name", "To delete a volume the volume_name must be known.")
		return
	}

	org := r.getOrg(data)
	resp, err := r.client.Instance.VolumesAPI.DeleteVolume(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString(), data.VolumeName.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		diags.AddError("Unable to delete volume", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func mapVolumeResponse(vol *quantadmingo.Volume, data *resource_volume.VolumeModel) {
	if vol.VolumeId != nil {
		data.VolumeId = types.StringValue(*vol.VolumeId)
	} else {
		data.VolumeId = types.StringNull()
	}
	if vol.VolumeName != nil {
		data.VolumeName = types.StringValue(*vol.VolumeName)
	}
	if vol.Description != nil {
		data.Description = types.StringValue(*vol.Description)
	} else if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	if vol.RootDirectory != nil {
		data.RootDirectory = types.StringValue(*vol.RootDirectory)
	} else if data.RootDirectory.IsUnknown() {
		data.RootDirectory = types.StringNull()
	}
	if vol.EnvironmentEfsId != nil {
		data.EnvironmentEfsId = types.StringValue(*vol.EnvironmentEfsId)
	} else {
		data.EnvironmentEfsId = types.StringNull()
	}
	if vol.AccessPointId != nil {
		data.AccessPointId = types.StringValue(*vol.AccessPointId)
	} else {
		data.AccessPointId = types.StringNull()
	}
	if vol.AccessPointArn != nil {
		data.AccessPointArn = types.StringValue(*vol.AccessPointArn)
	} else {
		data.AccessPointArn = types.StringNull()
	}
	if vol.CreatedAt != nil {
		data.CreatedAt = types.StringValue(*vol.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}
