package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_store"
)

// testKVClient is a client for srv with no client-level retries.
func testKVClient(srv *httptest.Server) *client.Client {
	return client.NewWithOptions("test-token", "test-org", &client.ClientOptions{BaseURL: srv.URL,
		RateLimitConfig: &client.RateLimitConfig{RequestsPerSecond: 1000, MaxRetries: 0, RetryCondition: client.DefaultRetryCondition}})
}

// upgradeServer answers the upgrade state of store-123 with format.
func upgradeServer(t *testing.T, format int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v2/organizations/test-org/projects/test-project/kv/store-123/upgrade" {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"format":` + string(rune('0'+format)) + `,"state":"active","table":"t","upgrade":null}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func kvStoreUpdate(t *testing.T, r *kvStoreResource, plan, state resource_kv_store.KvStoreModel) resource_kv_store.KvStoreModel {
	t.Helper()
	ctx := context.Background()
	s := resource_kv_store.KvStoreResourceSchema(ctx)
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: s}, State: tfsdk.State{Schema: s}}
	if d := req.Plan.Set(ctx, &plan); d.HasError() {
		t.Fatal(d)
	}
	if d := req.State.Set(ctx, &state); d.HasError() {
		t.Fatal(d)
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got resource_kv_store.KvStoreModel
	if d := resp.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	return got
}

func priorStore() resource_kv_store.KvStoreModel {
	return resource_kv_store.KvStoreModel{Id: types.StringValue("store-123"), StoreId: types.StringValue("store-123"),
		Name: types.StringValue("my-store"), Project: types.StringValue("test-project"), Organization: types.StringValue("test-org"),
		Format: types.Int64Value(1)}
}

func assertStoreIdentity(t *testing.T, got resource_kv_store.KvStoreModel) {
	t.Helper()
	if got.Id.ValueString() != "store-123" || got.StoreId.ValueString() != "store-123" ||
		got.Project.ValueString() != "test-project" || got.Organization.ValueString() != "test-org" {
		t.Fatalf("identity not copied from state: %+v", got)
	}
}

// An update whose config did not change, where the framework planned the
// computed attributes as unknown, keeps id and store_id known.
func TestKVStoreUpdate_UnchangedConfigKeepsIDKnown(t *testing.T) {
	r := &kvStoreResource{client: testKVClient(upgradeServer(t, 1))}
	plan := priorStore()
	plan.Id, plan.StoreId, plan.Organization = types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	got := kvStoreUpdate(t, r, plan, priorStore())
	assertStoreIdentity(t, got)
	if got.Format.ValueInt64() != 1 {
		t.Fatalf("format = %v", got.Format)
	}
}

// A state from a provider version before format has no format. An apply
// with -refresh=false reaches Update with that state and no Read before it.
func TestKVStoreUpdate_LegacyStateWithoutRefresh(t *testing.T) {
	r := &kvStoreResource{client: testKVClient(upgradeServer(t, 2))}
	legacy := priorStore()
	legacy.Format = types.Int64Null()
	plan := priorStore()
	plan.Format = types.Int64Value(2)
	plan.Id, plan.StoreId = types.StringUnknown(), types.StringValue("store-999")
	got := kvStoreUpdate(t, r, plan, legacy)
	assertStoreIdentity(t, got)
	if got.Format.ValueInt64() != 2 {
		t.Fatalf("format = %v, want the real format 2", got.Format)
	}
}
