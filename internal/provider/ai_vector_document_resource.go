package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
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

	// If the collection or documents were deleted externally, remove from state.
	if data.CollectionId.IsNull() {
		resp.State.RemoveResource(ctx)
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

// ImportState imports vector documents. Format: "collection-id" or "collection-id/key"
func (r *aiVectorDocumentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resource_ai_vector_document.AiVectorDocumentModel

	// Parse "collection-id" or "collection-id/key" format.
	parts := strings.SplitN(req.ID, "/", 2)
	data.CollectionId = types.StringValue(parts[0])
	if len(parts) == 2 {
		data.Key = types.StringValue(parts[1])
	}

	resp.Diagnostics.Append(callVectorDocumentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve all computed fields that the Read didn't populate to avoid
	// bridge Value Conversion errors on import.
	if data.ChunksCreated.IsUnknown() {
		data.ChunksCreated = types.Int64Null()
	}
	if data.Success.IsUnknown() {
		data.Success = types.BoolValue(true)
	}
	if data.Message.IsUnknown() {
		data.Message = types.StringNull()
	}
	if data.Organisation.IsUnknown() {
		data.Organisation = types.StringValue(r.getOrg(&data))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

// callVectorDocumentUpsertAPI uploads documents via the SDK's UploadVectorDocuments.
func callVectorDocumentUpsertAPI(ctx context.Context, r *aiVectorDocumentResource, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	// Extract documents from the model.
	var docElements []resource_ai_vector_document.DocumentsValue
	diags.Append(data.Documents.ElementsAs(ctx, &docElements, false)...)
	if diags.HasError() {
		return
	}

	sdkDocs := make([]quantadmingo.UploadVectorDocumentsRequestDocumentsInner, 0, len(docElements))
	for _, elem := range docElements {
		doc := quantadmingo.NewUploadVectorDocumentsRequestDocumentsInner(elem.Content.ValueString())
		if !elem.Key.IsNull() && !elem.Key.IsUnknown() {
			doc.SetKey(elem.Key.ValueString())
		}
		sdkDocs = append(sdkDocs, *doc)
	}

	req := quantadmingo.NewUploadVectorDocumentsRequest(sdkDocs)

	result, httpResp, err := r.client.Instance.AIVectorDatabaseAPI.UploadVectorDocuments(
		r.client.AuthContext, org, data.CollectionId.ValueString(),
	).UploadVectorDocumentsRequest(*req).Execute()
	if err != nil {
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to create vector document",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to create vector document", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	// Map response — DocumentIds from API.
	if docIds := result.GetDocumentIds(); len(docIds) > 0 {
		dl, d := types.ListValueFrom(ctx, types.StringType, docIds)
		diags.Append(d...)
		data.DocumentIds = dl
	} else {
		data.DocumentIds = types.ListNull(types.StringType)
	}

	data.Organisation = types.StringValue(org)
	data.ChunksCreated = types.Int64Value(int64(result.GetChunksCreated()))
	data.Success = types.BoolValue(result.GetSuccess())
	if msg, ok := result.GetMessageOk(); ok && msg != nil && *msg != "" {
		data.Message = types.StringValue(*msg)
	} else {
		data.Message = types.StringNull()
	}

	// Key, Limit, Offset — preserve from plan (user-supplied); null if unknown.
	if data.Key.IsUnknown() {
		data.Key = types.StringNull()
	}
	if data.Limit.IsUnknown() {
		data.Limit = types.Int64Null()
	}
	if data.Offset.IsUnknown() {
		data.Offset = types.Int64Null()
	}

	// Resolve unknown sub-fields in Documents to prevent Pulumi bridge panic.
	// After Create, plan values for Computed sub-fields (key, metadata and its
	// children) may still be "unknown". Terraform/Pulumi requires all Computed
	// fields in state to be known (or null) after Apply.
	metadataAttrTypes := resource_ai_vector_document.MetadataValue{}.AttributeTypes(ctx)

	var resolvedDocs []resource_ai_vector_document.DocumentsValue
	diags.Append(data.Documents.ElementsAs(ctx, &resolvedDocs, false)...)
	if diags.HasError() {
		return
	}

	for i := range resolvedDocs {
		// key is Optional+Computed — resolve to null if still unknown.
		if resolvedDocs[i].Key.IsUnknown() {
			resolvedDocs[i].Key = basetypes.NewStringNull()
		}
		// metadata is Optional+Computed — resolve the whole object to null when
		// unknown, which also covers its unknown children.
		if resolvedDocs[i].Metadata.IsUnknown() {
			resolvedDocs[i].Metadata = basetypes.NewObjectNull(metadataAttrTypes)
		} else if !resolvedDocs[i].Metadata.IsNull() {
			// metadata is known (user supplied it) — ensure its Computed children
			// are resolved too.
			attrs := resolvedDocs[i].Metadata.Attributes()
			changed := false

			if s, ok := attrs["section"].(basetypes.StringValue); ok && s.IsUnknown() {
				attrs["section"] = basetypes.NewStringNull()
				changed = true
			}
			if s, ok := attrs["source_url"].(basetypes.StringValue); ok && s.IsUnknown() {
				attrs["source_url"] = basetypes.NewStringNull()
				changed = true
			}
			if l, ok := attrs["tags"].(basetypes.ListValue); ok && l.IsUnknown() {
				attrs["tags"] = basetypes.NewListNull(types.StringType)
				changed = true
			}
			if s, ok := attrs["title"].(basetypes.StringValue); ok && s.IsUnknown() {
				attrs["title"] = basetypes.NewStringNull()
				changed = true
			}

			if changed {
				rebuilt, d := basetypes.NewObjectValue(metadataAttrTypes, attrs)
				diags.Append(d...)
				if !diags.HasError() {
					resolvedDocs[i].Metadata = rebuilt
				}
			}
		}
	}

	if !diags.HasError() && len(resolvedDocs) > 0 {
		docListType := resolvedDocs[0].Type(ctx)
		docList, d := types.ListValueFrom(ctx, docListType, resolvedDocs)
		diags.Append(d...)
		if !diags.HasError() {
			data.Documents = docList
		}
	}

	return
}

func callVectorDocumentReadAPI(ctx context.Context, r *aiVectorDocumentResource, data *resource_ai_vector_document.AiVectorDocumentModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	listReq := r.client.Instance.AIVectorDatabaseAPI.ListVectorDocuments(r.client.AuthContext, org, data.CollectionId.ValueString())
	if !data.Key.IsNull() && !data.Key.IsUnknown() {
		listReq = listReq.Key(data.Key.ValueString())
	}

	httpResp, err := listReq.Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			// Signal "not found" by nulling CollectionId — caller handles state removal.
			data.CollectionId = types.StringNull()
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

	// Collect keys from documents for deletion.
	var docElements []resource_ai_vector_document.DocumentsValue
	diags.Append(data.Documents.ElementsAs(ctx, &docElements, false)...)
	if diags.HasError() {
		return
	}

	keys := make([]string, 0, len(docElements))
	for _, elem := range docElements {
		if !elem.Key.IsNull() && !elem.Key.IsUnknown() {
			keys = append(keys, elem.Key.ValueString())
		}
	}

	if len(keys) == 0 {
		return
	}

	sdkReq := quantadmingo.NewDeleteVectorDocumentsRequest()
	sdkReq.Keys = keys

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

// parseVectorDocumentReadResponse reads the GET documents response.
//
// Expected response shape:
//
//	{
//	  "documents": [
//	    { "documentId": "uuid", "key": "...", "content": "..." }
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
			DocumentId string `json:"documentId"`
			Key        string `json:"key"`
			Content    string `json:"content"`
		} `json:"documents"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		diags.AddError("Unable to parse vector document response",
			fmt.Sprintf("Error: %s\nBody: %s", err.Error(), string(respBody)))
		return
	}

	// Collect document IDs.
	docIds := make([]string, 0, len(envelope.Documents))
	for _, doc := range envelope.Documents {
		docIds = append(docIds, doc.DocumentId)
	}

	if len(docIds) > 0 {
		dl, d := types.ListValueFrom(ctx, types.StringType, docIds)
		diags.Append(d...)
		data.DocumentIds = dl
	} else {
		data.DocumentIds = types.ListNull(types.StringType)
	}

	data.Organisation = types.StringValue(org)

	// Computed-only response metadata — not available from the list endpoint.
	if data.ChunksCreated.IsUnknown() {
		data.ChunksCreated = types.Int64Null()
	}
	data.Success = types.BoolValue(true) // GET succeeded
	data.Message = types.StringNull()

	// Key, Limit, Offset — preserve from state; null if unknown.
	if data.Key.IsUnknown() {
		data.Key = types.StringNull()
	}
	if data.Limit.IsUnknown() {
		data.Limit = types.Int64Null()
	}
	if data.Offset.IsUnknown() {
		data.Offset = types.Int64Null()
	}

	// Reconstruct Documents list from API response so Read/import has
	// the full input. Always rebuild the list to ensure the element type
	// is correctly set (a zero-value List has a missing element type which
	// causes Pulumi bridge Value Conversion errors).
	docAttrTypes := resource_ai_vector_document.DocumentsValue{}.AttributeTypes(ctx)
	metadataAttrTypes := resource_ai_vector_document.MetadataValue{}.AttributeTypes(ctx)

	// Get a canonical DocumentsType by building a populated value.
	canonicalDoc, _ := resource_ai_vector_document.NewDocumentsValue(
		docAttrTypes,
		map[string]attr.Value{
			"content":  types.StringNull(),
			"key":      types.StringNull(),
			"metadata": basetypes.NewObjectNull(metadataAttrTypes),
		},
	)
	docListType := canonicalDoc.Type(ctx)

	docValues := make([]resource_ai_vector_document.DocumentsValue, 0, len(envelope.Documents))
	for _, doc := range envelope.Documents {
		dv, d := resource_ai_vector_document.NewDocumentsValue(
			docAttrTypes,
			map[string]attr.Value{
				"content":  types.StringValue(doc.Content),
				"key":      types.StringValue(doc.Key),
				"metadata": basetypes.NewObjectNull(metadataAttrTypes),
			},
		)
		diags.Append(d...)
		if !diags.HasError() {
			docValues = append(docValues, dv)
		}
	}

	if len(docValues) > 0 {
		docList, d := types.ListValueFrom(ctx, docListType, docValues)
		diags.Append(d...)
		if !diags.HasError() {
			data.Documents = docList
		}
	} else {
		data.Documents = types.ListNull(docListType)
	}

	return
}
