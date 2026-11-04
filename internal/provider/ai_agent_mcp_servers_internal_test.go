package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_agent"
)

func agentMcpType(ctx context.Context) attr.Type {
	return resource_ai_agent.McpServersType{ObjectType: types.ObjectType{AttrTypes: resource_ai_agent.McpServersValue{}.AttributeTypes(ctx)}}
}

// The read shows the response's mcpServers, so a change made in the dashboard
// is drift. The create and update responses come as a map; the same
// attachments read the same way from both.
func TestAgentMcpServersValueReadsTheResponse(t *testing.T) {
	ctx := context.Background()
	agent := &quantadmingo.GetAIAgent200ResponseAgent{}
	agentListsFromMap(map[string]interface{}{"mcpServers": []interface{}{
		map[string]interface{}{"server": "github", "tools": []interface{}{"search_issues"}},
		map[string]interface{}{"server": "jira", "tools": []interface{}{}},
		map[string]interface{}{"server": "docs"},
	}}, agent)

	list, d := agentMcpServersValue(ctx, agent.McpServers, types.ListUnknown(agentMcpType(ctx)))
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

// filter_policies reads from the response, so a dashboard change is drift.
// An absent value keeps the plan, else null.
func TestAgentFilterPoliciesValue(t *testing.T) {
	ctx := context.Background()
	agent := &quantadmingo.GetAIAgent200ResponseAgent{}
	agentListsFromMap(map[string]interface{}{"filterPolicies": []interface{}{"pii", "secrets"}}, agent)

	list := agentFilterPoliciesValue(ctx, agent.FilterPolicies, types.ListValueMust(types.StringType, []attr.Value{}))
	var got []string
	list.ElementsAs(ctx, &got, false)
	if len(got) != 2 || got[0] != "pii" || got[1] != "secrets" {
		t.Fatalf("filter_policies = %s, want [pii secrets]", list)
	}

	planned := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("pii")})
	if list := agentFilterPoliciesValue(ctx, nil, planned); !list.Equal(planned) {
		t.Errorf("filter_policies = %s, want the plan", list)
	}
	if list := agentFilterPoliciesValue(ctx, nil, types.ListUnknown(types.StringType)); !list.IsNull() {
		t.Errorf("filter_policies = %s, want null", list)
	}
}
