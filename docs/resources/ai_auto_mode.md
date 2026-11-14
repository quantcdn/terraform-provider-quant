---
page_title: "Quant: quant_ai_auto_mode"
subcategory: "AI"
description: |-
  Manages the auto-mode setting of an organisation: the orchestrator model, the model roles, the watcher, and enforcement.
---

# AI Auto Mode Resource

Manages the auto-mode setting of an organisation: the orchestrator model, the model roles, the watcher, and enforcement. One setting exists for each organisation.

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
resource "quant_ai_governance" "example" {
  ai_enabled   = true
  model_policy = "unrestricted"
}

resource "quant_ai_auto_mode" "example" {
  enabled                  = true
  orchestrator_model       = "example-orchestrator-model"
  watcher_interval_minutes = 5
  max_parallel_tasks       = 4
  enforcement              = "default" # QuantGov accepts "required" only.
  autonomous_default       = false

  group = {
    roles = {
      fast     = ["example-fast-model"]
      standard = ["example-standard-model", "example-fallback-model"]
      deep     = ["example-deep-model"]
    }
  }

  # The API refuses auto mode until the organisation has governance settings.
  depends_on = [quant_ai_governance.example]
}
```

## Argument Reference

### Required

* `enabled` (Boolean) - (Required) Whether auto mode is on for the organisation.
* `watcher_interval_minutes` (Number) - (Required) Minutes between watcher checks, 0 to 60. The Portal default is 5.
* `enforcement` (String) - (Required) Valid values: `default`, `required`. QuantGov accepts `required` only.
* `autonomous_default` (Boolean) - (Required) Whether autonomous runs use auto mode by default. The Portal default is `false`.
* `group` (Object) - (Required) The model group.
  * `roles` (Object) - (Required) The models of each role, in order of preference. Terraform keeps the order.
    * `fast` (List of String) - (Required) Up to 5 model IDs. Use `[]` for none.
    * `standard` (List of String) - (Required) Up to 5 model IDs. Use `[]` for none.
    * `deep` (List of String) - (Required) Up to 5 model IDs. Use `[]` for none.

### Optional

* `orchestrator_model` (String) - (Optional) The orchestrator model ID. Required when `enabled` is `true`.
* `max_parallel_tasks` (Number) - (Optional) Maximum parallel tasks, 1 to 16. When unset, the provider does not send it and the platform default applies.
* `organisation` (String) - (Optional) The organisation machine name. Defaults to the provider organisation.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `group.revision` (Number) - (Read-Only) Revision of the setting. The provider sends it on update; a stale value returns 409.
* `updated_at` (String) - (Read-Only) When the setting was last saved.
* `updated_by` (String) - (Read-Only) Who last saved the setting.

## Notes

* Apply `quant_ai_governance` for the organisation first. Without governance settings, the API returns "Save the organisation governance settings first". Add `depends_on` when both resources are in one configuration.
* Every model must be eligible under the governance settings. If a model is not eligible, the apply fails and lists each ineligible model with the reason.
* If the setting changes in the Portal, the next plan shows the change and apply restores the configuration.
* If the setting is deleted in the Portal, the next plan creates it again.
* `terraform destroy` deletes the setting. The organisation then has no auto-mode setting.
* If the organisation already has a setting, create fails. Import the setting instead.
* `orchestrator_model` and `max_parallel_tasks` keep their last value when you remove them from the configuration. The provider continues to send that value. To change the value, set a new one in the configuration.

## Import

Import the setting with the organisation machine name. The import fails if the organisation has no setting.

```bash
terraform import quant_ai_auto_mode.example example-org
```
