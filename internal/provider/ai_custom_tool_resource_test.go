package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_custom_tool"
)

// TestBuildCustomToolCreateRequestSchemasAreObjects guards the v5.11.0
// regression: inputSchema was sent as a JSON string and the API rejected it
// with 422 "The input schema must be an array." Both schema fields must reach
// the wire as JSON objects.
func TestBuildCustomToolCreateRequestSchemasAreObjects(t *testing.T) {
	data := &resource_ai_custom_tool.AiCustomToolModel{
		Name:             types.StringValue("freshdesk_search"),
		Description:      types.StringValue("Search tickets"),
		EdgeFunctionCode: types.StringValue("export default () => {}"),
		InputSchema:      types.StringValue(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		OutputSchema:     types.StringValue(`{"type":"object","properties":{"tickets":{"type":"array"}}}`),
	}

	sdkReq, diags := buildCustomToolCreateRequest(data)
	if diags.HasError() {
		t.Fatalf("buildCustomToolCreateRequest: %v", diags)
	}

	body, err := json.Marshal(sdkReq)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var wire map[string]interface{}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	for _, key := range []string{"inputSchema", "outputSchema"} {
		obj, ok := wire[key].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: want JSON object on the wire, got %T (%v)", key, wire[key], wire[key])
		}
		if obj["type"] != "object" {
			t.Errorf("%s: want type=object, got %v", key, obj["type"])
		}
	}

	if got := wire["edgeFunctionCode"]; got != "export default () => {}" {
		t.Errorf("edgeFunctionCode: got %v", got)
	}
}

// TestBuildCustomToolCreateRequestDefaultInputSchema covers the unset case:
// the default schema must also be an object, not a string.
func TestBuildCustomToolCreateRequestDefaultInputSchema(t *testing.T) {
	data := &resource_ai_custom_tool.AiCustomToolModel{
		Name:             types.StringValue("noop"),
		Description:      types.StringValue("No inputs"),
		EdgeFunctionCode: types.StringValue("export default () => {}"),
		InputSchema:      types.StringNull(),
		OutputSchema:     types.StringNull(),
	}

	sdkReq, diags := buildCustomToolCreateRequest(data)
	if diags.HasError() {
		t.Fatalf("buildCustomToolCreateRequest: %v", diags)
	}

	body, err := json.Marshal(sdkReq)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var wire map[string]interface{}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if _, ok := wire["inputSchema"].(map[string]interface{}); !ok {
		t.Fatalf("inputSchema: want JSON object, got %T (%v)", wire["inputSchema"], wire["inputSchema"])
	}
}

// TestBuildCustomToolCreateRequestInvalidInputSchema keeps the parse error a
// diagnostic rather than a 422 from the API.
func TestBuildCustomToolCreateRequestInvalidInputSchema(t *testing.T) {
	data := &resource_ai_custom_tool.AiCustomToolModel{
		Name:             types.StringValue("broken"),
		Description:      types.StringValue("Bad schema"),
		EdgeFunctionCode: types.StringValue("export default () => {}"),
		InputSchema:      types.StringValue(`{"type":`),
	}

	if _, diags := buildCustomToolCreateRequest(data); !diags.HasError() {
		t.Fatal("want an error diagnostic for malformed input_schema")
	}
}
