package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/mapper"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_kv_ledger"
)

// Fields the mapper must not touch: path parameters, times (the mapper does
// not convert time.Time), optional strings that read back as null, and the
// anchor, whose empty strings must not be sent.
var kvLedgerCallerHandled = []string{
	"organization", "project", "store_id", "ledger_name", "anchor", "subject_field", "subject_mode", "subject_case", "error",
	"created_at", "active_after", "updated_at", "dropped_at", "retain_until",
}

// kvLedgerDeclareRequest builds the declare body. The mapper sends every known
// attribute that has a setter (camelCase on the wire through the SDK),
// including the nested writer_policy.
func kvLedgerDeclareRequest(ctx context.Context, d *resource_kv_ledger.KvLedgerModel) (*quantadmingo.V2KvLedgerAdminDeclareRequest, diag.Diagnostics) {
	req := quantadmingo.NewV2KvLedgerAdminDeclareRequest(d.Name.ValueString(), d.Collection.ValueString(),
		int32(d.Chains.ValueInt64()), int32(d.RetentionDays.ValueInt64()), d.LockMode.ValueString())
	diags := mapper.ToSDK(ctx, d, req, kvLedgerCallerHandled...)
	if isKnown(d.SubjectField) && d.SubjectField.ValueString() != "" {
		req.SetSubjectField(d.SubjectField.ValueString())
	}
	if s, ok := knownNonEmpty(d.SubjectMode); ok {
		req.SetSubjectMode(s)
	}
	if s, ok := knownNonEmpty(d.SubjectCase); ok {
		req.SetSubjectCase(s)
	}
	if isKnown(d.Anchor) {
		req.SetAnchor(kvLedgerAnchor(d.Anchor))
	}
	return req, diags
}

// kvLedgerAnchor sends only the anchor fields that have a value.
func kvLedgerAnchor(a resource_kv_ledger.AnchorValue) quantadmingo.V2KvLedgerAnchor {
	out := quantadmingo.NewV2KvLedgerAnchor(a.AnchorType.ValueString())
	out.Prefix = nil // the constructor presets the spec default ""; send it only when set
	if s, ok := knownNonEmpty(a.BucketArn); ok {
		out.SetBucketArn(s)
	}
	if s, ok := knownNonEmpty(a.Prefix); ok {
		out.SetPrefix(s)
	}
	if s, ok := knownNonEmpty(a.AccountId); ok {
		out.SetAccountId(s)
	}
	if s, ok := knownNonEmpty(a.TsaUrl); ok {
		out.SetTsaUrl(s)
	}
	return *out
}

func knownNonEmpty(v types.String) (string, bool) {
	return v.ValueString(), isKnown(v) && v.ValueString() != ""
}

// kvLedgerRaise sends only the attributes that went up.
func kvLedgerRaise(plan, state resource_kv_ledger.KvLedgerModel) (quantadmingo.V2KvLedgerAdminRaiseRequest, bool) {
	raise := quantadmingo.NewV2KvLedgerAdminRaiseRequest()
	changed := false
	if plan.Chains.ValueInt64() > state.Chains.ValueInt64() {
		raise.SetChains(int32(plan.Chains.ValueInt64()))
		changed = true
	}
	if plan.RetentionDays.ValueInt64() > state.RetentionDays.ValueInt64() {
		raise.SetRetentionDays(int32(plan.RetentionDays.ValueInt64()))
		changed = true
	}
	if lockRank[plan.LockMode.ValueString()] > lockRank[state.LockMode.ValueString()] {
		raise.SetLockMode(plan.LockMode.ValueString())
		changed = true
	}
	return *raise, changed
}

// setKVLedgerState writes every attribute from the API answer, so nothing is
// unknown after apply.
func setKVLedgerState(ctx context.Context, d *resource_kv_ledger.KvLedgerModel, l *quantadmingo.V2KvLedgerDetail) diag.Diagnostics {
	if d.PersonalFields.IsNull() || d.PersonalFields.IsUnknown() {
		d.PersonalFields = types.ListNull(types.StringType) // gives the mapper the element type
	}
	diags := mapper.FromSDK(ctx, l, d, kvLedgerCallerHandled...)
	if d.PersonalFields.IsNull() {
		d.PersonalFields = types.ListValueMust(types.StringType, []attr.Value{})
	}
	d.LedgerName = d.Name
	d.SubjectField = optionalString(l.SubjectField)
	d.SubjectMode = optionalString(l.SubjectMode)
	d.SubjectCase = optionalString(l.SubjectCase)
	d.Error = optionalString(l.Error)
	d.CreatedAt = timeString(l.GetCreatedAt())
	d.ActiveAfter = timeString(l.GetActiveAfter())
	d.UpdatedAt = kvOptionalTime(l.UpdatedAt)
	d.DroppedAt = kvOptionalTime(l.DroppedAt)
	d.RetainUntil = kvOptionalTime(l.RetainUntil)
	anchor, more := kvLedgerAnchorValue(ctx, l.GetAnchor())
	d.Anchor = anchor
	diags.Append(more...)
	return diags
}

// kvLedgerAnchorValue reads the anchor back. An absent prefix is "", the
// schema default, so a configured anchor shows no diff.
func kvLedgerAnchorValue(ctx context.Context, a quantadmingo.V2KvLedgerAnchor) (resource_kv_ledger.AnchorValue, diag.Diagnostics) {
	prefix := types.StringValue(a.GetPrefix())
	return resource_kv_ledger.NewAnchorValue(resource_kv_ledger.AnchorValue{}.AttributeTypes(ctx), map[string]attr.Value{
		"type":       types.StringValue(a.GetType()),
		"bucket_arn": optionalString(a.BucketArn),
		"prefix":     prefix,
		"account_id": optionalString(a.AccountId),
		"tsa_url":    optionalString(a.TsaUrl),
	})
}
