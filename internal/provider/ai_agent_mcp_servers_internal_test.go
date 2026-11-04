package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent"
)

func agentMcpType(ctx context.Context) attr.Type {
	return resource_ai_agent.McpServersType{ObjectType: types.ObjectType{AttrTypes: resource_ai_agent.McpServersValue{}.AttributeTypes(ctx)}}
}

// The agent response carries mcpServers as an additional property; the read
// shows it, so a change made in the dashboard is drift.
func TestAgentMcpServersValueReadsTheResponse(t *testing.T) {
	ctx := context.Background()
	props := map[string]interface{}{"mcpServers": []interface{}{
		map[string]interface{}{"server": "github", "tools": []interface{}{"search_issues"}},
		map[string]interface{}{"server": "jira", "tools": []interface{}{}},
		map[string]interface{}{"server": "docs"},
	}}

	list, d := agentMcpServersValue(ctx, props, types.ListUnknown(agentMcpType(ctx)))
	if d.HasError() {
		t.Fatal(d)
	}
	var servers []resource_ai_agent.McpServersValue
	list.ElementsAs(ctx, &servers, false)
	if len(servers) != 3 || servers[0].Server.ValueString() != "github" || len(servers[0].Tools.Elements()) != 1 {
		t.Fatalf("mcp_servers = %s", list)
	}
	if servers[1].Tools.IsNull() || len(servers[1].Tools.Elements()) != 0 {
		t.Errorf("jira tools = %s, want []", servers[1].Tools)
	}
	if !servers[2].Tools.IsNull() {
		t.Errorf("docs tools = %s, want null", servers[2].Tools)
	}
}

// The API stores no empty list on create, so an absent value keeps the plan.
func TestAgentMcpServersValueKeepsThePlanWhenTheResponseHasNone(t *testing.T) {
	ctx := context.Background()
	planned := types.ListValueMust(agentMcpType(ctx), []attr.Value{})

	list, _ := agentMcpServersValue(ctx, nil, planned)
	if !list.Equal(planned) {
		t.Fatalf("mcp_servers = %s, want the planned []", list)
	}
	list, _ = agentMcpServersValue(ctx, nil, types.ListUnknown(agentMcpType(ctx)))
	if !list.IsNull() {
		t.Fatalf("mcp_servers = %s, want null", list)
	}
}
