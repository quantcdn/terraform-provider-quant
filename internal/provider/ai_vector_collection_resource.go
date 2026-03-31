package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_ai_vector_collection"
)

var (
	_ resource.Resource                = (*aiVectorCollectionResource)(nil)
	_ resource.ResourceWithConfigure   = (*aiVectorCollectionResource)(nil)
	_ resource.ResourceWithImportState = (*aiVectorCollectionResource)(nil)
)

func NewAiVectorCollectionResource() resource.Resource {
	return &aiVectorCollectionResource{}
}

type aiVectorCollectionResource struct {
	client *client.Client
}

func (r *aiVectorCollectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_vector_collection"
}

func (r *aiVectorCollectionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_ai_vector_collection.AiVectorCollectionResourceSchema(ctx)
}

func (r *aiVectorCollectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aiVectorCollectionResource) getOrg(data *resource_ai_vector_collection.AiVectorCollectionModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func (r *aiVectorCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_ai_vector_collection.AiVectorCollectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorCollectionCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *aiVectorCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ai_vector_collection.AiVectorCollectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorCollectionReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not supported — name is RequiresReplace, and description updates
// are not supported by the API.
func (r *aiVectorCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Vector collections cannot be updated in-place. Changes require destroying and recreating the collection.",
	)
}

func (r *aiVectorCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ai_vector_collection.AiVectorCollectionModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callVectorCollectionDeleteAPI(ctx, r, &data)...)
}

// ImportState imports a vector collection by its UUID.
func (r *aiVectorCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID can be just the collection UUID, or "org/collection-uuid"
	id := req.ID
	var data resource_ai_vector_collection.AiVectorCollectionModel

	if parts := strings.SplitN(id, "/", 2); len(parts) == 2 {
		data.Organization = types.StringValue(parts[0])
		data.Id = types.StringValue(parts[1])
	} else {
		data.Id = types.StringValue(id)
	}

	resp.Diagnostics.Append(callVectorCollectionReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

func callVectorCollectionCreateAPI(ctx context.Context, r *aiVectorCollectionResource, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	body := map[string]interface{}{
		"name": data.Name.ValueString(),
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}

	apiResp, err := doAIRequest(r.client, http.MethodPost,
		fmt.Sprintf("/api/v3/organisations/%s/ai/vector-db/collections", org), body)
	if err != nil {
		diags.AddError("Unable to create vector collection", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to create vector collection",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseVectorCollectionResponse(apiResp, org, data)...)
	return
}

func callVectorCollectionReadAPI(ctx context.Context, r *aiVectorCollectionResource, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	apiResp, err := doAIRequest(r.client, http.MethodGet,
		fmt.Sprintf("/api/v3/organisations/%s/ai/vector-db/collections/%s", org, data.Id.ValueString()), nil)
	if err != nil {
		diags.AddError("Unable to read vector collection", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == http.StatusNotFound {
		diags.AddError("Vector collection not found",
			fmt.Sprintf("Collection '%s' not found.", data.Id.ValueString()))
		return
	}

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to read vector collection",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
		return
	}

	diags.Append(parseVectorCollectionResponse(apiResp, org, data)...)
	return
}

func callVectorCollectionDeleteAPI(ctx context.Context, r *aiVectorCollectionResource, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	apiResp, err := doAIRequest(r.client, http.MethodDelete,
		fmt.Sprintf("/api/v3/organisations/%s/ai/vector-db/collections/%s", org, data.Id.ValueString()), nil)
	if err != nil {
		diags.AddError("Unable to delete vector collection", fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode == http.StatusNotFound {
		return // Already deleted
	}

	if apiResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(apiResp.Body)
		diags.AddError("Unable to delete vector collection",
			fmt.Sprintf("API returned %d: %s", apiResp.StatusCode, string(respBody)))
	}

	return
}

// parseVectorCollectionResponse reads the JSON response and maps it onto the
// Terraform model.
//
// Expected response shape:
//
//	{
//	  "collection": {
//	    "collectionId": "uuid",
//	    "name": "my-collection",
//	    "description": "...",
//	    "createdAt": "2026-03-30T..."
//	  }
//	}
func parseVectorCollectionResponse(apiResp *http.Response, org string, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	respBody, err := io.ReadAll(apiResp.Body)
	if err != nil {
		diags.AddError("Unable to read vector collection response", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	var envelope struct {
		Collection struct {
			CollectionId string  `json:"collectionId"`
			Name         string  `json:"name"`
			Description  *string `json:"description"`
			CreatedAt    *string `json:"createdAt"`
		} `json:"collection"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		diags.AddError("Unable to parse vector collection response",
			fmt.Sprintf("Error: %s\nBody: %s", err.Error(), string(respBody)))
		return
	}

	col := envelope.Collection

	data.Id = types.StringValue(col.CollectionId)
	data.Name = types.StringValue(col.Name)
	data.Organization = types.StringValue(org)

	if col.Description != nil && *col.Description != "" {
		data.Description = types.StringValue(*col.Description)
	} else if data.Description.IsNull() || data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	if col.CreatedAt != nil && *col.CreatedAt != "" {
		data.CreatedAt = types.StringValue(*col.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}

	return
}
