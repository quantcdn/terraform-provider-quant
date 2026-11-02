package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_slack_bot"
)

func TestSlackBotMapsReasoningEffort(t *testing.T) {
	data := resource_slack_bot.SlackBotModel{}
	m := map[string]interface{}{"botId": "bot-1", "name": "B", "modelId": "anthropic.claude-opus-5-5", "reasoningEffort": "high"}

	if d := mapSlackBotFromMap(context.Background(), m, "test-org", &data); d.HasError() {
		t.Fatal(d)
	}
	if data.ReasoningEffort.ValueString() != "high" {
		t.Fatalf("state reasoning_effort = %s, want high", data.ReasoningEffort)
	}
}

func TestSlackBotCreateSendsReasoningEffort(t *testing.T) {
	sent := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method != http.MethodPost {
			http.NotFound(w, req)
			return
		}
		body, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(body, &sent)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"bot":{"botId":"bot-1","name":"B","modelId":"anthropic.claude-opus-5-5","reasoningEffort":"high"}}`))
	}))
	t.Cleanup(srv.Close)
	r := &slackBotResource{client: testKVClient(srv)}
	data := resource_slack_bot.SlackBotModel{
		Name: types.StringValue("B"), SystemPrompt: types.StringValue("p"), ModelId: types.StringValue("anthropic.claude-opus-5-5"),
		Organisation: types.StringValue("test-org"), ReasoningEffort: types.StringValue("high"),
	}

	if d := callSlackBotCreateAPI(context.Background(), r, &data); d.HasError() {
		t.Fatal(d)
	}
	if sent["reasoningEffort"] != "high" {
		t.Fatalf("request reasoningEffort = %v, want high", sent["reasoningEffort"])
	}
}
