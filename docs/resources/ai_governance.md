---
page_title: "Quant: quant_ai_governance"
subcategory: "AI"
description: |-
  Manages the AI governance settings of an organisation: model policy, spend limits, and mandatory filters.
---

# AI Governance Resource

Manages the AI governance settings of an organisation: model policy, spend limits, and mandatory filters.

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
  model_policy = "allowlist"
  model_list   = ["example-model-id"]

  spend_limits = {
    daily_budget_cents           = 5000
    monthly_budget_cents         = 100000
    per_user_daily_budget_cents  = 500
    warning_threshold_percent    = 80
  }
}
```

## Argument Reference

### Required

* `ai_enabled` (Boolean) - (Required) Whether AI features are enabled for the organisation.
* `model_policy` (String) - (Required) Model policy for the organisation (for example, an allow list or deny list). Valid values: `unrestricted`, `allowlist`, `blocklist`.

### Optional

* `mandatory_filter_policies` (List of String) - (Optional) Content filter policy IDs applied to every request.
* `mandatory_guardrail_preset` (String) - (Optional) Guardrail preset applied to every request. Valid values: `official`, `official-sensitive`, `protected`.
* `model_list` (List of String) - (Optional) Model identifiers that the model policy applies to.
* `organisation` (String) - (Optional) The organisation ID.
* `spend_limits` (Object) - (Optional) Spend caps for the organisation, users, API tokens, and interfaces.
  * `daily_budget_cents` (Number) - (Optional) Organisation-wide daily spend cap in cents.
  * `interface_limits` (Map of Object) - (Optional) Aggregate spend caps per interface label (slack, autonomous, api-gateway, streaming, websocket, durable, orchestration, workflow, embedding, ai-search). Keys are interface labels.
    * `daily_cents` (Number) - (Optional) Daily cap in cents.
    * `monthly_cents` (Number) - (Optional) Monthly cap in cents.
  * `monthly_budget_cents` (Number) - (Optional) Organisation-wide monthly spend cap in cents.
  * `per_token_daily_budget_cents` (Number) - (Optional) Flat daily cap in cents applied to every API token without a named override.
  * `per_token_monthly_budget_cents` (Number) - (Optional) Flat monthly cap in cents applied to every API token without a named override.
  * `per_user_daily_budget_cents` (Number) - (Optional) Flat daily spend cap in cents applied to every user without an override.
  * `per_user_monthly_budget_cents` (Number) - (Optional) Flat monthly spend cap in cents applied to every user without an override.
  * `token_overrides` (Map of Object) - (Optional) Per-token budget overrides keyed by API token id. Replaces the flat per-token budget for that token; unlimited=true exempts it.
    * `daily_cents` (Number) - (Optional) Daily cap in cents.
    * `monthly_cents` (Number) - (Optional) Monthly cap in cents.
    * `unlimited` (Boolean) - (Optional) Exempt from budget caps when `true`.
  * `user_overrides` (Map of Object) - (Optional) Per-user budget overrides keyed by userId (numeric portal id, slack-<id>, or system:code-agent). Replaces the flat per-user budget for that user; unlimited=true exempts them.
    * `daily_cents` (Number) - (Optional) Daily cap in cents.
    * `monthly_cents` (Number) - (Optional) Monthly cap in cents.
    * `unlimited` (Boolean) - (Optional) Exempt from budget caps when `true`.
  * `warning_threshold_percent` (Number) - (Optional) Percentage of a budget at which a warning is raised.
* `version` (Number) - (Optional) Version number used for optimistic concurrency.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `config` (Object) - (Read-Only) Governance configuration as stored by the platform.
* `org_id` (String) - (Read-Only) The organisation ID.
* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.

## Notes

One governance configuration exists for each organisation. The `model_policy` value is `unrestricted`, `allowlist`, or `blocklist`.

## Import

Import the governance settings with the organisation name. Use an empty ID to use the provider default organisation.

```bash
terraform import quant_ai_governance.example example-org
```
