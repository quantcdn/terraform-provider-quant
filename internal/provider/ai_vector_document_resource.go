package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_vector_document"
)

var (
	_ resource.Resource                = (*aiVectorDocumentResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiVectorDocumentResource)(nil)
	_ resource.ResourceWithImportState = (*aiVectorDocumentResource)(nil)
)

func NewAiVectorDocumentResource() resource.Resource {
	return &aiVectorDocumentResource{}
}

type aiVectorDocumentResource struct {
	client *client.Client
}

func (r *aiVectorDocumentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_vector_document"
}

func (r *aiVectorDocumentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_ai_vector_document.AiVectorDocumentResourceSchema(ctx)
}

func (r *aiVectorDocumentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aiVectorDocumentResource) getOrg(data *resource_ai_vector_document.AiVectorDocumentModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func (r *aiVectorDocumentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_vector_document.AiVectorDocumentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorDocumentUpsertAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiVectorDocumentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_vector_document.AiVectorDocumentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorDocumentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update uses the same upsert POST — the Lambda deletes the old document by
// key and inserts the new one.
func (r *aiVectorDocumentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ai_vector_document.AiVectorDocumentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorDocumentUpsertAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiVectorDocumentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_vector_document.AiVectorDocumentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorDocumentDeleteAPI(ctx, r, &data)...)
}

// ImportState imports a vector document. Format: "collection-id/document-key"
func (r *aiVectorDocumentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'collection-id/document-key'.",
		)
		return
	}

	var data resource_ai_vector_document.AiVectorDocumentModel
	data.CollectionId = types.StringValue(parts[0])
	data.Key = types.StringValue(parts[1])

	resp.Diagnostics.Append(callVectorDocumentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

func contentSha256(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// callVectorDocumentUpsertAPI uploads a document via doAIRequest because the
// SDK does not have an UploadVectorDocuments method.
func callVectorDocumentUpsertAPI(ctx context.Context, r *aiVectorDocumentResource, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	doc := map[string]interface{}{
		"key":     data.Key.ValueString(),
		"content": data.Content.ValueString(),
	}

	// metadata
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var meta map[string]string
		diags.Append(data.Metadata.ElementsAs(ctx, &meta, false)...)
		if diags.HasError() {
			return
		}
		doc["metadata"] = meta
	}

	// searchable_fields
	if !data.SearchableFields.IsNull() && !data.SearchableFields.IsUnknown() {
		var fields []string
		diags.Append(data.SearchableFields.ElementsAs(ctx, &fields, false)...)
		if diags.HasError() {
			return
		}
		doc["searchableFields"] = fields
	}

	body := map[string]interface{}{
		"documents": []interface{}{doc},
	}

	apiResp, err := doAIRequest(r.client, http.MethodPost,
		fmt.Sprintf("/api/v3/organizations/%s/ai/vector-db/collections/%s/documents",
			org, data.CollectionId.ValueString()), body)
	if err != nil {
		diags.AddError("Unable to create vector document", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to create vector document",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	// Parse response to get document_id
	diags.Append(parseVectorDocumentUpsertResponse(apiResp, org, data)...)
	return
}

func callVectorDocumentReadAPI(ctx context.Context, r *aiVectorDocumentResource, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	// ListVectorDocuments returns (*http.Response, error) — no typed body.
	httpResp, err := r.client.Instance.AIVectorDatabaseAPI.ListVectorDocuments(r.client.AuthContext, org, data.CollectionId.ValueString()).
		Key(data.Key.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			diags.AddError("Vector document not found",
				fmt.Sprintf("Document with key '%s' not found.", data.Key.ValueString()))
			return
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read vector document",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to read vector document", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	diags.Append(parseVectorDocumentReadResponse(ctx, httpResp, org, data)...)
	return
}

func callVectorDocumentDeleteAPI(ctx context.Context, r *aiVectorDocumentResource, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkReq := quantadmingo.NewDeleteVectorDocumentsRequest()
	sdkReq.Keys = []string{data.Key.ValueString()}

	_, httpResp, err := r.client.Instance.AIVectorDatabaseAPI.DeleteVectorDocuments(r.client.AuthContext, org, data.CollectionId.ValueString()).
		DeleteVectorDocumentsRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to delete vector document",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to delete vector document", fmt.Sprintf("Error: %s", err.Error()))
		}
	}

	return
}

// parseVectorDocumentUpsertResponse reads the create/update response.
//
// Expected response shape:
//
//	{
//	  "documents": [
//	    { "documentId": "uuid", "key": "...", "content": "...", "metadata": {...} }
//	  ]
//	}
func parseVectorDocumentUpsertResponse(apiResp *http.Response, org string, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	respBody, err := io.ReadAll(apiResp.Body)
	if err != nil {
		diags.AddError("Unable to read vector document response", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	var envelope struct {
		Documents []struct {
			DocumentId string `json:"documentId"`
			Key        string `json:"key"`
		} `json:"documents"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		diags.AddError("Unable to parse vector document response",
			fmt.Sprintf("Error: %s\nBody: %s", err.Error(), string(respBody)))
		return
	}

	if len(envelope.Documents) > 0 {
		doc := envelope.Documents[0]
		if doc.DocumentId != "" {
			data.DocumentId = types.StringValue(doc.DocumentId)
		}
	}

	data.Organization = types.StringValue(org)
	data.ContentSha256 = types.StringValue(contentSha256(data.Content.ValueString()))

	return
}

// parseVectorDocumentReadResponse reads the GET documents response and finds
// the document matching the key.
//
// Expected response shape:
//
//	{
//	  "documents": [
//	    { "documentId": "uuid", "key": "...", "content": "...", "metadata": {...}, "searchableFields": [...] }
//	  ]
//	}
func parseVectorDocumentReadResponse(ctx context.Context, apiResp *http.Response, org string, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	respBody, err := io.ReadAll(apiResp.Body)
	if err != nil {
		diags.AddError("Unable to read vector document response", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	var envelope struct {
		Documents []struct {
			DocumentId       string            `json:"documentId"`
			Key              string            `json:"key"`
			Content          string            `json:"content"`
			Metadata         map[string]string `json:"metadata"`
			SearchableFields []string          `json:"searchableFields"`
		} `json:"documents"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		diags.AddError("Unable to parse vector document response",
			fmt.Sprintf("Error: %s\nBody: %s", err.Error(), string(respBody)))
		return
	}

	// Find the document matching our key
	targetKey := data.Key.ValueString()
	var found bool
	for _, doc := range envelope.Documents {
		if doc.Key == targetKey {
			found = true
			data.DocumentId = types.StringValue(doc.DocumentId)
			data.Content = types.StringValue(doc.Content)
			data.ContentSha256 = types.StringValue(contentSha256(doc.Content))

			// metadata
			if len(doc.Metadata) > 0 {
				m, d := types.MapValueFrom(ctx, types.StringType, doc.Metadata)
				diags.Append(d...)
				data.Metadata = m
			} else if data.Metadata.IsNull() || data.Metadata.IsUnknown() {
				data.Metadata = types.MapNull(types.StringType)
			}

			// searchable_fields
			if len(doc.SearchableFields) > 0 {
				sf, d := types.ListValueFrom(ctx, types.StringType, doc.SearchableFields)
				diags.Append(d...)
				data.SearchableFields = sf
			} else if data.SearchableFields.IsNull() || data.SearchableFields.IsUnknown() {
				data.SearchableFields = types.ListNull(types.StringType)
			}

			break
		}
	}

	if !found {
		diags.AddError("Vector document not found",
			fmt.Sprintf("Document with key '%s' not found in API response.", targetKey))
		return
	}

	data.Organization = types.StringValue(org)

	return
}
