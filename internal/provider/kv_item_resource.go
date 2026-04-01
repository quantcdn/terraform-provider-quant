package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_item"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*kvItemResource)(nil)
	_ resource.ResourceWithConfigure   = (*kvItemResource)(nil)
	_ resource.ResourceWithImportState = (*kvItemResource)(nil)
)

func NewKVItemResource() resource.Resource {
	return &kvItemResource{}
}

type kvItemResource struct {
	client *client.Client
}

func (r *kvItemResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kv_item"
}

func (r *kvItemResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_kv_item.KvItemResourceSchema(ctx)
}

func (r *kvItemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *kvItemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_kv_item.KvItemModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVItemCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvItemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_kv_item.KvItemModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVItemReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvItemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_kv_item.KvItemModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVItemUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Don't read back if it's a secret — the API returns [ENCRYPTED] for secret values.
	// Just store the plan value which we know is correct.

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvItemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_kv_item.KvItemModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVItemDeleteAPI(ctx, r, &data)...)
}

func (r *kvItemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "project/store_id/key"
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'project/store_id/key'.",
		)
		return
	}

	var data resource_kv_item.KvItemModel
	data.Project = types.StringValue(parts[0])
	data.StoreId = types.StringValue(parts[1])
	data.Key = types.StringValue(parts[2])

	resp.Diagnostics.Append(callKVItemReadAPI(ctx, &kvItemResource{client: r.client}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvItemResource) getOrg(data *resource_kv_item.KvItemModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func callKVItemCreateAPI(ctx context.Context, r *kvItemResource, data *resource_kv_item.KvItemModel) (diags diag.Diagnostics) {
	if data.Key.IsNull() || data.Key.IsUnknown() {
		diags.AddAttributeError(path.Root("key"), "Missing key", "Cannot create a KV item without a key.")
		return
	}
	if data.Value.IsNull() || data.Value.IsUnknown() {
		diags.AddAttributeError(path.Root("value"), "Missing value", "Cannot create a KV item without a value.")
		return
	}

	sdkReq := quantadmingo.NewV2StoreItemRequest(data.Key.ValueString(), data.Value.ValueString())

	if !data.Secret.IsNull() && !data.Secret.IsUnknown() {
		v := data.Secret.ValueBool()
		sdkReq.Secret = &v
	}

	org := r.getOrg(data)
	result, resp, err := r.client.Instance.KVAPI.KVItemsCreate(r.client.AuthContext, org, data.Project.ValueString(), data.StoreId.ValueString()).V2StoreItemRequest(*sdkReq).Execute()

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
				diags.AddError("Invalid KV Item Configuration", extractAPIErrorMessage(resp, err))
				return
			}
		}
		diags.AddError("Unable to Create KV Item", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if result.Key != nil {
		data.Key = types.StringValue(*result.Key)
	}
	// Don't overwrite the value from the response for secrets — it may be [ENCRYPTED]
	if result.Value != nil && !data.Secret.ValueBool() {
		data.Value = types.StringValue(*result.Value)
	}
	data.Organization = types.StringValue(org)

	// Map success from response
	if result.Success != nil {
		data.Success = types.BoolValue(*result.Success)
	} else {
		data.Success = types.BoolValue(true) // API call succeeded
	}

	// Resolve any remaining unknown Computed fields
	if data.Secret.IsUnknown() {
		data.Secret = types.BoolValue(false)
	}
	if data.StoreId.IsUnknown() {
		data.StoreId = types.StringNull()
	}
	if data.Project.IsUnknown() {
		data.Project = types.StringNull()
	}

	return
}

func callKVItemReadAPI(ctx context.Context, r *kvItemResource, data *resource_kv_item.KvItemModel) (diags diag.Diagnostics) {
	if data.Key.IsNull() || data.Key.IsUnknown() {
		diags.AddError("Unable to read KV item", "The key is unknown.")
		return
	}

	org := r.getOrg(data)
	item, resp, err := r.client.Instance.KVAPI.KVItemsShow(r.client.AuthContext, org, data.Project.ValueString(), data.StoreId.ValueString(), data.Key.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			diags.AddError("KV Item Not Found", fmt.Sprintf("KV item '%s' not found.", data.Key.ValueString()))
			return
		}
		diags.AddError("Unable to read KV item", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if item.Key != nil {
		data.Key = types.StringValue(*item.Key)
	}

	// Handle the value - it's a union type (string or map)
	if item.Value != nil {
		if item.Value.String != nil {
			val := *item.Value.String
			// If secret and value is [ENCRYPTED], don't overwrite the stored value
			if val == "[ENCRYPTED]" {
				// Keep existing value in state for secrets
				if !data.Secret.IsNull() && data.Secret.ValueBool() {
					// Don't update value — keep what's in state
				} else {
					data.Value = types.StringValue(val)
				}
			} else {
				data.Value = types.StringValue(val)
			}
		} else if item.Value.MapmapOfStringAny != nil {
			// JSON encode the map value
			valJSON, err := json.Marshal(*item.Value.MapmapOfStringAny)
			if err == nil {
				data.Value = types.StringValue(string(valJSON))
			}
		}
	}

	data.Organization = types.StringValue(org)

	// Read succeeded — mark success
	data.Success = types.BoolValue(true)

	// Resolve unknown optional/computed fields
	if data.Secret.IsUnknown() {
		data.Secret = types.BoolValue(false)
	}
	if data.StoreId.IsUnknown() {
		data.StoreId = types.StringNull()
	}
	if data.Project.IsUnknown() {
		data.Project = types.StringNull()
	}

	return
}

func callKVItemUpdateAPI(ctx context.Context, r *kvItemResource, data *resource_kv_item.KvItemModel) (diags diag.Diagnostics) {
	sdkReq := quantadmingo.NewV2StoreItemUpdateRequest(data.Value.ValueString())

	if !data.Secret.IsNull() && !data.Secret.IsUnknown() {
		v := data.Secret.ValueBool()
		sdkReq.Secret = &v
	}

	org := r.getOrg(data)
	_, resp, err := r.client.Instance.KVAPI.KVItemsUpdate(r.client.AuthContext, org, data.Project.ValueString(), data.StoreId.ValueString(), data.Key.ValueString()).V2StoreItemUpdateRequest(*sdkReq).Execute()

	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				diags.AddError("Authentication Failed", "Please check your API token.")
				return
			case http.StatusBadRequest:
				diags.AddError("Invalid KV Item Update", extractAPIErrorMessage(resp, err))
				return
			}
		}
		diags.AddError("Unable to update KV item", fmt.Sprintf("Error: %s", err.Error()))
	}

	data.Organization = types.StringValue(org)
	data.Success = types.BoolValue(true) // Update succeeded

	if data.Secret.IsUnknown() {
		data.Secret = types.BoolValue(false)
	}
	if data.StoreId.IsUnknown() {
		data.StoreId = types.StringNull()
	}
	if data.Project.IsUnknown() {
		data.Project = types.StringNull()
	}

	return
}

func callKVItemDeleteAPI(ctx context.Context, r *kvItemResource, data *resource_kv_item.KvItemModel) (diags diag.Diagnostics) {
	if data.Key.IsNull() || data.Key.IsUnknown() {
		diags.AddAttributeError(path.Root("key"), "Missing key", "To delete a KV item the key must be known.")
		return
	}

	org := r.getOrg(data)
	_, resp, err := r.client.Instance.KVAPI.KVItemsDelete(r.client.AuthContext, org, data.Project.ValueString(), data.StoreId.ValueString(), data.Key.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		diags.AddError("Unable to delete KV item", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}
