package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jarcoal/httpmock"
)

const autoModeRes = "quant_ai_auto_mode.test"

// autoModeFake is the auto-mode API of one organisation. cfg is nil when the
// organisation has no setting. A PUT with a revision other than the stored
// one (or any revision when there is no setting) returns 409, like the API.
type autoModeFake struct {
	mu        sync.Mutex
	cfg       map[string]any
	puts      []map[string]any
	deletes   int
	putStatus int
	putBody   any
}

func mockAiAutoModeServer(t *testing.T, org string) *autoModeFake {
	t.Helper()
	f := &autoModeFake{}
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
	url := fmt.Sprintf("https://dashboard.quantcdn.io/api/v3/organizations/%s/ai/auto-mode", org)
	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})
	httpmock.RegisterResponder("GET", url, func(*http.Request) (*http.Response, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		return httpmock.NewJsonResponse(200, map[string]any{"autoMode": f.cfg, "eligible": nil})
	})
	httpmock.RegisterResponder("PUT", url, f.put)
	httpmock.RegisterResponder("DELETE", url, func(*http.Request) (*http.Response, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.deletes++
		f.cfg = nil
		return httpmock.NewJsonResponse(200, map[string]any{"autoMode": nil})
	})
	return f
}

func (f *autoModeFake) put(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := io.ReadAll(req.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	f.puts = append(f.puts, body)
	if f.putStatus != 0 {
		return httpmock.NewJsonResponse(f.putStatus, f.putBody)
	}
	group := body["group"].(map[string]any)
	sent, hasRev := group["revision"]
	current := float64(0)
	if f.cfg != nil {
		current = f.cfg["group"].(map[string]any)["revision"].(float64)
	}
	if (f.cfg == nil && hasRev) || (f.cfg != nil && sent != current) {
		return httpmock.NewJsonResponse(409, map[string]any{"error": "The setting changed since it was read"})
	}
	cfg := map[string]any{
		"enabled":                body["enabled"],
		"orchestratorModel":      "",
		"group":                  map[string]any{"revision": current + 1, "roles": group["roles"]},
		"watcherIntervalMinutes": body["watcherIntervalMinutes"],
		"enforcement":            body["enforcement"],
		"autonomousDefault":      body["autonomousDefault"],
		"updatedAt":              fmt.Sprintf("2026-11-14T00:00:0%.0fZ", current+1),
		"updatedBy":              "tf-token",
	}
	copyIfSet(cfg, body, "orchestratorModel", "maxParallelTasks")
	f.cfg = cfg
	return httpmock.NewJsonResponse(200, map[string]any{"autoMode": cfg})
}

func copyIfSet(dst, src map[string]any, keys ...string) {
	for _, k := range keys {
		if v, ok := src[k]; ok {
			dst[k] = v
		}
	}
}

func (f *autoModeFake) lastPut(t *testing.T) map[string]any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.puts) == 0 {
		t.Fatal("no PUT was sent")
	}
	return f.puts[len(f.puts)-1]
}

// edit changes the stored setting, as a Portal save does.
func (f *autoModeFake) edit(fast []any, revision float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	g := f.cfg["group"].(map[string]any)
	roles := g["roles"].(map[string]any)
	roles["fast"] = fast
	g["revision"] = revision
}

func (f *autoModeFake) clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cfg = nil
}

func autoModeConfig(org, extra string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_auto_mode" "test" {
  enabled                  = true
  orchestrator_model       = "orch-model"
  watcher_interval_minutes = 5
  enforcement              = "default"
  autonomous_default       = false
  %[2]s
  group = {
    roles = {
      fast     = ["b", "a"]
      standard = ["s1"]
      deep     = []
    }
  }
}
`, org, extra)
}

func checkPutRevision(t *testing.T, f *autoModeFake, want any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		group := f.lastPut(t)["group"].(map[string]any)
		got, ok := group["revision"]
		if want == nil && ok {
			return fmt.Errorf("PUT sent revision %v, want none", got)
		}
		if want != nil && got != want {
			return fmt.Errorf("PUT sent revision %v, want %v", got, want)
		}
		return nil
	}
}

func checkPutBody(t *testing.T, f *autoModeFake) resource.TestCheckFunc {
	return func(*terraform.State) error {
		body := f.lastPut(t)
		if _, ok := body["maxParallelTasks"]; ok {
			return fmt.Errorf("PUT sent maxParallelTasks %v, want none", body["maxParallelTasks"])
		}
		fast := body["group"].(map[string]any)["roles"].(map[string]any)["fast"]
		if fmt.Sprint(fast) != "[b a]" {
			return fmt.Errorf("PUT sent fast %v, want [b a]", fast)
		}
		if body["orchestratorModel"] != "orch-model" {
			return fmt.Errorf("PUT sent orchestratorModel %v", body["orchestratorModel"])
		}
		return nil
	}
}

func TestAiAutoMode_CreateReadUpdateImportDelete(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			if f.deletes != 1 || f.cfg != nil {
				return fmt.Errorf("destroy did not send DELETE: deletes=%d", f.deletes)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: autoModeConfig(org, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPutRevision(t, f, nil),
					checkPutBody(t, f),
					resource.TestCheckResourceAttr(autoModeRes, "organisation", org),
					resource.TestCheckResourceAttr(autoModeRes, "enabled", "true"),
					resource.TestCheckResourceAttr(autoModeRes, "orchestrator_model", "orch-model"),
					resource.TestCheckResourceAttr(autoModeRes, "watcher_interval_minutes", "5"),
					resource.TestCheckResourceAttr(autoModeRes, "enforcement", "default"),
					resource.TestCheckResourceAttr(autoModeRes, "autonomous_default", "false"),
					resource.TestCheckResourceAttr(autoModeRes, "group.revision", "1"),
					resource.TestCheckResourceAttr(autoModeRes, "group.roles.fast.#", "2"),
					resource.TestCheckResourceAttr(autoModeRes, "group.roles.fast.0", "b"),
					resource.TestCheckResourceAttr(autoModeRes, "group.roles.fast.1", "a"),
					resource.TestCheckResourceAttr(autoModeRes, "group.roles.standard.0", "s1"),
					resource.TestCheckResourceAttr(autoModeRes, "group.roles.deep.#", "0"),
					resource.TestCheckNoResourceAttr(autoModeRes, "max_parallel_tasks"),
					resource.TestCheckResourceAttr(autoModeRes, "updated_at", "2026-11-14T00:00:01Z"),
					resource.TestCheckResourceAttr(autoModeRes, "updated_by", "tf-token"),
				),
			},
			// Role order and an unset max_parallel_tasks survive a refresh: no diff.
			{Config: autoModeConfig(org, ""), PlanOnly: true},
			{
				Config: autoModeConfig(org, "max_parallel_tasks = 4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPutRevision(t, f, float64(1)),
					resource.TestCheckResourceAttr(autoModeRes, "max_parallel_tasks", "4"),
					resource.TestCheckResourceAttr(autoModeRes, "group.revision", "2"),
					resource.TestCheckResourceAttr(autoModeRes, "updated_at", "2026-11-14T00:00:02Z"),
				),
			},
			{Config: autoModeConfig(org, "max_parallel_tasks = 4"), PlanOnly: true},
			{
				ResourceName:                         autoModeRes,
				ImportState:                          true,
				ImportStateId:                        org,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "organisation",
			},
		},
	})
}

func TestAiAutoMode_PortalEditShowsDriftAndUsesReadRevision(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: autoModeConfig(org, "")},
			{
				PreConfig: func() { f.edit([]any{"x"}, 7) },
				Config:    autoModeConfig(org, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(autoModeRes, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPutRevision(t, f, float64(7)),
					resource.TestCheckResourceAttr(autoModeRes, "group.roles.fast.0", "b"),
					resource.TestCheckResourceAttr(autoModeRes, "group.revision", "8"),
				),
			},
		},
	})
}

func TestAiAutoMode_PortalDeleteRecreates(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: autoModeConfig(org, "")},
			{
				PreConfig: f.clear,
				Config:    autoModeConfig(org, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(autoModeRes, plancheck.ResourceActionCreate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkPutRevision(t, f, nil),
					resource.TestCheckResourceAttr(autoModeRes, "group.revision", "1"),
				),
			},
		},
	})
}

func TestAiAutoMode_OrganisationChangeReplaces(t *testing.T) {
	org, other := "test-org", "other-org"
	f := mockAiAutoModeServer(t, org)
	g := mockAiAutoModeServer(t, other)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Unset organisation takes the provider default.
			{Config: autoModeConfig(org, ""), Check: resource.TestCheckResourceAttr(autoModeRes, "organisation", org)},
			// Unset again, or set to the same organisation: no replace, no diff.
			{Config: autoModeConfig(org, ""), PlanOnly: true},
			{Config: autoModeConfig(org, `organisation = "test-org"`), PlanOnly: true},
			{
				Config: autoModeConfig(org, `organisation = "other-org"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(autoModeRes, plancheck.ResourceActionDestroyBeforeCreate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(autoModeRes, "organisation", other),
					resource.TestCheckResourceAttr(autoModeRes, "group.revision", "1"),
					checkPutRevision(t, g, nil),
					func(*terraform.State) error {
						if f.deletes != 1 || f.cfg != nil {
							return fmt.Errorf("the old organisation's setting was not deleted: deletes=%d", f.deletes)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAiAutoMode_400ListsIneligibleModels(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)
	f.putStatus = 400
	f.putBody = map[string]any{
		"error": "Some models are not eligible",
		"eligible": map[string]any{
			"usable":       false,
			"orchestrator": map[string]any{"modelId": "orch-model", "resolvedId": "orch-model", "eligible": true},
			"roles": map[string]any{
				"fast":     []any{map[string]any{"modelId": "b", "resolvedId": "b-v1", "eligible": false, "reason": "not in the model allowlist"}, map[string]any{"modelId": "a", "resolvedId": "a", "eligible": true}},
				"standard": []any{map[string]any{"modelId": "s1", "resolvedId": "s1", "eligible": false, "reason": "no tool support"}},
				"deep":     []any{},
			},
		},
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      autoModeConfig(org, ""),
			ExpectError: regexp.MustCompile(`(?s)Some models are not eligible.*group.roles.fast: b \(b-v1\): not in the model allowlist.*group.roles.standard: s1 \(s1\): no tool support`),
		}},
	})
}

func TestAiAutoMode_409GovernanceFirst(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)
	f.putStatus = 409
	f.putBody = map[string]any{"error": "Save the organisation governance settings first"}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      autoModeConfig(org, ""),
			ExpectError: regexp.MustCompile(`(?s)Save the organisation governance settings first.*Apply quant_ai_governance.*depends_on`),
		}},
	})
}

func TestAiAutoMode_409OnCreateSaysImport(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)
	f.cfg = map[string]any{"group": map[string]any{"revision": float64(3)}}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      autoModeConfig(org, ""),
			ExpectError: regexp.MustCompile(`(?s)already has an auto-mode setting.*terraform\s+import\s+quant_ai_auto_mode.<name>\s+test-org.*pulumi\s+import\s+quant:index:AiAutoMode\s+<name>\s+test-org`),
		}},
	})
	if _, ok := f.puts[0]["group"].(map[string]any)["revision"]; ok {
		t.Fatal("create sent a revision")
	}
}

func TestAiAutoMode_409OnUpdateSaysRefresh(t *testing.T) {
	org := "test-org"
	f := mockAiAutoModeServer(t, org)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: autoModeConfig(org, "")},
			{
				PreConfig: func() {
					f.mu.Lock()
					defer f.mu.Unlock()
					f.putStatus = 409
					f.putBody = map[string]any{"error": "The setting changed since it was read"}
				},
				Config:      autoModeConfig(org, "max_parallel_tasks = 4"),
				ExpectError: regexp.MustCompile(`(?s)The setting changed since it was read.*Refresh the state, then apply\s+again.*terraform apply.*pulumi\s+up\s+--refresh`),
			},
		},
	})
	if got := f.lastPut(t)["group"].(map[string]any)["revision"]; got != float64(1) {
		t.Fatalf("update sent revision %v, want 1", got)
	}
}

func TestAiAutoMode_EmptyOrchestratorModelRejected(t *testing.T) {
	org := "test-org"
	mockAiAutoModeServer(t, org)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      strings.Replace(autoModeConfig(org, ""), `"orch-model"`, `""`, 1),
			ExpectError: regexp.MustCompile(`(?s)orchestrator_model.*at least 1`),
		}},
	})
}

func TestAiAutoMode_ImportWithoutSettingFails(t *testing.T) {
	org := "test-org"
	mockAiAutoModeServer(t, org)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:        autoModeConfig(org, ""),
			ResourceName:  autoModeRes,
			ImportState:   true,
			ImportStateId: org,
			ExpectError:   regexp.MustCompile(`Organisation "test-org" has no auto-mode setting`),
		}},
	})
}
