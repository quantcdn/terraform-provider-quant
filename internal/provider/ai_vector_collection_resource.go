package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
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
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
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

	// If the collection was not found, remove from state so Terraform plans recreation.
	if data.CollectionId.IsNull() {
		resp.State.RemoveResource(ctx)
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
		data.Organisation = types.StringValue(parts[0])
		data.CollectionId = types.StringValue(parts[1])
	} else {
		data.CollectionId = types.StringValue(id)
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

	sdkReq := quantadmingo.NewCreateVectorCollectionRequest(data.Name.ValueString())
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		sdkReq.Description = &desc
	}
	if !data.EmbeddingModel.IsNull() && !data.EmbeddingModel.IsUnknown() {
		model := data.EmbeddingModel.ValueString()
		sdkReq.EmbeddingModel = &model
	} else {
		// Default embedding model when not specified
		defaultModel := "amazon.titan-embed-text-v2:0"
		sdkReq.EmbeddingModel = &defaultModel
	}

	sdkResp, httpResp, err := r.client.Instance.AIVectorDatabaseAPI.CreateVectorCollection(r.client.AuthContext, org).
		CreateVectorCollectionRequest(*sdkReq).Execute()
	if err != nil {
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to create vector collection",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to create vector collection", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	diags.Append(mapCreateVectorCollectionResponse(sdkResp, org, data)...)
	return
}

func callVectorCollectionReadAPI(ctx context.Context, r *aiVectorCollectionResource, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	sdkResp, httpResp, err := r.client.Instance.AIVectorDatabaseAPI.GetVectorCollection(r.client.AuthContext, org, data.CollectionId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			// Signal "not found" by nulling Id — caller handles state removal.
			data.CollectionId = types.StringNull()
			return
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to read vector collection",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to read vector collection", fmt.Sprintf("Error: %s", err.Error()))
		}
		return
	}

	diags.Append(mapGetVectorCollectionResponse(sdkResp, org, data)...)
	return
}

func callVectorCollectionDeleteAPI(ctx context.Context, r *aiVectorCollectionResource, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	org := r.getOrg(data)

	_, httpResp, err := r.client.Instance.AIVectorDatabaseAPI.DeleteVectorCollection(r.client.AuthContext, org, data.CollectionId.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		if httpResp != nil {
			respBody, _ := io.ReadAll(httpResp.Body)
			diags.AddError("Unable to delete vector collection",
				fmt.Sprintf("API returned %d: %s", httpResp.StatusCode, string(respBody)))
		} else {
			diags.AddError("Unable to delete vector collection", fmt.Sprintf("Error: %s", err.Error()))
		}
	}

	return
}

// mapCreateVectorCollectionResponse maps the SDK CreateVectorCollection201Response
// onto the Terraform model.
func mapCreateVectorCollectionResponse(resp *quantadmingo.CreateVectorCollection201Response, org string, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	col := resp.GetCollection()

	data.CollectionId = types.StringValue(col.GetCollectionId())
	data.Name = types.StringValue(col.GetName())
	data.Organisation = types.StringValue(org)

	if desc, ok := col.GetDescriptionOk(); ok && desc != nil && *desc != "" {
		data.Description = types.StringValue(*desc)
	} else if data.Description.IsNull() || data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	if em, ok := col.GetEmbeddingModelOk(); ok && em != nil && *em != "" {
		data.EmbeddingModel = types.StringValue(*em)
	}

	return
}

// mapGetVectorCollectionResponse maps the SDK GetVectorCollection200Response
// onto the Terraform model.
func mapGetVectorCollectionResponse(resp *quantadmingo.GetVectorCollection200Response, org string, data *resource_ai_vector_collection.AiVectorCollectionModel) (diags diag.Diagnostics) {
	col := resp.GetCollection()

	data.CollectionId = types.StringValue(col.GetCollectionId())
	data.Name = types.StringValue(col.GetName())
	data.Organisation = types.StringValue(org)

	if desc, ok := col.GetDescriptionOk(); ok && desc != nil && *desc != "" {
		data.Description = types.StringValue(*desc)
	} else if data.Description.IsNull() || data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}

	if em, ok := col.GetEmbeddingModelOk(); ok && em != nil && *em != "" {
		data.EmbeddingModel = types.StringValue(*em)
	}

	return
}
