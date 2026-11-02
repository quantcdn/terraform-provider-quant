---
page_title: "Quant: quant_cron_job"
subcategory: "Quant Cloud"
description: |-
  Manages a scheduled cron job for a Quant Cloud environment.
---

# Cron Job Resource

Manages a scheduled cron job for a Quant Cloud environment.

## Example Usage

```hcl
provider "quant" {
  bearer       = var.quant_api_token
  organization = "example-org"
  # AU Government customers: uncomment to use the QuantGov dashboard.
  # base_url   = "https://dash.quantgov.cloud"
}
```

```hcl
resource "quant_cron_job" "nightly" {
  application = "example-app"
  environment = "production"
  name        = "nightly-cleanup"
  description = "Remove expired sessions"

  schedule_expression   = "cron(0 2 * * ? *)"
  command               = ["php", "artisan", "sessions:prune"]
  target_container_name = "web"
  is_enabled            = true
}
```

## Argument Reference

### Required

* `command` (List of String) - (Required) Command to run, as a list of arguments.
* `name` (String) - (Required) Name.
* `schedule_expression` (String) - (Required) Schedule expression for the job, for example `cron(0 2 * * ? *)`.

### Optional

* `application` (String) - (Optional) The application ID.
* `cron` (String) - (Optional) The cron job ID.
* `description` (String) - (Optional) Free-text description.
* `environment` (String) - (Optional) The environment ID.
* `is_enabled` (Boolean) - (Optional) Whether the cron job is enabled. Defaults to `true`.
* `organisation` (String) - (Optional) The organisation ID.
* `target_container_name` (String) - (Optional) Name of the container that runs the command.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `consecutive_failures` (Number) - (Read-Only) Failed runs (FAILED or FAILED_TO_START) from the newest run back to the first run that did not fail, counted over the latest 10 runs. `0` when the newest run did not fail. Null when the API does not return it.
* `last_run` (Object) - (Read-Only) Summary of the newest run of the cron job. Null when the job has not run.
  * `run_id` (String) - (Read-Only) Run ID.
  * `status` (String) - (Read-Only) Run status. `FAILED_TO_START` means the task could not be launched, so the command never ran.
  * `created_at` (String) - (Read-Only) Start time of the run (RFC 3339).
  * `exit_code` (Number) - (Read-Only) Exit code of the command.
  * `error_message` (String) - (Read-Only) Why the run failed. Set when the status is `FAILED_TO_START`.

`last_run` and `consecutive_failures` change each time the job runs. Terraform refreshes them on every read; they never cause a plan difference.

## Import

Import a cron job with the ID `application/environment/cron_name`.

```bash
terraform import quant_cron_job.nightly example-app/production/nightly-cleanup
```
