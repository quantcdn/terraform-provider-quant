package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_index"
)

var (
	_ resource.Resource                = (*kvIndexResource)(nil)
	_ resource.ResourceWithConfigure   = (*kvIndexResource)(nil)
	_ resource.ResourceWithImportState = (*kvIndexResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*kvIndexResource)(nil)
)

// KVIndexWaitTimeout bounds the wait for ACTIVE after a create and for the
// index to be gone after a delete. QUANTCDN_KV_INDEX_TIMEOUT (a Go duration,
// for example "30m") overrides it. KVIndexWaitInterval is the poll interval.
var (
	KVIndexWaitTimeout  = 10 * time.Minute
	KVIndexWaitInterval = 5 * time.Second
)

const (
	kvIndexActive   = "ACTIVE"
	kvIndexBuilding = "BUILDING"
	kvIndexDropping = "DROPPING"
	kvIndexFailed   = "FAILED"
)

// kvIndexCountAttr is the Terraform name of the API field count.
const kvIndexCountAttr = "live_count"

// kvIndexModel is resource_kv_index.KvIndexModel with count renamed to live_count.
type kvIndexModel struct {
	Collection   types.String `tfsdk:"collection"`
	Conflicts    types.List   `tfsdk:"conflicts"`
	LiveCount    types.Bool   `tfsdk:"live_count"`
	CreatedAt    types.String `tfsdk:"created_at"`
	Error        types.String `tfsdk:"error"`
	Fields       types.List   `tfsdk:"fields"`
	IndexName    types.String `tfsdk:"index_name"`
	Name         types.String `tfsdk:"name"`
	Organization types.String `tfsdk:"organization"`
	Project      types.String `tfsdk:"project"`
	Slot         types.Int64  `tfsdk:"slot"`
	SortBy       types.String `tfsdk:"sort_by"`
	State        types.String `tfsdk:"state"`
	StoreId      types.String `tfsdk:"store_id"`
	Unique       types.Bool   `tfsdk:"unique"`
}

func NewKVIndexResource() resource.Resource { return &kvIndexResource{} }

type kvIndexResource struct{ client *client.Client }

func (r *kvIndexResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kv_index"
}

func (r *kvIndexResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_kv_index.KvIndexResourceSchema(ctx)
	s.Description = "A secondary index of a format 2 KV store. An index definition cannot change: every input forces a replacement (drop, then declare again)."
	s.MarkdownDescription = s.Description
	// "count" is a reserved root attribute name in Terraform, so the
	// generated schema does not load as it is. The API field is count.
	s.Attributes[kvIndexCountAttr] = s.Attributes["count"]
	delete(s.Attributes, "count")
	addReplaceKeepingState(s.Attributes, "organization", "project", "store_id", "name", "index_name",
		"collection", "fields", "sort_by", "unique", kvIndexCountAttr)
	addKeepState(s.Attributes, "created_at", "slot")
	resp.Schema = s
}

func (r *kvIndexResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *kvIndexResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data kvIndexModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(requireKVPath(data.Project, data.StoreId)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Organization = types.StringValue(orgOr(data.Organization, r.client.Organization))
	body, diags := kvIndexRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, httpResp, err := kvRetry429(ctx, kvIndexTimeout(), func() (*quantadmingo.V2KvIndex, *http.Response, error) {
		return r.client.Instance.KVAPI.KVIndexesCreate(r.client.AuthContext, data.Organization.ValueString(), data.Project.ValueString(), data.StoreId.ValueString()).
			V2KvIndexRequest(body).Execute()
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to declare KV index", kvAPIErrorDetail(httpResp, err))
		return
	}
	// Save the BUILDING index first: when the wait fails, the index stays in
	// state, tainted, and the next apply drops and declares it again.
	resp.Diagnostics.Append(setKVIndexState(ctx, &data, created)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	final, diags := r.waitFor(ctx, &data, indexBuilt)
	resp.Diagnostics.Append(diags...)
	if final == nil {
		return
	}
	resp.Diagnostics.Append(setKVIndexState(ctx, &data, final)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if final.GetState() != kvIndexActive {
		resp.Diagnostics.AddError("KV index build failed", fmt.Sprintf("Index %s is %s: %s %v. The next apply drops the index and declares it again.",
			data.Name.ValueString(), final.GetState(), final.GetError(), final.GetConflicts()))
	}
}

func (r *kvIndexResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data kvIndexModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A DROPPING index stays in state until it is gone: ModifyPlan then
	// replaces it, and the replacement waits for the drop to end before it
	// declares the name again.
	idx, httpResp, err := r.show(&data)
	if kvAPIStatus(httpResp) == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read KV index", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(setKVIndexState(ctx, &data, idx)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvIndexResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("KV index cannot be updated", "Every KV index input forces a replacement. Report this as a provider bug.")
}

func (r *kvIndexResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data kvIndexModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A DROPPING index is already on its way out: do not drop it again.
	if data.State.ValueString() != kvIndexDropping {
		_, httpResp, err := kvRetry429(ctx, kvIndexTimeout(), func() (*quantadmingo.V2KvIndex, *http.Response, error) {
			return r.client.Instance.KVAPI.KVIndexesDelete(r.client.AuthContext, r.org(&data), data.Project.ValueString(), data.StoreId.ValueString(), data.Name.ValueString()).Execute()
		})
		if kvAPIStatus(httpResp) == http.StatusNotFound {
			return // already gone
		}
		if err != nil && !kvAPIOK(httpResp) {
			resp.Diagnostics.AddError("Unable to drop KV index", kvAPIErrorDetail(httpResp, err))
			return
		}
	}
	// Wait until the index is gone, so a replacement can declare the same name.
	_, diags := r.waitFor(ctx, &data, indexGone)
	resp.Diagnostics.Append(diags...)
}

// ModifyPlan replaces a FAILED index (also one just imported) and a
// DROPPING index, with a warning. The replacement drops a FAILED index, or
// waits until a DROPPING index is gone, then declares it again.
func (r *kvIndexResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var state kvIndexModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	detail := kvIndexReplaceReason(&state)
	if detail == "" {
		return
	}
	resp.Diagnostics.AddWarning("KV index is replaced", detail)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("state"), types.StringUnknown())...)
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("state"))
}

// kvIndexReplaceReason explains why an index in this state is replaced, or
// is empty when it is not.
func kvIndexReplaceReason(d *kvIndexModel) string {
	switch d.State.ValueString() {
	case kvIndexFailed:
		return fmt.Sprintf("Index %s is FAILED (%s %s). The apply drops it and declares it again.", d.Name.ValueString(), d.Error.ValueString(), d.Conflicts.String())
	case kvIndexDropping:
		return fmt.Sprintf("Index %s is DROPPING. The apply waits until the drop ends, then declares it again.", d.Name.ValueString())
	}
	return ""
}

func (r *kvIndexResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ref, err := parseKVImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	data := kvIndexModel{Organization: types.StringValue(orgOr(types.StringValue(ref.Organization), r.client.Organization)),
		Project: types.StringValue(ref.Project), StoreId: types.StringValue(ref.StoreID), Name: types.StringValue(ref.Name)}
	idx, httpResp, err := r.show(&data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import KV index", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(setKVIndexState(ctx, &data, idx)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *kvIndexResource) org(d *kvIndexModel) string {
	return orgOr(d.Organization, r.client.Organization)
}

func (r *kvIndexResource) show(d *kvIndexModel) (*quantadmingo.V2KvIndex, *http.Response, error) {
	return r.client.Instance.KVAPI.KVIndexesShow(r.client.AuthContext, r.org(d), d.Project.ValueString(), d.StoreId.ValueString(), d.Name.ValueString()).Execute()
}

// kvIndexTimeout is the wait timeout of an index: for the build, for the
// drop, and for 429 retries of the declare and the drop.
func kvIndexTimeout() time.Duration {
	return waitTimeout("QUANTCDN_KV_INDEX_TIMEOUT", KVIndexWaitTimeout)
}

// indexBuilt is true when the build has ended (ACTIVE or FAILED).
func indexBuilt(i *quantadmingo.V2KvIndex, status int, err error) bool {
	return err == nil && status == http.StatusOK && i.GetState() != kvIndexBuilding
}

func indexGone(_ *quantadmingo.V2KvIndex, status int, _ error) bool {
	return status == http.StatusNotFound
}

// waitFor polls the index until done is true or the wait timeout passes.
func (r *kvIndexResource) waitFor(ctx context.Context, d *kvIndexModel, done func(*quantadmingo.V2KvIndex, int, error) bool) (*quantadmingo.V2KvIndex, diag.Diagnostics) {
	var diags diag.Diagnostics
	timeout := kvIndexTimeout()
	deadline := time.Now().Add(timeout)
	for {
		idx, httpResp, err := r.show(d)
		if done(idx, kvAPIStatus(httpResp), err) {
			return idx, diags
		}
		if time.Now().After(deadline) {
			diags.AddError("KV index wait timed out", fmt.Sprintf("Index %s did not settle within %s (set QUANTCDN_KV_INDEX_TIMEOUT to wait longer).", d.Name.ValueString(), timeout))
			return nil, diags
		}
		if serr := sleepCtx(ctx, KVIndexWaitInterval); serr != nil {
			diags.AddError("KV index wait cancelled", serr.Error())
			return nil, diags
		}
	}
}

func kvIndexRequest(ctx context.Context, d *kvIndexModel) (quantadmingo.V2KvIndexRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := quantadmingo.NewV2KvIndexRequest(d.Name.ValueString(), d.Collection.ValueString())
	if isKnown(d.Fields) {
		var fields []string
		diags.Append(d.Fields.ElementsAs(ctx, &fields, false)...)
		req.SetFields(fields)
	}
	if isKnown(d.SortBy) {
		req.SetSortBy(d.SortBy.ValueString())
	}
	if isKnown(d.Unique) {
		req.SetUnique(d.Unique.ValueBool())
	}
	if isKnown(d.LiveCount) {
		req.SetCount(d.LiveCount.ValueBool())
	}
	return *req, diags
}

// setKVIndexState writes every attribute from the API answer, so nothing is
// unknown after apply.
func setKVIndexState(ctx context.Context, d *kvIndexModel, i *quantadmingo.V2KvIndex) diag.Diagnostics {
	var diags, more diag.Diagnostics
	d.Name = types.StringValue(i.GetName())
	d.IndexName = d.Name
	d.Collection = types.StringValue(i.GetCollection())
	d.Fields, diags = types.ListValueFrom(ctx, types.StringType, nonNilStrings(i.GetFields()))
	d.SortBy = optionalString(i.SortBy)
	d.Unique = types.BoolValue(i.GetUnique())
	d.LiveCount = types.BoolValue(i.GetCount())
	d.Slot = types.Int64Value(int64(i.GetSlot()))
	d.State = types.StringValue(i.GetState())
	d.CreatedAt = timeString(i.GetCreatedAt())
	d.Error = optionalString(i.Error)
	d.Conflicts, more = types.ListValueFrom(ctx, types.StringType, nonNilStrings(i.GetConflicts()))
	diags.Append(more...)
	return diags
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// requireKVPath reports a missing project or store_id. The generated schema
// marks both Optional because they are path parameters.
func requireKVPath(project, storeID types.String) diag.Diagnostics {
	var diags diag.Diagnostics
	if project.IsNull() {
		diags.AddAttributeError(path.Root("project"), "Missing project", "Set project.")
	}
	if storeID.IsNull() {
		diags.AddAttributeError(path.Root("store_id"), "Missing store_id", "Set store_id.")
	}
	return diags
}
