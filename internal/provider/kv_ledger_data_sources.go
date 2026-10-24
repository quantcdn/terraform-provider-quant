package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/datasource_kv_ledger"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/datasource_kv_ledger_public_key"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/mapper"
)

var (
	_ datasource.DataSourceWithConfigure = (*kvLedgerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kvLedgerPublicKeyDataSource)(nil)
)

// kvDataSource holds the client for the KV data sources.
type kvDataSource struct{ client *client.Client }

func (d *kvDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected data source configure type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	d.client = c
}

// optionalOrganization lets the provider organization apply, as it does for
// the resources. The generated schema makes the path parameter required.
func optionalOrganization(s *schema.Schema) {
	s.Attributes["organization"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "Organization identifier. Defaults to the provider organization.",
		MarkdownDescription: "Organization identifier. Defaults to the provider organization.",
	}
}

// --- quant_kv_ledger ---

type kvLedgerDataSource struct{ kvDataSource }

func NewKVLedgerDataSource() datasource.DataSource { return &kvLedgerDataSource{} }

func (d *kvLedgerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kv_ledger"
}

func (d *kvLedgerDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := datasource_kv_ledger.KvLedgerDataSourceSchema(ctx)
	s.Description = "A KV ledger definition, its state, its retention end and the last verification."
	s.MarkdownDescription = s.Description
	optionalOrganization(&s)
	resp.Schema = s
}

func (d *kvLedgerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m datasource_kv_ledger.KvLedgerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m.Organization = types.StringValue(orgOr(m.Organization, d.client.Organization))
	l, httpResp, err := d.client.Instance.KVAPI.KVLedgersShow(d.client.AuthContext, m.Organization.ValueString(), m.Project.ValueString(), m.StoreId.ValueString(), m.LedgerName.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the KV ledger", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(fillKVLedgerDataSource(ctx, &m, l)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func fillKVLedgerDataSource(ctx context.Context, m *datasource_kv_ledger.KvLedgerModel, l *quantadmingo.V2KvLedgerDetail) diag.Diagnostics {
	diags := mapper.FromSDK(ctx, l, m, "organization", "project", "store_id", "ledger_name", "subject_field", "error",
		"created_at", "active_after", "updated_at", "dropped_at", "retain_until", "last_digest_at")
	m.SubjectField = optionalString(l.SubjectField)
	m.Error = optionalString(l.Error)
	m.CreatedAt = timeString(l.GetCreatedAt())
	m.ActiveAfter = timeString(l.GetActiveAfter())
	m.UpdatedAt = kvOptionalTime(l.UpdatedAt)
	m.DroppedAt = kvOptionalTime(l.DroppedAt)
	m.RetainUntil = kvOptionalTime(l.RetainUntil)
	m.LastDigestAt = kvOptionalTime(l.LastDigestAt)
	lastVerify, more := withLastVerifyTimes(ctx, m.LastVerify, l.LastVerify)
	m.LastVerify = lastVerify
	diags.Append(more...)
	return diags
}

// withLastVerifyTimes sets the two time attributes of last_verify, which the
// mapper leaves null because it does not convert time.Time.
func withLastVerifyTimes(ctx context.Context, v datasource_kv_ledger.LastVerifyValue, src *quantadmingo.V2KvLedgerLastVerify) (datasource_kv_ledger.LastVerifyValue, diag.Diagnostics) {
	if src == nil || v.IsNull() || v.IsUnknown() {
		return v, nil
	}
	obj, diags := v.ToObjectValue(ctx)
	if diags.HasError() {
		return v, diags
	}
	attrs := make(map[string]attr.Value, len(obj.Attributes()))
	for k, a := range obj.Attributes() {
		attrs[k] = a
	}
	attrs["at"] = timeString(src.GetAt())
	attrs["started_at"] = timeString(src.GetStartedAt())
	out, more := datasource_kv_ledger.NewLastVerifyValue(obj.AttributeTypes(ctx), attrs)
	diags.Append(more...)
	return out, diags
}

// --- quant_kv_ledger_public_key ---

type kvLedgerPublicKeyDataSource struct{ kvDataSource }

func NewKVLedgerPublicKeyDataSource() datasource.DataSource { return &kvLedgerPublicKeyDataSource{} }

func (d *kvLedgerPublicKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kv_ledger_public_key"
}

func (d *kvLedgerPublicKeyDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := datasource_kv_ledger_public_key.KvLedgerPublicKeyDataSourceSchema(ctx)
	s.Description = "The public key (PEM) that verifies the KV ledger digest signatures of a store, with the current and historical keys."
	s.MarkdownDescription = s.Description
	optionalOrganization(&s)
	resp.Schema = s
}

func (d *kvLedgerPublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m datasource_kv_ledger_public_key.KvLedgerPublicKeyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m.Organization = types.StringValue(orgOr(m.Organization, d.client.Organization))
	k, httpResp, err := d.client.Instance.KVAPI.KVLedgersPublicKey(d.client.AuthContext, m.Organization.ValueString(), m.Project.ValueString(), m.StoreId.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read the KV ledger public key", extractAPIErrorMessage(httpResp, err))
		return
	}
	resp.Diagnostics.Append(mapper.FromSDK(ctx, k, &m, "organization", "project", "store_id")...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
