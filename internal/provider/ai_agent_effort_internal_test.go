package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent"
)

const testAgentsPath = "/api/v3/organizations/test-org/ai/agents"

// agentServer answers the agent create and read with an agent that carries
// reasoningEffort "high", and records the create request body.
func agentServer(t *testing.T, sent *map[string]any) *httptest.Server {
	t.Helper()
	agent := `{"agentId":"agent-1","name":"A","description":"d","systemPrompt":"p","modelId":"anthropic.claude-opus-5-5","reasoningEffort":"high"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == testAgentsPath:
			body, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(body, sent)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"success":true,"agent":` + agent + `}`))
		case req.Method == http.MethodGet && req.URL.Path == testAgentsPath+"/agent-1":
			_, _ = w.Write([]byte(`{"agent":` + agent + `}`))
		default:
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func agentPlan(effort types.String) resource_ai_agent.AiAgentModel {
	return resource_ai_agent.AiAgentModel{
		Name: types.StringValue("A"), Description: types.StringValue("d"), SystemPrompt: types.StringValue("p"),
		ModelId: types.StringValue("anthropic.claude-opus-5-5"), Organisation: types.StringValue("test-org"),
		ReasoningEffort: effort,
	}
}

func TestAiAgentCreateSendsAndStoresReasoningEffort(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentResource{client: testKVClient(agentServer(t, &sent))}
	data := agentPlan(types.StringValue("high"))

	if d := callAiAgentCreateAPI(context.Background(), r, &data); d.HasError() {
		t.Fatal(d)
	}
	if sent["reasoningEffort"] != "high" {
		t.Fatalf("request reasoningEffort = %v, want high", sent["reasoningEffort"])
	}
	if data.ReasoningEffort.ValueString() != "high" {
		t.Fatalf("state reasoning_effort = %s, want high", data.ReasoningEffort)
	}
}

func TestAiAgentCreateWithoutEffortSendsNone(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentResource{client: testKVClient(agentServer(t, &sent))}
	data := agentPlan(types.StringNull())

	if d := callAiAgentCreateAPI(context.Background(), r, &data); d.HasError() {
		t.Fatal(d)
	}
	if _, ok := sent["reasoningEffort"]; ok {
		t.Fatalf("request carried reasoningEffort = %v without a configured value", sent["reasoningEffort"])
	}
}

func TestAiAgentReadMapsReasoningEffort(t *testing.T) {
	r := &aiAgentResource{client: testKVClient(agentServer(t, &map[string]any{}))}
	data := agentPlan(types.StringNull())
	data.AgentId = types.StringValue("agent-1")

	if d := callAiAgentReadAPI(context.Background(), r, &data); d.HasError() {
		t.Fatal(d)
	}
	if data.ReasoningEffort.ValueString() != "high" {
		t.Fatalf("state reasoning_effort = %s, want high", data.ReasoningEffort)
	}
}
