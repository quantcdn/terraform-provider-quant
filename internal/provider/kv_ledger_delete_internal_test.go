package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_ledger"
)

// ledgerServer answers the show of ledger aiUse with serverState and counts
// the drops.
func ledgerServer(t *testing.T, serverState string) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var drops atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodDelete {
			drops.Add(1)
		}
		_, _ = w.Write([]byte(`{"name":"aiUse","collection":"ai-use:*","state":"` + serverState + `","chains":1,"retentionDays":30,"lockMode":"governance",` +
			`"personalFields":[],"anchor":{"type":"none"},"createdAt":"2026-10-16T00:00:00.000Z","activeAfter":"2026-10-16T00:01:15.000Z","definitionHash":"c4a5"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &drops
}

func kvLedgerDelete(t *testing.T, r *kvLedgerResource, priorState string) resource.DeleteResponse {
	t.Helper()
	ctx := context.Background()
	s := resource_kv_ledger.KvLedgerResourceSchema(ctx)
	m := resource_kv_ledger.KvLedgerModel{Organization: types.StringValue("test-org"), Project: types.StringValue("test-project"),
		StoreId: types.StringValue("store-123"), Name: types.StringValue("aiUse"), State: types.StringValue(priorState),
		PersonalFields: types.ListNull(types.StringType),
		Anchor:         resource_kv_ledger.NewAnchorValueNull(), WriterPolicy: resource_kv_ledger.NewWriterPolicyValueNull()}
	req := resource.DeleteRequest{State: tfsdk.State{Schema: s}}
	if d := req.State.Set(ctx, &m); d.HasError() {
		t.Fatal(d)
	}
	resp := resource.DeleteResponse{State: req.State}
	r.Delete(ctx, req, &resp)
	return resp
}

// The plan replaces a FAILED ledger. When the server shows it ACTIVE at the
// apply, the drop would destroy a working ledger: Delete refuses.
func TestKVLedgerDelete_RefusesWhenAFailedLedgerIsActiveAgain(t *testing.T) {
	srv, drops := ledgerServer(t, kvLedgerActive)
	resp := kvLedgerDelete(t, &kvLedgerResource{client: testKVClient(srv)}, kvLedgerFailed)
	if !resp.Diagnostics.HasError() || drops.Load() != 0 {
		t.Fatalf("want a refusal and no drop, got %v and %d drops", resp.Diagnostics, drops.Load())
	}
	if d := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(d, "ACTIVE") || !strings.Contains(d, "FAILED") {
		t.Fatalf("detail: %s", d)
	}
}

func TestKVLedgerDelete_DropsAFailedLedgerThatIsStillFailed(t *testing.T) {
	srv, drops := ledgerServer(t, kvLedgerFailed)
	resp := kvLedgerDelete(t, &kvLedgerResource{client: testKVClient(srv)}, kvLedgerFailed)
	if resp.Diagnostics.HasError() || drops.Load() != 1 {
		t.Fatalf("want one drop, got %v and %d drops", resp.Diagnostics, drops.Load())
	}
}
