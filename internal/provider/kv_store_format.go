package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_store"
)

// The store format is chosen at create. Terraform never upgrades a store: the
// upgrade API (POST .../kv/{store_id}/upgrade) copies the data while writes
// are frozen, which is an operation, not a configuration change.

const kvStoreFormatUpgradeHint = "The format of an existing KV store cannot change in Terraform. " +
	"To upgrade a format 1 store, use the upgrade API (POST /api/v2/organizations/{organization}/projects/{project}/kv/{store_id}/upgrade), " +
	"then set format to the new value."

// setKVStoreFormat sends the planned format on create.
func setKVStoreFormat(data *resource_kv_store.KvStoreModel, req *quantadmingo.V2StoreRequest) {
	if isKnown(data.Format) {
		req.SetFormat(int32(data.Format.ValueInt64()))
	}
}

// ModifyPlan makes a format change or a store_id change on an existing store
// a plan-time error, never a replacement: a replacement would delete the
// store and its data, and a store_id change would point the resource at
// another store.
func (r *kvStoreResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state resource_kv_store.KvStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(checkKVStoreFormatChange(plan.Format, state.Format)...)
	resp.Diagnostics.Append(checkKVStoreIDChange(plan.StoreId, state.StoreId)...)
}

func checkKVStoreFormatChange(planned, prior types.Int64) diag.Diagnostics {
	var diags diag.Diagnostics
	// A state from a provider version before format has no value: Update
	// compares the plan with the real format instead.
	if !isKnown(planned) || !isKnown(prior) || planned.Equal(prior) {
		return diags
	}
	diags.AddAttributeError(path.Root("format"), "KV store format cannot change",
		fmt.Sprintf("The store is format %d and the configuration asks for format %d. %s", prior.ValueInt64(), planned.ValueInt64(), kvStoreFormatUpgradeHint))
	return diags
}

func checkKVStoreIDChange(planned, prior types.String) diag.Diagnostics {
	var diags diag.Diagnostics
	if !isKnown(planned) || !isKnown(prior) || planned.Equal(prior) {
		return diags
	}
	diags.AddAttributeError(path.Root("store_id"), "KV store store_id cannot change",
		fmt.Sprintf("The resource manages store %q and the configuration asks for store %q. The API sets store_id; Terraform does not move a resource to another store. "+
			"Remove store_id from the configuration, or import the other store into a new resource.", prior.ValueString(), planned.ValueString()))
	return diags
}

// Update applies only the changes allowed in place: format, when it equals
// the real format (for example a state written before the provider knew the
// format). id, store_id, project and organization come from the prior state,
// so nothing is unknown after apply, also with -refresh=false.
func (r *kvStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resource_kv_store.KvStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Id, plan.StoreId, plan.Project, plan.Organization = state.Id, state.StoreId, state.Project, state.Organization
	actual, httpResp, err := readKVStoreFormat(r, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the KV store format", extractAPIErrorMessage(httpResp, err))
		return
	}
	if isKnown(plan.Format) && plan.Format.ValueInt64() != actual {
		resp.Diagnostics.AddAttributeError(path.Root("format"), "KV store format cannot change",
			fmt.Sprintf("The store is format %d. %s", actual, kvStoreFormatUpgradeHint))
		return
	}
	plan.Format = types.Int64Value(actual)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// refreshKVStoreFormat sets format from the upgrade state of the store. The
// store show response has no format. When the upgrade endpoint fails (403,
// 5xx), the store keeps its prior format and the read gives a warning.
func refreshKVStoreFormat(r *kvStoreResource, data *resource_kv_store.KvStoreModel) diag.Diagnostics {
	var diags diag.Diagnostics
	actual, httpResp, err := readKVStoreFormat(r, data)
	if err != nil {
		diags.AddWarning("Unable to read the KV store format",
			fmt.Sprintf("The provider keeps the format from state for store %s. %s", data.StoreId.ValueString(), extractAPIErrorMessage(httpResp, err)))
		return diags
	}
	data.Format = types.Int64Value(actual)
	return diags
}

// readKVStoreFormat is the real format of the store. A store that the
// upgrade endpoint does not know (404) is format 1.
func readKVStoreFormat(r *kvStoreResource, data *resource_kv_store.KvStoreModel) (int64, *http.Response, error) {
	state, httpResp, err := r.client.Instance.KVAPI.KVUpgradeShow(r.client.AuthContext, r.getOrg(data), data.Project.ValueString(), data.StoreId.ValueString()).Execute()
	if kvAPIStatus(httpResp) == http.StatusNotFound {
		return 1, httpResp, nil
	}
	if err != nil {
		return 0, httpResp, err
	}
	return max(int64(state.GetFormat()), 1), httpResp, nil
}
