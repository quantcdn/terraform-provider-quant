package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_ledger"
)

var (
	_ resource.Resource                = (*kvLedgerResource)(nil)
	_ resource.ResourceWithConfigure   = (*kvLedgerResource)(nil)
	_ resource.ResourceWithImportState = (*kvLedgerResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*kvLedgerResource)(nil)
)

// KVLedgerWaitTimeout bounds the wait for a declared ledger to leave PENDING
// (the API holds it PENDING for about 75 seconds). QUANTCDN_KV_LEDGER_TIMEOUT
// (a Go duration) overrides it. KVLedgerWaitInterval is the poll interval.
var (
	KVLedgerWaitTimeout  = 5 * time.Minute
	KVLedgerWaitInterval = 10 * time.Second
)

const (
	kvLedgerActive  = "ACTIVE"
	kvLedgerPending = "PENDING"
	kvLedgerFailed  = "FAILED"
	kvLedgerDropped = "DROPPED"
)

func NewKVLedgerResource() resource.Resource { return &kvLedgerResource{} }

type kvLedgerResource struct{ client *client.Client }

func (r *kvLedgerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kv_ledger"
}

func (r *kvLedgerResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_kv_ledger.KvLedgerResourceSchema(ctx)
	s.Description = "A tamper-evident ledger on a collection of a format 2 KV store. The definition (name, collection, personal fields, subject field, anchor, writer policy) never changes; chains, retention_days and lock_mode can only go up. A ledger cannot be dropped before its archive retention ends."
	s.MarkdownDescription = s.Description
	addKeepState(s.Attributes, "organization", "project", "store_id", "ledger_name", "personal_fields", "subject_field", "created_at")
	resp.Schema = s
}

func (r *kvLedgerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *kvLedgerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(requireKVPath(data.Project, data.StoreId)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Organization = types.StringValue(orgOr(data.Organization, r.client.Organization))
	body, diags := kvLedgerDeclareRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := kvRetry429(ctx, kvLedgerTimeout(), func() (*quantadmingo.V2KvLedger, *http.Response, error) {
		return r.client.Instance.KVAPI.KVLedgersCreate(r.client.AuthContext, data.Organization.ValueString(), data.Project.ValueString(), data.StoreId.ValueString()).
			V2KvLedgerAdminDeclareRequest(*body).Execute()
	})
	if err != nil && !kvAPIOK(httpResp) {
		resp.Diagnostics.AddError("Unable to declare KV ledger", kvAPIErrorDetail(httpResp, err))
		return
	}
	// A wait that ends before the ledger leaves PENDING returns an error
	// before anything is saved. Saved state with an error is tainted, and the
	// next apply would replace it: drop the declared ledger, which could then
	// be retained or lose entries. Unsaved, the ledger is only imported.
	final, diags := r.waitDeclared(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if final == nil {
		return
	}
	resp.Diagnostics.Append(setKVLedgerState(ctx, &data, final)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if final.GetState() != kvLedgerActive {
		resp.Diagnostics.AddError("KV ledger declare failed", fmt.Sprintf("Ledger %s is %s: %s. The next apply declares it again.",
			data.Name.ValueString(), final.GetState(), final.GetError()))
	}
}

func (r *kvLedgerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	l, httpResp, err := r.show(&data)
	if kvAPIStatus(httpResp) == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KV ledger", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(setKVLedgerState(ctx, &data, l)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update raises chains, retention_days or lock_mode. ModifyPlan has already
// refused every other change, and every lowering.
func (r *kvLedgerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if raise, changed := kvLedgerRaise(plan, state); changed {
		_, httpResp, err := kvRetry429(ctx, kvLedgerTimeout(), func() (*quantadmingo.V2KvLedger, *http.Response, error) {
			return r.client.Instance.KVAPI.KVLedgersUpdate(r.client.AuthContext, r.org(&state), state.Project.ValueString(), state.StoreId.ValueString(), state.Name.ValueString()).
				V2KvLedgerAdminRaiseRequest(raise).Execute()
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to raise KV ledger", kvAPIErrorDetail(httpResp, err))
			return
		}
	}
	l, httpResp, err := r.show(&state)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KV ledger", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(setKVLedgerState(ctx, &state, l)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the ledger. The API refuses while the archive retention runs;
// the diagnostic then says how to stop managing the ledger without a drop.
func (r *kvLedgerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.State.ValueString() == kvLedgerDropped {
		return
	}
	drop, diags := r.dropAllowed(&data)
	resp.Diagnostics.Append(diags...)
	if !drop {
		return
	}
	_, httpResp, err := kvRetry429(ctx, kvLedgerTimeout(), func() (*quantadmingo.V2KvLedger, *http.Response, error) {
		return r.client.Instance.KVAPI.KVLedgersDelete(r.client.AuthContext, r.org(&data), data.Project.ValueString(), data.StoreId.ValueString(), data.Name.ValueString()).Execute()
	})
	if err == nil || kvAPIOK(httpResp) || kvAPIStatus(httpResp) == http.StatusNotFound {
		return
	}
	if kvAPIStatus(httpResp) == http.StatusTooManyRequests {
		resp.Diagnostics.AddError("KV ledger cannot be dropped", kvAPIErrorDetail(httpResp, err))
		return
	}
	resp.Diagnostics.AddError("KV ledger cannot be dropped", kvLedgerDropRefused(data.Name.ValueString(), err))
}

// dropAllowed reads the ledger again before the drop. It is false when the
// ledger is gone, and false with an error when the prior state is FAILED
// (the plan replaces it) but the server no longer shows FAILED: a drop would
// destroy a ledger that works.
func (r *kvLedgerResource) dropAllowed(d *resource_kv_ledger.KvLedgerModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	l, httpResp, err := r.show(d)
	switch {
	case kvAPIStatus(httpResp) == http.StatusNotFound:
		return false, diags
	case err != nil:
		diags.AddError("Unable to read KV ledger before the drop", extractAPIErrorMessage(httpResp, err))
		return false, diags
	case l.GetState() == kvLedgerDropped:
		return false, diags
	case d.State.ValueString() == kvLedgerFailed && l.GetState() != kvLedgerFailed:
		diags.AddError("KV ledger is not dropped", fmt.Sprintf("The plan replaces ledger %s because it was FAILED, but the server now shows it %s. "+
			"The provider does not drop a ledger that is not FAILED. Run terraform plan again.", d.Name.ValueString(), l.GetState()))
		return false, diags
	}
	return true, diags
}

func (r *kvLedgerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ref, err := parseKVImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	var data resource_kv_ledger.KvLedgerModel
	data.Organization = types.StringValue(orgOr(types.StringValue(ref.Organization), r.client.Organization))
	data.Project, data.StoreId, data.Name = types.StringValue(ref.Project), types.StringValue(ref.StoreID), types.StringValue(ref.Name)
	l, httpResp, err := r.show(&data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import KV ledger", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(setKVLedgerState(ctx, &data, l)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvLedgerResource) org(d *resource_kv_ledger.KvLedgerModel) string {
	return orgOr(d.Organization, r.client.Organization)
}

func (r *kvLedgerResource) show(d *resource_kv_ledger.KvLedgerModel) (*quantadmingo.V2KvLedgerDetail, *http.Response, error) {
	return r.client.Instance.KVAPI.KVLedgersShow(r.client.AuthContext, r.org(d), d.Project.ValueString(), d.StoreId.ValueString(), d.Name.ValueString()).Execute()
}

// waitDeclared polls the ledger until it leaves PENDING.
func (r *kvLedgerResource) waitDeclared(ctx context.Context, d *resource_kv_ledger.KvLedgerModel) (*quantadmingo.V2KvLedgerDetail, diag.Diagnostics) {
	var diags diag.Diagnostics
	timeout := kvLedgerTimeout()
	deadline := time.Now().Add(timeout)
	for {
		l, _, err := r.show(d)
		if err == nil && l.GetState() != kvLedgerPending {
			return l, diags
		}
		if time.Now().After(deadline) {
			diags.AddError("KV ledger wait timed out", fmt.Sprintf("Ledger %s is still PENDING after %s (set QUANTCDN_KV_LEDGER_TIMEOUT to wait longer). %s", d.Name.ValueString(), timeout, r.importHint(d)))
			return nil, diags
		}
		if serr := sleepCtx(ctx, KVLedgerWaitInterval); serr != nil {
			diags.AddError("KV ledger wait cancelled", fmt.Sprintf("%s. %s", serr.Error(), r.importHint(d)))
			return nil, diags
		}
	}
}

// importHint says that a declared ledger exists and how to import it.
func (r *kvLedgerResource) importHint(d *resource_kv_ledger.KvLedgerModel) string {
	id := fmt.Sprintf("%s/%s/%s", d.Project.ValueString(), d.StoreId.ValueString(), d.Name.ValueString())
	if org := r.org(d); org != r.client.Organization {
		id = org + "/" + id
	}
	return fmt.Sprintf("The provider did not save it in state, but the ledger exists on the server; import it with "+
		"terraform import quant_kv_ledger.<name> %s (or an import block) before the next apply.", id)
}

// kvLedgerTimeout is the wait timeout of a ledger: for the declare, and for
// 429 retries of the declare, the raise and the drop.
func kvLedgerTimeout() time.Duration {
	return waitTimeout("QUANTCDN_KV_LEDGER_TIMEOUT", KVLedgerWaitTimeout)
}

// kvLedgerDropRefused explains a refused drop and the ways to stop managing
// the ledger without dropping it.
func kvLedgerDropRefused(name string, err error) string {
	e := parseKVAPIError(err)
	if e.Code != "LEDGER_RETAINED" {
		return fmt.Sprintf("Ledger %s: %s %s (%v)", name, e.Code, e.Message, err)
	}
	if e.Reason == "unarchived" || e.Reason == "retention_unknown" {
		return fmt.Sprintf("Ledger %s still has entries that are not archived (%s %s). Try again in a few minutes. (%s)", name, e.Collection, e.Shard, e.Message)
	}
	return fmt.Sprintf("Ledger %s is retained until %s (retainUntil) and cannot be dropped before then. "+
		"To stop managing it without dropping it, use a Terraform removed block:\n\n"+
		"  removed {\n    from = quant_kv_ledger.<name>\n    lifecycle {\n      destroy = false\n    }\n  }\n\n"+
		"or `terraform state rm`. In Pulumi, set retainOnDelete: true or run `pulumi state delete`. (%s: %s)",
		name, e.RetainUntil, e.Code, e.Message)
}
