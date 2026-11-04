package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

// mcpAttachment converts one mcp_servers element to the API shape. An unset or
// unknown tools list means every tool the server allows (nil, so the key is
// left out). A configured empty list allows none, so it stays a non-nil empty
// slice: the SDK sends [] for it.
func mcpAttachment(ctx context.Context, server basetypes.StringValue, tools basetypes.ListValue) (quantadmingo.CreateAIAgentRequestMcpServersInner, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := quantadmingo.CreateAIAgentRequestMcpServersInner{Server: server.ValueString()}
	if tools.IsNull() || tools.IsUnknown() {
		return out, diags
	}
	vals := []string{}
	diags.Append(tools.ElementsAs(ctx, &vals, false)...)
	if vals == nil {
		vals = []string{}
	}
	out.Tools = vals
	return out, diags
}

// mcpAttachmentsFromStored converts the overlay read's attachments to the
// request shape, so a PUT can send back what is stored.
func mcpAttachmentsFromStored(stored []quantadmingo.GetAgentOverlay200ResponseOverlayMcpServersInner) []quantadmingo.CreateAIAgentRequestMcpServersInner {
	if stored == nil {
		return nil
	}
	out := make([]quantadmingo.CreateAIAgentRequestMcpServersInner, 0, len(stored))
	for _, s := range stored {
		if s.Server == nil || *s.Server == "" {
			continue
		}
		out = append(out, quantadmingo.CreateAIAgentRequestMcpServersInner{Server: *s.Server, Tools: s.Tools})
	}
	return out
}

// mcpServerAttrs is the attribute map of one mcp_servers element read from
// the API. A missing tools list reads as null (every tool).
func mcpServerAttrs(ctx context.Context, s quantadmingo.GetAgentOverlay200ResponseOverlayMcpServersInner) map[string]attr.Value {
	tools := types.ListNull(types.StringType)
	if s.Tools != nil {
		tools = stringListFromSlice(ctx, s.Tools)
	}
	return map[string]attr.Value{
		"server": stringValueOrEmpty(s.Server),
		"tools":  tools,
	}
}
