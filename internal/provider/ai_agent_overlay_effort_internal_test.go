package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent_overlay"
)

const testOverlayPath = "/api/v3/organizations/test-org/ai/agents/quantcode-debug/overlay"

// overlayServer answers the overlay upsert and read; the stored overlay has
// reasoningEffort "low". The upsert request body is recorded.
func overlayServer(t *testing.T, sent *map[string]any) *httptest.Server {
	t.Helper()
	overlay := `{"baseAgentId":"quantcode-debug","organizationId":"test-org","reasoningEffort":"low","version":1}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path != testOverlayPath {
			http.NotFound(w, req)
			return
		}
		if req.Method == http.MethodPut {
			body, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(body, sent)
		}
		_, _ = w.Write([]byte(`{"success":true,"overlay":` + overlay + `,"base":{"agentId":"quantcode-debug","name":"debug","modelId":"anthropic.claude-opus-5-5","allowedTools":[],"assignedSkillIds":[]}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAiAgentOverlayUpsertSendsAndStoresReasoningEffort(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentOverlayResource{client: testKVClient(overlayServer(t, &sent))}
	data := resource_ai_agent_overlay.AiAgentOverlayModel{
		AgentId: types.StringValue("quantcode-debug"), Organisation: types.StringValue("test-org"),
		ReasoningEffort: types.StringValue("low"),
	}

	if d := r.upsert(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	if sent["reasoningEffort"] != "low" {
		t.Fatalf("request reasoningEffort = %v, want low", sent["reasoningEffort"])
	}
	if d := r.read(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	if data.ReasoningEffort.ValueString() != "low" {
		t.Fatalf("state reasoning_effort = %s, want low", data.ReasoningEffort)
	}
}
