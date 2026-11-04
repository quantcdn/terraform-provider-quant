package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent_overlay"
)

// storedFullOverlay holds every field the AI API stores on an overlay.
const storedFullOverlay = `{
	"baseAgentId":"quantcode-debug","organizationId":"test-org",
	"modelId":"anthropic.claude-sonnet","temperature":0.3,"maxTokens":4096,"longContext":false,
	"reasoningEffort":"high","disabledSkills":["skill-a"],"additionalSkills":["skill-b"],
	"additionalTools":["tool-a"],"disabledTools":["tool-b"],"systemPromptAppend":"Use Australian spelling.",
	"allowedCollections":["col-1"],"guardrailPreset":"official-sensitive","filterPolicies":["pol-1","pol-2"],
	"mcpServers":[{"server":"github","tools":["search_issues"]},{"server":"jira","tools":[]},{"server":"docs"}],
	"version":7}`

// fullOverlayServer answers the overlay GET with storedFullOverlay and records
// the body of each PUT.
func fullOverlayServer(t *testing.T, sent *map[string]any) *httptest.Server {
	t.Helper()
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
		_, _ = w.Write([]byte(`{"success":true,"overlay":` + storedFullOverlay + `,"base":{"agentId":"quantcode-debug","name":"debug","modelId":"anthropic.claude-opus-5-5","allowedTools":[],"assignedSkillIds":[]}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func unknownOverlayModel() resource_ai_agent_overlay.AiAgentOverlayModel {
	return resource_ai_agent_overlay.AiAgentOverlayModel{
		AgentId: types.StringValue("quantcode-debug"), Organisation: types.StringValue("test-org"),
		ModelId: types.StringUnknown(), Temperature: types.NumberUnknown(), MaxTokens: types.Int64Unknown(),
		LongContext: types.BoolUnknown(), ReasoningEffort: types.StringUnknown(), SystemPromptAppend: types.StringUnknown(),
		GuardrailPreset: types.StringUnknown(), Version: types.Int64Unknown(),
		DisabledSkills: types.ListUnknown(types.StringType), AdditionalSkills: types.ListUnknown(types.StringType),
		AdditionalTools: types.ListUnknown(types.StringType), DisabledTools: types.ListUnknown(types.StringType),
		AllowedCollections: types.ListUnknown(types.StringType), FilterPolicies: types.ListUnknown(types.StringType),
		McpServers: types.ListUnknown(overlayMcpServersType(context.Background())),
	}
}

// The overlay PUT replaces the whole overlay. An attribute the configuration
// does not set (unknown in the plan) must be sent with its stored value, or the
// apply deletes it.
func TestAiAgentOverlayUpsertKeepsEveryStoredFieldTheConfigDoesNotSet(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentOverlayResource{client: testKVClient(fullOverlayServer(t, &sent))}
	data := unknownOverlayModel()
	data.SystemPromptAppend = types.StringValue("Changed.")

	if d := r.upsert(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}

	var stored map[string]any
	if err := json.Unmarshal([]byte(storedFullOverlay), &stored); err != nil {
		t.Fatal(err)
	}
	stored["systemPromptAppend"] = "Changed."
	for _, k := range []string{"baseAgentId", "organizationId"} {
		delete(stored, k)
	}
	for field, want := range stored {
		if !reflect.DeepEqual(sent[field], want) {
			t.Errorf("PUT %s = %#v, want %#v", field, sent[field], want)
		}
	}
}

// An empty tools list allows no tools. Sending it as absent would allow every
// tool, so it must stay [] on the wire.
func TestAiAgentOverlayUpsertKeepsAnEmptyToolsList(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentOverlayResource{client: testKVClient(fullOverlayServer(t, &sent))}
	data := unknownOverlayModel()

	if d := r.upsert(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}

	servers, _ := sent["mcpServers"].([]any)
	if len(servers) != 3 {
		t.Fatalf("PUT mcpServers = %#v, want 3 attachments", sent["mcpServers"])
	}
	jira, _ := servers[1].(map[string]any)
	if tools, ok := jira["tools"].([]any); !ok || len(tools) != 0 {
		t.Fatalf("jira tools = %#v, want []", jira["tools"])
	}
	if _, ok := servers[2].(map[string]any)["tools"]; ok {
		t.Fatalf("docs tools sent, want absent (every tool)")
	}
}

func TestAiAgentOverlayUpsertSendsConfiguredValuesOverStoredOnes(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentOverlayResource{client: testKVClient(fullOverlayServer(t, &sent))}
	ctx := context.Background()
	data := unknownOverlayModel()
	data.FilterPolicies = types.ListValueMust(types.StringType, []attr.Value{})
	data.LongContext = types.BoolValue(true)
	tools := types.ListValueMust(types.StringType, []attr.Value{})
	server, d := resource_ai_agent_overlay.NewMcpServersValue(resource_ai_agent_overlay.McpServersValue{}.AttributeTypes(ctx),
		map[string]attr.Value{"server": types.StringValue("linear"), "tools": tools})
	if d.HasError() {
		t.Fatal(d)
	}
	data.McpServers = types.ListValueMust(overlayMcpServersType(ctx), []attr.Value{server})

	if d := r.upsert(ctx, &data); d.HasError() {
		t.Fatal(d)
	}

	if !reflect.DeepEqual(sent["filterPolicies"], []any{}) {
		t.Errorf("PUT filterPolicies = %#v, want [] (cleared)", sent["filterPolicies"])
	}
	if sent["longContext"] != true {
		t.Errorf("PUT longContext = %#v, want true", sent["longContext"])
	}
	want := []any{map[string]any{"server": "linear", "tools": []any{}}}
	if !reflect.DeepEqual(sent["mcpServers"], want) {
		t.Errorf("PUT mcpServers = %#v, want %#v", sent["mcpServers"], want)
	}
	if sent["modelId"] != "anthropic.claude-sonnet" {
		t.Errorf("PUT modelId = %#v, want the stored value", sent["modelId"])
	}
}

func TestAiAgentOverlayReadSetsTheNewFields(t *testing.T) {
	sent := map[string]any{}
	r := &aiAgentOverlayResource{client: testKVClient(fullOverlayServer(t, &sent))}
	ctx := context.Background()
	data := unknownOverlayModel()

	if d := r.read(ctx, &data); d.HasError() {
		t.Fatal(d)
	}

	if data.LongContext.ValueBool() != false || data.LongContext.IsNull() {
		t.Errorf("long_context = %s, want false", data.LongContext)
	}
	var policies []string
	data.FilterPolicies.ElementsAs(ctx, &policies, false)
	if !reflect.DeepEqual(policies, []string{"pol-1", "pol-2"}) {
		t.Errorf("filter_policies = %v", policies)
	}
	var servers []resource_ai_agent_overlay.McpServersValue
	data.McpServers.ElementsAs(ctx, &servers, false)
	if len(servers) != 3 || servers[0].Server.ValueString() != "github" {
		t.Fatalf("mcp_servers = %s", data.McpServers)
	}
	if servers[1].Tools.IsNull() || len(servers[1].Tools.Elements()) != 0 {
		t.Errorf("jira tools = %s, want []", servers[1].Tools)
	}
	if !servers[2].Tools.IsNull() {
		t.Errorf("docs tools = %s, want null", servers[2].Tools)
	}
	if data.Overlay.IsNull() {
		t.Error("nested overlay is null")
	}
}

// With no stored overlay (404), the PUT holds only the configured values.
func TestAiAgentOverlayUpsertWithoutAStoredOverlay(t *testing.T) {
	sent := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodGet {
			http.NotFound(w, req)
			return
		}
		body, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	t.Cleanup(srv.Close)
	r := &aiAgentOverlayResource{client: testKVClient(srv)}
	data := unknownOverlayModel()
	data.SystemPromptAppend = types.StringValue("New.")

	if d := r.upsert(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	if !reflect.DeepEqual(sent, map[string]any{"systemPromptAppend": "New."}) {
		t.Fatalf("PUT body = %#v", sent)
	}
}

// A failed read must stop the PUT: sending without the stored values would delete them.
func TestAiAgentOverlayUpsertStopsWhenTheReadFails(t *testing.T) {
	puts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			puts++
		}
		http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	r := &aiAgentOverlayResource{client: testKVClient(srv)}
	data := unknownOverlayModel()

	if d := r.upsert(context.Background(), &data); !d.HasError() {
		t.Fatal("upsert succeeded, want an error")
	}
	if puts != 0 {
		t.Fatalf("PUT sent %d times after a failed read", puts)
	}
}

// A value cleared with "" or [] reads back as absent. The read keeps the
// planned empty value, so Terraform does not report an inconsistent result.
func TestAiAgentOverlayReadKeepsPlannedEmptyValues(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"overlay":{"baseAgentId":"quantcode-debug","maxTokens":100,"version":2},"base":{"agentId":"quantcode-debug"}}`))
	}))
	t.Cleanup(srv.Close)
	r := &aiAgentOverlayResource{client: testKVClient(srv)}
	data := unknownOverlayModel()
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	data.FilterPolicies = empty
	data.DisabledTools = empty
	data.ModelId = types.StringValue("")
	data.SystemPromptAppend = types.StringValue("")
	data.McpServers = types.ListValueMust(overlayMcpServersType(ctx), []attr.Value{})

	if d := r.read(ctx, &data); d.HasError() {
		t.Fatal(d)
	}

	if !data.FilterPolicies.Equal(empty) || !data.DisabledTools.Equal(empty) {
		t.Errorf("lists = %s, %s; want the planned []", data.FilterPolicies, data.DisabledTools)
	}
	if data.ModelId.IsNull() || data.SystemPromptAppend.IsNull() {
		t.Errorf("strings = %s, %s; want the planned \"\"", data.ModelId, data.SystemPromptAppend)
	}
	if data.McpServers.IsNull() || len(data.McpServers.Elements()) != 0 {
		t.Errorf("mcp_servers = %s, want the planned []", data.McpServers)
	}
	if !data.AdditionalSkills.IsNull() {
		t.Errorf("additional_skills = %s, want null (not planned)", data.AdditionalSkills)
	}
	if data.MaxTokens.ValueInt64() != 100 || data.Version.ValueInt64() != 2 {
		t.Errorf("max_tokens %s, version %s", data.MaxTokens, data.Version)
	}
}
