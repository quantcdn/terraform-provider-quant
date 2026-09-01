package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_custom_tool"
)

var (
	_ resource.Resource                = (*aiCustomToolResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiCustomToolResource)(nil)
	_ resource.ResourceWithImportState = (*aiCustomToolResource)(nil)
)

func NewAiCustomToolResource() resource.Resource {
	return &aiCustomToolResource{}
}

type aiCustomToolResource struct {
	client *client.Client
}

func (r *aiCustomToolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_custom_tool"
}

func (r *aiCustomToolResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_ai_custom_tool.AiCustomToolResourceSchema(ctx)
	addUseStateForUnknown(s.Attributes)
	// These change on every create/update — must not carry state forward.
	clearPlanModifiers(s.Attributes, "success", "is_update", "message")
	resp.Schema = s
}

func (r *aiCustomToolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *aiCustomToolResource) getOrg(data *resource_ai_custom_tool.AiCustomToolModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

func (r *aiCustomToolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_custom_tool.AiCustomToolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCustomToolCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiCustomToolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_custom_tool.AiCustomToolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCustomToolReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the tool was not found, remove from state so TF plans recreation.
	if data.Name.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiCustomToolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ai_custom_tool.AiCustomToolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve ToolName from state (it's the canonical identifier).
	var state resource_ai_custom_tool.AiCustomToolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.ToolName.IsNull() && !state.ToolName.IsUnknown() {
		data.ToolName = state.ToolName
	}

	// Update uses the same POST (idempotent create-or-update).
	resp.Diagnostics.Append(callCustomToolCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiCustomToolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_custom_tool.AiCustomToolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCustomToolDeleteAPI(ctx, r, &data)...)
}

// ImportState imports a custom tool by its toolName.
func (r *aiCustomToolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_ai_custom_tool.AiCustomToolModel
	data.Name = types.StringValue(req.ID)
	data.ToolName = types.StringValue(req.ID)

	resp.Diagnostics.Append(callCustomToolReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

// getToolName returns the tool name to use as the API identifier.
// Prefers ToolName (canonical), falls back to Name.
func getToolName(data *resource_ai_custom_tool.AiCustomToolModel) string {
	if !data.ToolName.IsNull() && !data.ToolName.IsUnknown() {
		return data.ToolName.ValueString()
	}
	return data.Name.ValueString()
}

// buildCustomToolCreateRequest maps the Terraform model onto the SDK create
// request. Split out from callCustomToolCreateAPI so the wire shape is unit
// testable.
func buildCustomToolCreateRequest(data *resource_ai_custom_tool.AiCustomToolModel) (sdkReq *quantadmingo.CreateCustomToolRequest, diags diag.Diagnostics) {
	// Build the SDK request. InputSchema is a JSON-encoded string in both the TF
	// schema and the SDK request (v4.19.0+), but the API validates inputSchema
	// with an array/object rule and rejects a JSON string with 422. Decode it
	// and send the object via AdditionalProperties (same pattern as
	// outputSchema below) — AdditionalProperties is written after the typed
	// fields in CreateCustomToolRequest.ToMap(), so it overrides them.
	inputSchema := `{"type":"object","properties":{}}`
	if !data.InputSchema.IsNull() && !data.InputSchema.IsUnknown() && data.InputSchema.ValueString() != "" {
		inputSchema = data.InputSchema.ValueString()
	}
	var inputSchemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(inputSchema), &inputSchemaMap); err != nil {
		diags.AddError("Invalid input_schema JSON", fmt.Sprintf("Failed to parse input_schema: %s", err.Error()))
		return
	}

	sdkReq = quantadmingo.NewCreateCustomToolRequest(
		data.Name.ValueString(),
		data.Description.ValueString(),
		"", // edgeFunctionUrl — computed by API from edgeFunctionCode
		inputSchema,
	)

	if !data.IsAsync.IsNull() && !data.IsAsync.IsUnknown() {
		sdkReq.SetIsAsync(data.IsAsync.ValueBool())
	}
	if !data.TimeoutSeconds.IsNull() && !data.TimeoutSeconds.IsUnknown() {
		sdkReq.SetTimeoutSeconds(int32(data.TimeoutSeconds.ValueInt64()))
	}

	// Fields not in the typed SDK struct — set via AdditionalProperties.
	if sdkReq.AdditionalProperties == nil {
		sdkReq.AdditionalProperties = make(map[string]interface{})
	}
	// edgeFunctionCode is required — the API deploys it and computes the URL.
	sdkReq.AdditionalProperties["edgeFunctionCode"] = data.EdgeFunctionCode.ValueString()
	// InputSchema must reach the API as an object, not a JSON string.
	sdkReq.AdditionalProperties["inputSchema"] = inputSchemaMap
	if !data.Category.IsNull() && !data.Category.IsUnknown() {
		sdkReq.AdditionalProperties["category"] = data.Category.ValueString()
	}
	if !data.OutputSchemaDescription.IsNull() && !data.OutputSchemaDescription.IsUnknown() {
		sdkReq.AdditionalProperties["outputSchemaDescription"] = data.OutputSchemaDescription.ValueString()
	}
	if !data.ResponseMode.IsNull() && !data.ResponseMode.IsUnknown() {
		sdkReq.AdditionalProperties["responseMode"] = data.ResponseMode.ValueString()
	}
	// OutputSchema — parse JSON string to object for the API.
	if !data.OutputSchema.IsNull() && !data.OutputSchema.IsUnknown() {
		var outputSchema map[string]interface{}
		if err := json.Unmarshal([]byte(data.OutputSchema.ValueString()), &outputSchema); err != nil {
			diags.AddError("Invalid output_schema JSON", fmt.Sprintf("Failed to parse output_schema: %s", err.Error()))
			return
		}
		sdkReq.AdditionalProperties["outputSchema"] = outputSchema
	}

	return sdkReq, diags
}

func callCustomToolCreateAPI(ctx context.Context, r *aiCustomToolResource, data *resource_ai_custom_tool.AiCustomToolModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq, buildDiags := buildCustomToolCreateRequest(data)
	diags.Append(buildDiags...)
	if diags.HasError() {
		return
	}

	sdkResp, httpResp, err := r.client.Instance.AICustomToolsAPI.CreateCustomTool(r.client.AuthContext, org).
		CreateCustomToolRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to create AI custom tool",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to create AI custom tool",
				fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Map the response tool map into state.
	if sdkResp != nil {
		toolMap := sdkResp.GetTool()
		if toolMap != nil {
			diags.Append(mapCustomToolFromMap(ctx, toolMap, org, data)...)
		}
	}

	data.Organisation = types.StringValue(org)
	return
}

// callCustomToolReadAPI reads a single custom tool by name.
// The SDK only has ListCustomTools, so we use a raw HTTP GET to
// /ai/custom-tools/{toolName} which returns {tool: {...}}.
func callCustomToolReadAPI(ctx context.Context, r *aiCustomToolResource, data *resource_ai_custom_tool.AiCustomToolModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)
	toolName := getToolName(data)

	// Build the raw URL using the SDK's configured base URL.
	cfg := r.client.Instance.GetConfig()
	baseURL := cfg.Servers[0].URL
	url := fmt.Sprintf("%s/api/v3/organizations/%s/ai/custom-tools/%s", baseURL, org, toolName)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		diags.AddError("Unable to read AI custom tool",
			fmt.Sprintf("Error creating request: %s", err.Error()))
		return
	}

	// Copy authorization header from SDK config.
	for k, v := range cfg.DefaultHeader {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := cfg.HTTPClient.Do(httpReq)
	if err != nil {
		diags.AddError("Unable to read AI custom tool",
			fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode == http.StatusNotFound {
		// Signal "not found" by nulling Name — caller handles state removal.
		data.Name = types.StringNull()
		return
	}

	if httpResp.StatusCode >= 300 {
		diags.AddError("Unable to read AI custom tool",
			fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	// Parse response: {tool: {...}, success: bool}
	var respBody map[string]interface{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		diags.AddError("Unable to read AI custom tool",
			fmt.Sprintf("Error parsing response: %s", err.Error()))
		return
	}

	toolMap, ok := respBody["tool"].(map[string]interface{})
	if !ok || toolMap == nil {
		// No tool in the response — treat as not found.
		data.Name = types.StringNull()
		return
	}

	diags.Append(mapCustomToolFromMap(ctx, toolMap, org, data)...)
	data.Organisation = types.StringValue(org)
	return
}

func callCustomToolDeleteAPI(ctx context.Context, r *aiCustomToolResource, data *resource_ai_custom_tool.AiCustomToolModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)
	toolName := getToolName(data)

	_, httpResp, err := r.client.Instance.AICustomToolsAPI.DeleteCustomTool(r.client.AuthContext, org, toolName).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to delete AI custom tool",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(body)))
		} else {
			diags.AddError("Unable to delete AI custom tool",
				fmt.Sprintf("Error: %s", err.Error()))
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Response mapping
// ---------------------------------------------------------------------------

// mapCustomToolFromMap converts a generic tool map (from API responses) into
// the Terraform model.
func mapCustomToolFromMap(ctx context.Context, m map[string]interface{}, org string, data *resource_ai_custom_tool.AiCustomToolModel) (diags diag.Diagnostics) {
	if m == nil {
		return
	}

	// Top-level configurable fields.
	data.Name = mapStringFromAny(m, "name")
	data.Description = mapStringFromAny(m, "description")
	data.EdgeFunctionUrl = mapStringFromAny(m, "edgeFunctionUrl")
	data.EdgeFunctionCode = mapStringFromAny(m, "edgeFunctionCode")
	data.Category = mapStringFromAny(m, "category")
	data.OutputSchemaDescription = mapStringFromAny(m, "outputSchemaDescription")
	data.ResponseMode = mapStringFromAny(m, "responseMode")

	// ToolName — try both "toolName" and "name" from API response.
	data.ToolName = mapStringFromAny(m, "toolName")
	if data.ToolName.IsNull() {
		data.ToolName = data.Name
	}
	if data.Name.IsNull() {
		data.Name = data.ToolName
	}

	// IsAsync
	data.IsAsync = mapBoolFromAny(m, "isAsync")

	// TimeoutSeconds — API returns "timeout" in milliseconds, TF uses seconds.
	if v, ok := m["timeout"]; ok {
		switch t := v.(type) {
		case float64:
			data.TimeoutSeconds = types.Int64Value(int64(t) / 1000)
		case int64:
			data.TimeoutSeconds = types.Int64Value(t / 1000)
		default:
			data.TimeoutSeconds = types.Int64Null()
		}
	} else {
		data.TimeoutSeconds = types.Int64Null()
	}

	// InputSchema — JSON string. Keep the user's original string if it's
	// semantically equivalent to the API response (avoids key-ordering diffs).
	data.InputSchema = normalizeJSONSchemaField(m, "inputSchema", data.InputSchema)

	// OutputSchema — same treatment.
	data.OutputSchema = normalizeJSONSchemaField(m, "outputSchema", data.OutputSchema)

	data.Organisation = types.StringValue(org)

	// Computed-only response metadata.
	data.Message = types.StringNull()
	data.Success = types.BoolValue(true)
	data.IsUpdate = mapBoolFromAny(m, "isUpdate")

	// Nested computed `tool` object — set to null to avoid perpetual bridge
	// diffs. All tool fields are already available at the top level.
	data.Tool = resource_ai_custom_tool.NewToolValueNull()

	return
}

// normalizeJSONSchemaField compares the API response value to the existing state
// value. If they're semantically equivalent JSON objects, keep the state string
// to avoid perpetual diffs from key reordering.
func normalizeJSONSchemaField(m map[string]interface{}, key string, existing types.String) types.String {
	apiVal, ok := m[key]
	if !ok || apiVal == nil {
		return types.StringNull()
	}

	apiJSON, err := json.Marshal(apiVal)
	if err != nil {
		return types.StringNull()
	}

	// If state already has a value, check semantic equality.
	if !existing.IsNull() && !existing.IsUnknown() {
		var stateObj, apiObj interface{}
		if json.Unmarshal([]byte(existing.ValueString()), &stateObj) == nil &&
			json.Unmarshal(apiJSON, &apiObj) == nil {
			stateNorm, _ := json.Marshal(stateObj)
			apiNorm, _ := json.Marshal(apiObj)
			if string(stateNorm) == string(apiNorm) {
				return existing // Keep user's original string.
			}
		}
	}

	return types.StringValue(string(apiJSON))
}
