package provider

import (
	"context"
	"testing"
	"time"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_cron_job"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_environment"
)

func TestSetCronJobRunState_AbsentIsNullNeverUnknown(t *testing.T) {
	data := resource_cron_job.CronJobModel{LastRun: resource_cron_job.NewLastRunValueUnknown()}
	if diags := setCronJobRunState(context.Background(), &quantadmingo.Cron{}, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.LastRun.IsNull() || data.LastRun.IsUnknown() {
		t.Errorf("last_run = %v, want null", data.LastRun)
	}
	if !data.ConsecutiveFailures.IsNull() || data.ConsecutiveFailures.IsUnknown() {
		t.Errorf("consecutive_failures = %v, want null", data.ConsecutiveFailures)
	}
}

func TestSetCronJobRunState_Present(t *testing.T) {
	cron := quantadmingo.Cron{}
	cron.SetConsecutiveFailures(2)
	lr := quantadmingo.NewCronLastRun()
	lr.SetRunId("run-1")
	lr.SetStatus("FAILED_TO_START")
	lr.SetCreatedAt(time.Date(2026, 11, 2, 1, 0, 0, 0, time.UTC))
	lr.SetErrorMessage("container missing")
	cron.SetLastRun(*lr)

	data := resource_cron_job.CronJobModel{}
	if diags := setCronJobRunState(context.Background(), &cron, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.ConsecutiveFailures.ValueInt64() != 2 {
		t.Errorf("consecutive_failures = %v, want 2", data.ConsecutiveFailures)
	}
	if data.LastRun.RunId.ValueString() != "run-1" || data.LastRun.Status.ValueString() != "FAILED_TO_START" {
		t.Errorf("last_run = %v", data.LastRun)
	}
	if data.LastRun.CreatedAt.ValueString() != "2026-11-02T01:00:00Z" {
		t.Errorf("created_at = %v", data.LastRun.CreatedAt)
	}
	if !data.LastRun.ExitCode.IsNull() || data.LastRun.ErrorMessage.ValueString() != "container missing" {
		t.Errorf("exit_code = %v, error_message = %v", data.LastRun.ExitCode, data.LastRun.ErrorMessage)
	}
}

func TestBuildEnvCronListFromSDK_RunState(t *testing.T) {
	ctx := context.Background()
	withRun := quantadmingo.Cron{}
	withRun.SetName("ran")
	withRun.SetConsecutiveFailures(0)
	lr := quantadmingo.NewCronLastRun()
	lr.SetStatus("SUCCEEDED")
	lr.SetExitCode(0)
	withRun.SetLastRun(*lr)
	never := quantadmingo.Cron{}
	never.SetName("never")

	list, diags := buildEnvCronListFromSDK(ctx, []quantadmingo.Cron{withRun, never})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var got []resource_environment.CronValue
	if d := list.ElementsAs(ctx, &got, false); d.HasError() {
		t.Fatalf("ElementsAs: %v", d)
	}
	if got[0].ConsecutiveFailures.ValueInt64() != 0 || got[0].LastRun.IsNull() {
		t.Errorf("cron[0] run state = %v / %v", got[0].ConsecutiveFailures, got[0].LastRun)
	}
	if !got[1].LastRun.IsNull() || !got[1].ConsecutiveFailures.IsNull() {
		t.Errorf("cron[1] run state = %v / %v, want null", got[1].ConsecutiveFailures, got[1].LastRun)
	}
}
