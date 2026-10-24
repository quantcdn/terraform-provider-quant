package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_ledger"
)

// A ledger is never replaced by a plan: a retained ledger cannot be dropped,
// and a new ledger needs an empty collection. So a change to the definition
// is a plan-time error, and so is a lowering of a raise-only attribute.

// lockRank orders lock_mode: governance can become compliance, never back.
var lockRank = map[string]int{"governance": 1, "compliance": 2}

// ledgerImmutable lists the attributes that are part of the signed
// definition or of the ledger path, with their model values.
func ledgerImmutable(m *resource_kv_ledger.KvLedgerModel) map[string]attr.Value {
	return map[string]attr.Value{
		"organization": m.Organization, "project": m.Project, "store_id": m.StoreId,
		"name": m.Name, "ledger_name": m.LedgerName, "collection": m.Collection,
		"personal_fields": m.PersonalFields, "subject_field": m.SubjectField,
		"subject_mode": m.SubjectMode, "subject_case": m.SubjectCase,
		"anchor": m.Anchor, "writer_policy": m.WriterPolicy,
	}
}

// ModifyPlan turns what the API would refuse into plan-time errors.
func (r *kvLedgerResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var state resource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(keepPriorLedgerObjects(ctx, resp, state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	switch state.State.ValueString() {
	case kvLedgerFailed:
		// A FAILED ledger is declared again: drop it (allowed) and create.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("state"), types.StringUnknown())...)
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("state"))
		return
	case kvLedgerDropped:
		resp.Diagnostics.AddError("KV ledger is dropped", fmt.Sprintf("Ledger %s is DROPPED and its name cannot be used again. Remove it from the configuration, or declare a ledger with a new name.", state.Name.ValueString()))
		return
	}
	var plan resource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(resp.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(checkLedgerImmutables(plan, state)...)
	resp.Diagnostics.Append(checkLedgerRaise(plan, state)...)
	if !resp.Diagnostics.HasError() && !ledgerInputsChanged(plan, state) {
		// No input changed. The framework may still have marked computed
		// attributes unknown (the nested anchor.prefix default makes it see
		// a change), which would plan an empty update on every run.
		resp.Plan.Raw = req.State.Raw
	}
}

// ledgerInputsChanged is true when a configurable attribute has a planned
// value that differs from state. Unknown planned values count as no change.
func ledgerInputsChanged(plan, state resource_kv_ledger.KvLedgerModel) bool {
	planned, prior := ledgerImmutable(&plan), ledgerImmutable(&state)
	planned["chains"], prior["chains"] = plan.Chains, state.Chains
	planned["retention_days"], prior["retention_days"] = plan.RetentionDays, state.RetentionDays
	planned["lock_mode"], prior["lock_mode"] = plan.LockMode, state.LockMode
	for name, v := range planned {
		if !v.IsUnknown() && !v.Equal(prior[name]) {
			return true
		}
	}
	return false
}

// keepPriorLedgerObjects fills unknown parts of the planned anchor and writer
// policy from state. The generated nested types take no plan modifiers, and
// without this a raise would plan them as "known after apply".
func keepPriorLedgerObjects(ctx context.Context, resp *resource.ModifyPlanResponse, state resource_kv_ledger.KvLedgerModel) diag.Diagnostics {
	var plan resource_kv_ledger.KvLedgerModel
	diags := resp.Plan.Get(ctx, &plan)
	if diags.HasError() {
		return diags
	}
	for name, pair := range map[string][2]basetypes.ObjectValuable{
		"anchor":        {plan.Anchor, state.Anchor},
		"writer_policy": {plan.WriterPolicy, state.WriterPolicy},
	} {
		merged, changed, d := mergeUnknownFromPrior(ctx, pair[0], pair[1])
		diags.Append(d...)
		if changed {
			diags.Append(resp.Plan.SetAttribute(ctx, path.Root(name), merged)...)
		}
	}
	return diags
}

// mergeUnknownFromPrior is planned with its unknown parts taken from prior.
func mergeUnknownFromPrior(ctx context.Context, planned, prior basetypes.ObjectValuable) (attr.Value, bool, diag.Diagnostics) {
	p, diags := planned.ToObjectValue(ctx)
	s, d := prior.ToObjectValue(ctx)
	diags.Append(d...)
	if diags.HasError() || s.IsNull() || s.IsUnknown() || p.IsNull() {
		return planned, false, diags
	}
	if p.IsUnknown() {
		return prior, true, diags
	}
	attrs, changed := knownAttributes(p.Attributes(), s.Attributes())
	if !changed {
		return planned, false, diags
	}
	obj, d := types.ObjectValue(p.AttributeTypes(ctx), attrs)
	diags.Append(d...)
	typable, ok := planned.Type(ctx).(basetypes.ObjectTypable)
	if diags.HasError() || !ok {
		return planned, false, diags
	}
	v, d := typable.ValueFromObject(ctx, obj)
	diags.Append(d...)
	return v, true, diags
}

func knownAttributes(planned, prior map[string]attr.Value) (map[string]attr.Value, bool) {
	out := make(map[string]attr.Value, len(planned))
	changed := false
	for k, v := range planned {
		if pv, ok := prior[k]; ok && v.IsUnknown() {
			v, changed = pv, true
		}
		out[k] = v
	}
	return out, changed
}

func checkLedgerImmutables(plan, state resource_kv_ledger.KvLedgerModel) diag.Diagnostics {
	var diags diag.Diagnostics
	prior := ledgerImmutable(&state)
	for name, planned := range ledgerImmutable(&plan) {
		if planned.IsUnknown() || planned.Equal(prior[name]) {
			continue
		}
		diags.AddAttributeError(path.Root(name), fmt.Sprintf("KV ledger %s cannot change", name),
			fmt.Sprintf("Ledger %s: %s is part of the ledger definition and cannot change, and Terraform does not replace a ledger. Declare a new ledger with a new name on an empty collection instead.", state.Name.ValueString(), name))
	}
	return diags
}

func checkLedgerRaise(plan, state resource_kv_ledger.KvLedgerModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if lowered(plan.Chains, state.Chains) {
		diags.AddAttributeError(path.Root("chains"), "chains can only go up", fmt.Sprintf("chains is %d and cannot go down.", state.Chains.ValueInt64()))
	}
	if lowered(plan.RetentionDays, state.RetentionDays) {
		diags.AddAttributeError(path.Root("retention_days"), "retention_days can only go up", fmt.Sprintf("retention_days is %d and cannot go down.", state.RetentionDays.ValueInt64()))
	}
	if isKnown(plan.LockMode) && lockRank[plan.LockMode.ValueString()] < lockRank[state.LockMode.ValueString()] {
		diags.AddAttributeError(path.Root("lock_mode"), "lock_mode cannot go from compliance to governance", "S3 Object Lock compliance mode cannot be lowered.")
	}
	return diags
}

func lowered(plan, state types.Int64) bool {
	return isKnown(plan) && isKnown(state) && plan.ValueInt64() < state.ValueInt64()
}
