package provider_test

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/provider"
)

// kvLedgerFake keeps one ledger "aiUse". The declare answers 202 PENDING; the
// show answers PENDING for `pending` reads, then `final`.
type kvLedgerFake struct {
	*kvFake
	mu        sync.Mutex
	ledger    map[string]any
	pending   int
	final     string
	dropCode  int
	dropBody  map[string]any
	showReads int
}

func shortKVLedgerWait(t *testing.T) {
	t.Helper()
	pi, pt := provider.KVLedgerWaitInterval, provider.KVLedgerWaitTimeout
	provider.KVLedgerWaitInterval, provider.KVLedgerWaitTimeout = 5*time.Millisecond, 2*time.Second
	t.Cleanup(func() { provider.KVLedgerWaitInterval, provider.KVLedgerWaitTimeout = pi, pt })
}

const ledgerPath = "/store-123/ledgers/aiUse"

func fakeKVLedger(t *testing.T) *kvLedgerFake {
	shortKVLedgerWait(t)
	f := &kvLedgerFake{kvFake: newKVFake(t), pending: 2, final: "ACTIVE", dropCode: 200}
	f.on("POST /store-123/ledgers", f.declare)
	f.on("GET "+ledgerPath, f.show)
	f.on("PATCH "+ledgerPath, f.raise)
	f.on("DELETE "+ledgerPath, f.drop)
	return f
}

func (f *kvLedgerFake) declare(body map[string]any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ledger = map[string]any{"state": "PENDING", "createdAt": "2026-10-16T00:00:00.000Z", "activeAfter": "2026-10-16T00:01:15.000Z",
		"definitionHash": "c4a5", "retainUntil": "2033-10-14T00:00:00.000Z", "personalFields": []any{}, "anchor": map[string]any{"type": "none"}}
	for k, v := range body {
		f.ledger[k] = v
	}
	f.showReads = 0
	return 202, f.ledger
}

func (f *kvLedgerFake) show(map[string]any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ledger == nil {
		return 404, map[string]any{"error": true, "code": "LEDGER_NOT_FOUND", "message": "No ledger", "retryable": false}
	}
	f.showReads++
	if f.ledger["state"] == "PENDING" && f.showReads > f.pending {
		f.ledger["state"] = f.final
	}
	return 200, f.ledger
}

func (f *kvLedgerFake) raise(body map[string]any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, v := range body {
		f.ledger[k] = v
	}
	f.ledger["updatedAt"] = "2026-10-17T00:00:00.000Z"
	return 200, f.ledger
}

func (f *kvLedgerFake) drop(map[string]any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.dropCode != 200 {
		return f.dropCode, f.dropBody
	}
	dropped := map[string]any{}
	for k, v := range f.ledger {
		dropped[k] = v
	}
	dropped["state"] = "DROPPED"
	f.ledger = nil
	return 200, dropped
}

func (f *kvLedgerFake) set(fn func(f *kvLedgerFake)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

type ledgerArgs struct {
	collection, chains, retention, lock, personal string
}

func defaultLedger() ledgerArgs {
	return ledgerArgs{collection: "ai-use:*", chains: "4", retention: "2555", lock: "governance", personal: `"requesterName"`}
}

func ledgerConfig(f *kvLedgerFake, a ledgerArgs) string {
	return f.providerBlock() + `
resource "quant_kv_ledger" "test" {
  project         = "test-project"
  store_id        = "store-123"
  name            = "aiUse"
  collection      = "` + a.collection + `"
  chains          = ` + a.chains + `
  retention_days  = ` + a.retention + `
  lock_mode       = "` + a.lock + `"
  personal_fields = [` + a.personal + `]
  subject_field   = "requesterId"
  anchor          = { type = "none" }
}
`
}

func with(a ledgerArgs, fn func(*ledgerArgs)) ledgerArgs {
	fn(&a)
	return a
}

func TestKVLedger_DeclareBodyIsCamelCaseAndWaitsForActive(t *testing.T) {
	f := fakeKVLedger(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger()), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("quant_kv_ledger.test", "state", "ACTIVE"),
				resource.TestCheckResourceAttr("quant_kv_ledger.test", "retain_until", "2033-10-14T00:00:00Z"),
				resource.TestCheckResourceAttr("quant_kv_ledger.test", "anchor.type", "none"))},
			{Config: ledgerConfig(f, defaultLedger()), PlanOnly: true},
			{ResourceName: "quant_kv_ledger.test", ImportState: true, ImportStateId: "test-project/store-123/aiUse",
				ImportStateVerify: true, ImportStateVerifyIdentifierAttribute: "name"},
		},
	})
	body := f.bodies("POST /store-123/ledgers")[0]
	want := map[string]any{"name": "aiUse", "collection": "ai-use:*", "chains": float64(4), "retentionDays": float64(2555),
		"lockMode": "governance", "subjectField": "requesterId"}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("declare body %s = %v, want %v (body %v)", k, body[k], v, body)
		}
	}
	if pf, _ := body["personalFields"].([]any); len(pf) != 1 || pf[0] != "requesterName" {
		t.Errorf("personalFields = %v", body["personalFields"])
	}
	if anchor, _ := body["anchor"].(map[string]any); len(anchor) != 1 || anchor["type"] != "none" {
		t.Errorf("anchor = %v, want only type none (no empty prefix)", body["anchor"])
	}
	for k := range body {
		if strings.Contains(k, "_") {
			t.Errorf("declare body has a snake_case key %q", k)
		}
	}
	if f.showReads < 3 {
		t.Errorf("create did not wait for ACTIVE: %d reads", f.showReads)
	}
}

func TestKVLedger_RaiseSendsOnlyChangedFieldsAndLoweringFailsAtPlan(t *testing.T) {
	f := fakeKVLedger(t)
	raised := with(defaultLedger(), func(a *ledgerArgs) { a.retention, a.lock = "3650", "compliance" })

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger())},
			{Config: ledgerConfig(f, raised), ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("quant_kv_ledger.test", plancheck.ResourceActionUpdate)}},
				Check: resource.TestCheckResourceAttr("quant_kv_ledger.test", "retention_days", "3650")},
			{Config: ledgerConfig(f, raised), PlanOnly: true},
			{Config: ledgerConfig(f, with(raised, func(a *ledgerArgs) { a.retention = "365" })), PlanOnly: true,
				ExpectError: regexp.MustCompile(`retention_days can only go up`)},
			{Config: ledgerConfig(f, with(raised, func(a *ledgerArgs) { a.chains = "2" })), PlanOnly: true,
				ExpectError: regexp.MustCompile(`chains can only go up`)},
			{Config: ledgerConfig(f, with(raised, func(a *ledgerArgs) { a.lock = "governance" })), PlanOnly: true,
				ExpectError: regexp.MustCompile(`lock_mode cannot go from compliance to governance`)},
		},
	})
	patches := f.bodies("PATCH " + ledgerPath)
	if len(patches) != 1 || len(patches[0]) != 2 || patches[0]["retentionDays"] != float64(3650) || patches[0]["lockMode"] != "compliance" {
		t.Fatalf("want one PATCH with only retentionDays and lockMode, got %v", patches)
	}
}

func TestKVLedger_DefinitionChangesFailAtPlanNotReplace(t *testing.T) {
	f := fakeKVLedger(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger())},
			{Config: ledgerConfig(f, with(defaultLedger(), func(a *ledgerArgs) { a.collection = "other:*" })), PlanOnly: true,
				ExpectError: regexp.MustCompile(`KV ledger collection cannot change`)},
			{Config: ledgerConfig(f, with(defaultLedger(), func(a *ledgerArgs) { a.personal = `"requesterName", "email"` })), PlanOnly: true,
				ExpectError: regexp.MustCompile(`KV ledger personal_fields cannot change`)},
			{Config: strings.Replace(ledgerConfig(f, defaultLedger()), `{ type = "none" }`, `{ type = "rfc3161", tsa_url = "http://timestamp.digicert.com" }`, 1), PlanOnly: true,
				ExpectError: regexp.MustCompile(`KV ledger anchor cannot change`)},
		},
	})
	if n := f.count("PATCH " + ledgerPath); n != 0 {
		t.Fatalf("a refused change reached the API: %d PATCH", n)
	}
}

func TestKVLedger_RetainedDropExplainsHowToForget(t *testing.T) {
	f := fakeKVLedger(t)
	f.dropCode = 409
	f.dropBody = map[string]any{"error": true, "code": "LEDGER_RETAINED", "message": "Retention has not ended", "retryable": false,
		"reason": "retention", "retainUntil": "2033-10-14T00:00:00.000Z"}
	providerOnly := f.providerBlock()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger())},
			{Config: providerOnly, ExpectError: regexp.MustCompile(`(?s)retained until 2033-10-14T00:00:00.000Z.*removed \{.*destroy = false.*retainOnDelete`)},
			// Retention ends: the drop succeeds, so the final destroy is clean.
			{PreConfig: func() { f.set(func(f *kvLedgerFake) { f.dropCode = 200 }) }, Config: providerOnly},
		},
	})
}

func TestKVLedger_FailedLedgerIsDeclaredAgain(t *testing.T) {
	f := fakeKVLedger(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger())},
			{PreConfig: func() {
				f.set(func(f *kvLedgerFake) { f.ledger["state"] = "FAILED"; f.ledger["error"] = "LEDGER_COLLECTION_NOT_EMPTY" })
			},
				Config: ledgerConfig(f, defaultLedger()), ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("quant_kv_ledger.test", plancheck.ResourceActionDestroyBeforeCreate)}},
				Check: resource.TestCheckResourceAttr("quant_kv_ledger.test", "state", "ACTIVE")},
		},
	})
}

func TestKVLedger_FailedDeclareIsAnError(t *testing.T) {
	f := fakeKVLedger(t)
	f.final = "FAILED"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    []resource.TestStep{{Config: ledgerConfig(f, defaultLedger()), ExpectError: regexp.MustCompile(`KV ledger declare failed`)}},
	})
}

func TestKVLedger_DroppedLedgerInConfigIsAPlanError(t *testing.T) {
	f := fakeKVLedger(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger())},
			{PreConfig: func() { f.set(func(f *kvLedgerFake) { f.ledger["state"] = "DROPPED" }) },
				Config: ledgerConfig(f, defaultLedger()), PlanOnly: true, ExpectError: regexp.MustCompile(`KV ledger is dropped`)},
		},
	})
}

func TestKVLedger_MinimalConfigHasNoDrift(t *testing.T) {
	f := fakeKVLedger(t)
	cfg := f.providerBlock() + `
resource "quant_kv_ledger" "test" {
  project        = "test-project"
  store_id       = "store-123"
  name           = "aiUse"
  collection     = "ai-use:*"
  chains         = 1
  retention_days = 30
  lock_mode      = "governance"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("quant_kv_ledger.test", "anchor.type", "none"),
				resource.TestCheckResourceAttr("quant_kv_ledger.test", "personal_fields.#", "0"),
				resource.TestCheckNoResourceAttr("quant_kv_ledger.test", "subject_field"))},
			{Config: cfg, PlanOnly: true},
		},
	})
	body := f.bodies("POST /store-123/ledgers")[0]
	for _, k := range []string{"anchor", "personalFields", "subjectField", "writerPolicy"} {
		if _, ok := body[k]; ok {
			t.Errorf("declare body sent %s for an omitted attribute: %v", k, body)
		}
	}
}

// The KV API allows 10 ledger changes per minute per project. A 429 on the
// declare, the raise and the drop waits the full Retry-After and sends again.
func TestKVLedger_RateLimitedDeclareRaiseAndDropRetry(t *testing.T) {
	f := fakeKVLedger(t)
	provider.KVLedgerWaitTimeout = 10 * time.Second
	for _, route := range []string{"POST /store-123/ledgers", "PATCH " + ledgerPath, "DELETE " + ledgerPath} {
		f.on(route, limitedTwice(f.routes[route]))
	}
	raised := with(defaultLedger(), func(a *ledgerArgs) { a.retention = "3650" })

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: ledgerConfig(f, defaultLedger())},
			{Config: ledgerConfig(f, raised), Check: resource.TestCheckResourceAttr("quant_kv_ledger.test", "retention_days", "3650")},
		},
	})
	for _, route := range []string{"POST /store-123/ledgers", "PATCH " + ledgerPath, "DELETE " + ledgerPath} {
		if n := f.count(route); n != 3 {
			t.Errorf("%s sent %d times, want 3 (two 429)", route, n)
		}
	}
}

// A declare that is still PENDING at the wait timeout is an error that says
// the ledger exists and how to import it. The resource is not saved, so it
// is not tainted and no later apply drops it.
func TestKVLedger_PendingAtTimeoutSaysImportAndIsNotSaved(t *testing.T) {
	f := fakeKVLedger(t)
	f.pending = 1 << 30
	provider.KVLedgerWaitTimeout = 50 * time.Millisecond

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{Config: ledgerConfig(f, defaultLedger()),
			ExpectError: regexp.MustCompile(`the\s+ledger\s+exists\s+on\s+the\s+server;\s+import\s+it\s+with\s+terraform\s+import\s+quant_kv_ledger.<name>\s+test-project/store-123/aiUse`)}},
	})
	if n := f.count("DELETE " + ledgerPath); n != 0 {
		t.Fatalf("a PENDING ledger was dropped: %d DELETE", n)
	}
}
