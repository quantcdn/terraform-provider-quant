package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-quant/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
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
	Id           types.String `tfsdk:"id"`
	Headers      types.Map    `tfsdk:"headers"`
	Organization types.String `tfsdk:"organization"`
	Project      types.String `tfsdk:"project"`
}

func (r *headerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_header"
}

func (r *headerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}

	r.client = client
}

func (r *headerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data headerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callHeaderCreateUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data headerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callHeaderReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data headerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callHeaderCreateUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data headerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callHeaderDeleteAPI(ctx, r, &data)...)
}

func (r *headerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data headerResourceModel
	data.Project = types.StringValue(req.ID)

	resp.Diagnostics.Append(callHeaderReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *headerResource) getOrg(data *headerResourceModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

// generateID produces a deterministic hash from header key-value pairs.
func generateID(headers map[string]string) string {
	var keys []string
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(headers[k])
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

// extractHeadersMap converts the TF map attribute to a Go map[string]string.
func extractHeadersMap(data *headerResourceModel) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	headers := make(map[string]string, len(data.Headers.Elements()))
	for k, v := range data.Headers.Elements() {
		strVal, ok := v.(types.String)
		if !ok {
			diags.AddError("Invalid header value type", "Expected string value")
			return nil, diags
		}
		headers[k] = strVal.ValueString()
	}
	return headers, diags
}

// callHeaderCreateUpdateAPI creates or updates custom headers via the API.
func callHeaderCreateUpdateAPI(ctx context.Context, h *headerResource, data *headerResourceModel) (diags diag.Diagnostics) {
	headers, d := extractHeadersMap(data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	req := *quantadmingo.NewV2CustomHeaderRequest(headers)

	org := h.getOrg(data)
	_, _, err := h.client.Instance.HeadersAPI.HeadersCreate(
		h.client.AuthContext, org, data.Project.ValueString(),
	).V2CustomHeaderRequest(req).Execute()

	if err != nil {
		diags.AddError("Failed to add custom headers", err.Error())
		return
	}

	data.Id = types.StringValue(generateID(headers))
	data.Organization = types.StringValue(org)
	return
}

// callHeaderReadAPI reads custom headers from the API.
func callHeaderReadAPI(ctx context.Context, h *headerResource, data *headerResourceModel) (diags diag.Diagnostics) {
	org := h.getOrg(data)
	allHeaders, _, err := h.client.Instance.HeadersAPI.HeadersList(
		h.client.AuthContext, org, data.Project.ValueString(),
	).Execute()

	if err != nil {
		diags.AddError("Error getting custom headers", err.Error())
		return
	}

	a := make(map[string]attr.Value, len(allHeaders))
	for k, v := range allHeaders {
		a[k] = types.StringValue(v)
	}

	headers, d := types.MapValue(types.StringType, a)
	if d.HasError() {
		diags.Append(d...)
		return
	}

	data.Id = types.StringValue(generateID(allHeaders))
	data.Headers = headers
	data.Organization = types.StringValue(org)
	return
}

// callHeaderDeleteAPI deletes custom headers via the API.
func callHeaderDeleteAPI(ctx context.Context, h *headerResource, data *headerResourceModel) (diags diag.Diagnostics) {
	headersToDelete := make(map[string]string, len(data.Headers.Elements()))
	for k := range data.Headers.Elements() {
		headersToDelete[k] = ""
	}
	req := *quantadmingo.NewV2CustomHeaderRequest(headersToDelete)

	org := h.getOrg(data)
	_, err := h.client.Instance.HeadersAPI.HeadersDelete(
		h.client.AuthContext, org, data.Project.ValueString(),
	).V2CustomHeaderRequest(req).Execute()

	if err != nil {
		diags.AddError("Error removing custom headers", err.Error())
	}
	return
}
