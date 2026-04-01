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
	data.CollectionId = types.StringValue(req.ID)

	resp.Diagnostics.Append(callVectorDocumentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
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

	return
}
