package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_cron_job"
)

// Cron run state (last_run, consecutive_failures) is read-only and changes on
// its own each time the job runs. It is copied from every API answer and is
// never unknown after apply: an absent value becomes null. The attributes have
// no plan modifiers, so a new run shows up on refresh without a plan diff.

// cronLastRunAttrs maps the API's lastRun summary to Terraform attribute values.
func cronLastRunAttrs(lr *quantadmingo.CronLastRun) map[string]attr.Value {
	return map[string]attr.Value{
		"run_id":        types.StringPointerValue(ptrStr(lr.GetRunIdOk())),
		"status":        types.StringPointerValue(ptrStr(lr.GetStatusOk())),
		"created_at":    types.StringPointerValue(ptrTimeStr(lr.GetCreatedAtOk())),
		"exit_code":     types.Int64PointerValue(ptrInt32To64(lr.GetExitCodeOk())),
		"error_message": types.StringPointerValue(ptrStr(lr.GetErrorMessageOk())),
	}
}

// ptrInt32To64 widens an optional int32 to *int64, or nil when absent.
func ptrInt32To64(v *int32, ok bool) *int64 {
	if !ok || v == nil {
		return nil
	}
	w := int64(*v)
	return &w
}

// cronConsecutiveFailures returns the failure count, or null when the API
// answer does not carry it (the single-job GET may omit it).
func cronConsecutiveFailures(cron *quantadmingo.Cron) types.Int64 {
	return types.Int64PointerValue(ptrInt32To64(cron.GetConsecutiveFailuresOk()))
}

// cronLastRunObject builds the last_run object for the given attribute types,
// or a null object when the job has not run.
func cronLastRunObject(attrTypes map[string]attr.Type, cron *quantadmingo.Cron) (types.Object, diag.Diagnostics) {
	lr, ok := cron.GetLastRunOk()
	if !ok || lr == nil {
		return types.ObjectNull(attrTypes), nil
	}
	return types.ObjectValue(attrTypes, cronLastRunAttrs(lr))
}

// setCronJobRunState copies last_run and consecutive_failures to the cron job model.
func setCronJobRunState(ctx context.Context, cron *quantadmingo.Cron, data *resource_cron_job.CronJobModel) diag.Diagnostics {
	data.ConsecutiveFailures = cronConsecutiveFailures(cron)

	lr, ok := cron.GetLastRunOk()
	if !ok || lr == nil {
		data.LastRun = resource_cron_job.NewLastRunValueNull()
		return nil
	}
	v, diags := resource_cron_job.NewLastRunValue(resource_cron_job.LastRunValue{}.AttributeTypes(ctx), cronLastRunAttrs(lr))
	data.LastRun = v
	return diags
}
