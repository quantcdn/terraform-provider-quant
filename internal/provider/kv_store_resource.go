package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_kv_store"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*kvStoreResource)(nil)
	_ resource.ResourceWithConfigure   = (*kvStoreResource)(nil)
	_ resource.ResourceWithImportState = (*kvStoreResource)(nil)
)

func NewKVStoreResource() resource.Resource {
	return &kvStoreResource{}
}

type kvStoreResource struct {
	client *client.Client
}

func (r *kvStoreResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kv_store"
}

func (r *kvStoreResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_kv_store.KvStoreResourceSchema(ctx)
}

func (r *kvStoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *kvStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_kv_store.KvStoreModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVStoreCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_kv_store.KvStoreModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVStoreReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// KV stores have no update API — name changes require destroy + create.
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"KV stores cannot be updated in-place. Changes require destroying and recreating the store.",
	)
}

func (r *kvStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_kv_store.KvStoreModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callKVStoreDeleteAPI(ctx, r, &data)...)
}

func (r *kvStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "project/store_id"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'project/store_id'.",
		)
		return
	}

	var data resource_kv_store.KvStoreModel
	data.Project = types.StringValue(parts[0])
	data.StoreId = types.StringValue(parts[1])

	resp.Diagnostics.Append(callKVStoreReadAPI(ctx, &kvStoreResource{client: r.client}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvStoreResource) getOrg(data *resource_kv_store.KvStoreModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func callKVStoreCreateAPI(ctx context.Context, r *kvStoreResource, data *resource_kv_store.KvStoreModel) (diags diag.Diagnostics) {
	if data.Name.IsNull() || data.Name.IsUnknown() {
		diags.AddAttributeError(path.Root("name"), "Missing name", "Cannot create a KV store without a name.")
		return
	}
	if data.Project.IsNull() || data.Project.IsUnknown() {
		diags.AddAttributeError(path.Root("project"), "Missing project", "Cannot create a KV store without a project.")
		return
	}

	sdkReq := quantadmingo.NewV2StoreRequest(data.Name.ValueString())

	org := r.getOrg(data)
	store, resp, err := r.client.Instance.KVAPI.KVCreate(r.client.AuthContext, org, data.Project.ValueString()).V2StoreRequest(*sdkReq).Execute()

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
				diags.AddError("Invalid KV Store Configuration", extractAPIErrorMessage(resp, err))
				return
			}
		}
		diags.AddError("Unable to Create KV Store", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	data.StoreId = types.StringValue(store.GetId())
	data.Name = types.StringValue(store.GetName())
	data.Organization = types.StringValue(org)

	return
}

func callKVStoreReadAPI(ctx context.Context, r *kvStoreResource, data *resource_kv_store.KvStoreModel) (diags diag.Diagnostics) {
	if data.StoreId.IsNull() || data.StoreId.IsUnknown() {
		diags.AddError("Unable to read KV store", "The store_id is unknown.")
		return
	}

	org := r.getOrg(data)
	store, resp, err := r.client.Instance.KVAPI.KVShow(r.client.AuthContext, org, data.Project.ValueString(), data.StoreId.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			diags.AddError("KV Store Not Found", fmt.Sprintf("KV store '%s' not found.", data.StoreId.ValueString()))
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
		diags.AddError("Unable to read KV store", fmt.Sprintf("Error: %s", errMsg))
		return
	}

	data.StoreId = types.StringValue(store.GetId())
	data.Name = types.StringValue(store.GetName())
	data.Organization = types.StringValue(r.getOrg(data))

	return
}

func callKVStoreDeleteAPI(ctx context.Context, r *kvStoreResource, data *resource_kv_store.KvStoreModel) (diags diag.Diagnostics) {
	if data.StoreId.IsNull() || data.StoreId.IsUnknown() {
		diags.AddAttributeError(path.Root("store_id"), "Missing store_id", "To delete a KV store the store_id must be known.")
		return
	}

	org := r.getOrg(data)
	resp, err := r.client.Instance.KVAPI.KVDelete(r.client.AuthContext, org, data.Project.ValueString(), data.StoreId.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		diags.AddError("Unable to delete KV store", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}
