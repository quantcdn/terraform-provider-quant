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

// ModifyPlan makes a format change on an existing store a plan-time error,
// never a replacement: a replacement would delete the store and its data.
func (r *kvStoreResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planned, prior types.Int64
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("format"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("format"), &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A state from a provider version before format has no value: Update
	// compares the plan with the real format instead.
	if !isKnown(planned) || !isKnown(prior) || planned.Equal(prior) {
		return
	}
	resp.Diagnostics.AddAttributeError(path.Root("format"), "KV store format cannot change",
		fmt.Sprintf("The store is format %d and the configuration asks for format %d. %s", prior.ValueInt64(), planned.ValueInt64(), kvStoreFormatUpgradeHint))
}

// Update is reached only by format (name, project and organization force a
// replacement). It adopts a planned format that equals the real format, for
// example a state written before the provider knew the format.
func (r *kvStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resource_kv_store.KvStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	actual, diags := readKVStoreFormat(r, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if actual == 0 {
		actual = 1 // the upgrade endpoint does not know the store: format 1
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
// store show response has no format. A store that the upgrade endpoint does
// not know (404) keeps its prior format, or format 1 when there is none.
func refreshKVStoreFormat(r *kvStoreResource, data *resource_kv_store.KvStoreModel) diag.Diagnostics {
	actual, diags := readKVStoreFormat(r, data)
	if diags.HasError() {
		return diags
	}
	if actual == 0 {
		if !isKnown(data.Format) {
			data.Format = types.Int64Value(1)
		}
		return diags
	}
	data.Format = types.Int64Value(actual)
	return diags
}

// readKVStoreFormat is the real format of the store, or 0 when the upgrade
// endpoint answers 404.
func readKVStoreFormat(r *kvStoreResource, data *resource_kv_store.KvStoreModel) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	state, httpResp, err := r.client.Instance.KVAPI.KVUpgradeShow(r.client.AuthContext, r.getOrg(data), data.Project.ValueString(), data.StoreId.ValueString()).Execute()
	if kvAPIStatus(httpResp) == http.StatusNotFound {
		return 0, diags
	}
	if err != nil {
		diags.AddError("Unable to read the KV store format", extractAPIErrorMessage(httpResp, err))
		return 0, diags
	}
	if state.GetFormat() < 1 {
		return 1, diags
	}
	return int64(state.GetFormat()), diags
}
