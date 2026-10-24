package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_store"
)

const (
	testKVCreatePath = "/api/v2/organizations/test-org/projects/test-project/kv"
	testKVItemsPath  = "/api/v2/organizations/test-org/projects/test-project/kv/store-123/items"
)

// createServer answers the store create with code and body, and counts the
// item list calls that the readiness wait makes.
func createServer(t *testing.T, code int, body string, itemCalls *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == testKVCreatePath:
			w.WriteHeader(code)
			_, _ = w.Write([]byte(body))
		case req.Method == http.MethodGet && req.URL.Path == testKVItemsPath:
			atomic.AddInt32(itemCalls, 1)
			_, _ = w.Write([]byte(`{"data":[],"next_cursor":null}`))
		default:
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func kvStoreCreate(t *testing.T, srv *httptest.Server) resource_kv_store.KvStoreModel {
	t.Helper()
	prevInterval, prevTimeout := KVStoreReadyInterval, KVStoreReadyTimeout
	KVStoreReadyInterval, KVStoreReadyTimeout = time.Millisecond, 5*time.Second
	t.Cleanup(func() { KVStoreReadyInterval, KVStoreReadyTimeout = prevInterval, prevTimeout })

	ctx := context.Background()
	s := resource_kv_store.KvStoreResourceSchema(ctx)
	plan := resource_kv_store.KvStoreModel{Id: types.StringUnknown(), StoreId: types.StringUnknown(), Status: types.StringUnknown(),
		Name: types.StringValue("my-store"), Project: types.StringValue("test-project"), Organization: types.StringValue("test-org"),
		Format: types.Int64Value(1)}
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: s}}
	if d := req.Plan.Set(ctx, &plan); d.HasError() {
		t.Fatal(d)
	}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r := &kvStoreResource{client: testKVClient(srv)}
	r.Create(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got resource_kv_store.KvStoreModel
	if d := resp.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	return got
}

func assertCreatedStore(t *testing.T, got resource_kv_store.KvStoreModel, itemCalls int32, wantCalls bool) {
	t.Helper()
	assertStoreIdentity(t, got)
	if got.Status.ValueString() != kvStoreStatusReady {
		t.Fatalf("status = %s, want ready", got.Status)
	}
	if wantCalls && itemCalls == 0 {
		t.Fatal("expected Create to wait for the store, got no item list calls")
	}
	if !wantCalls && itemCalls != 0 {
		t.Fatalf("expected no wait, got %d item list calls", itemCalls)
	}
}

// A create that answers 200 with status ready does not wait.
func TestKVStoreCreate_ReadyDoesNotWait(t *testing.T) {
	var calls int32
	srv := createServer(t, http.StatusOK, `{"id":"store-123","name":"my-store","status":"ready"}`, &calls)
	got := kvStoreCreate(t, srv)
	assertCreatedStore(t, got, atomic.LoadInt32(&calls), false)
}

// A create that answers 202 with status provisioning waits for the store.
func TestKVStoreCreate_ProvisioningWaits(t *testing.T) {
	var calls int32
	srv := createServer(t, http.StatusAccepted, `{"id":"store-123","name":"my-store","status":"provisioning"}`, &calls)
	got := kvStoreCreate(t, srv)
	assertCreatedStore(t, got, atomic.LoadInt32(&calls), true)
}

// A create that answers 200 with no status (an older API) waits for the store.
func TestKVStoreCreate_NoStatusWaits(t *testing.T) {
	var calls int32
	srv := createServer(t, http.StatusOK, `{"id":"store-123","name":"my-store"}`, &calls)
	got := kvStoreCreate(t, srv)
	assertCreatedStore(t, got, atomic.LoadInt32(&calls), true)
}
