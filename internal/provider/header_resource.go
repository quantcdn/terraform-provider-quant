package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	quantadmingo "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource                = (*headerResource)(nil)
	_ resource.ResourceWithConfigure   = (*headerResource)(nil)
	_ resource.ResourceWithImportState = (*headerResource)(nil)
)

func NewHeaderResource() resource.Resource {
	return &headerResource{}
}

type headerResource struct {
	client *client.Client
}

type headerResourceModel struct {
	Id      types.String `tfsdk:"id"`
	Headers types.Map    `tfsdk:"headers"`
	Project types.String `tfsdk:"project"`
}

func (r *headerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_header"
}

func (r *headerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"project": schema.StringAttribute{
				Required: true,
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "HTTP headers to be set for the project",
			},
		},
	}
}

func (r *headerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *internal.Client, got: %T. Please report this issue to the provider developers", req.ProviderData),
		)
	}

	r.client = client
}

func (r *headerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data headerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	resp.Diagnostics.Append(callHeaderCreateUpdateAPI(ctx, r, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data headerResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	resp.Diagnostics.Append(callHeaderReadAPI(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data headerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic
	resp.Diagnostics.Append(callHeaderCreateUpdateAPI(ctx, r, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data headerResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callHeaderDeleteAPI(ctx, r, &data)...)
}

func (r *headerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data headerResourceModel

	data.Project = types.StringValue(req.ID)

	// Read API call logic
	resp.Diagnostics.Append(callHeaderReadAPI(ctx, r, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Generate the ID from the header value.
func generateID(headers map[string]string) string {
	var keys []string
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create a string builder to concatenate key-value pairs
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(headers[k])
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

// Create headers with the API.
func callHeaderCreateUpdateAPI(ctx context.Context, h *headerResource, resource *headerResourceModel) (diags diag.Diagnostics) {
	req := *quantadmingo.NewV2CustomHeaderRequest(make(map[string]string))

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}

	headers := resource.Headers.Elements()
	for k, v := range headers {
		strVal, ok := v.(types.String)
		if !ok {
			diags.AddError("Invalid header value type", "Expected string value")
			return
		}
		req.Headers[k] = strVal.ValueString()
	}

	_, _, err := h.client.Instance.HeadersAPI.HeadersCreate(h.client.AuthContext, h.client.Organization, resource.Project.ValueString()).V2CustomHeaderRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to add custom headers", err.Error())
		return
	}

	resource.Id = types.StringValue(generateID(req.Headers))
	return
}

// Load headers from the API.
func callHeaderReadAPI(ctx context.Context, h *headerResource, resource *headerResourceModel) (diags diag.Diagnostics) {
	api, httpResp, err := h.client.Instance.HeadersAPI.HeadersList(h.client.AuthContext, h.client.Organization, resource.Project.ValueString()).Execute()

	// Debug: Log the raw response
	if httpResp != nil {
		tflog.Debug(ctx, "HeadersList API Response", map[string]interface{}{
			"status": httpResp.Status,
		})
	}
	
	// Debug: Log what we got back
	apiJSON, _ := json.Marshal(api)
	tflog.Debug(ctx, "HeadersList API Data", map[string]interface{}{
		"raw_response": string(apiJSON),
		"type":         fmt.Sprintf("%T", api),
	})

	if err != nil {
		diags.AddError("Error getting custom headers", err.Error())
		return
	}

	// V2 API returns a single map[string]string, not an array
	allHeaders := api
	
	a := make(map[string]attr.Value)
	for k, v := range allHeaders {
		a[k] = types.StringValue(v)
	}

	headers, d := types.MapValue(types.StringType, a)

	if d.HasError() {
		diags.Append(d...)
		return
	}

	resource.Id = types.StringValue(generateID(allHeaders))
	resource.Headers = headers
	return
}

// To delete headers we remove just update with an empty map.
func callHeaderDeleteAPI(ctx context.Context, h *headerResource, resource *headerResourceModel) (diags diag.Diagnostics) {
	// V2CustomHeaderRequest expects headers map, not array of header names
	headersToDelete := make(map[string]string)
	for k := range resource.Headers.Elements() {
		headersToDelete[k] = "" // Empty value to indicate deletion
	}
	req := *quantadmingo.NewV2CustomHeaderRequest(headersToDelete)
	_, err := h.client.Instance.HeadersAPI.HeadersDelete(h.client.AuthContext, h.client.Organization, resource.Project.ValueString()).V2CustomHeaderRequest(req).Execute()
	if err != nil {
		diags.AddError("Error removing custom headers", err.Error())
		return
	}
	return
}
