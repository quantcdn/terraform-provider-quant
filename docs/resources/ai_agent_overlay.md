---
page_title: "Quant: quant_ai_agent_overlay"
subcategory: "AI"
description: |-
  Manages an organisation overlay on a platform-managed global AI agent. The overlay changes the model, tools, skills, or prompt of the global agent for the organisation only.
---

# AI Agent Overlay Resource

Manages an organisation overlay on a platform-managed global AI agent. The overlay changes the model, tools, skills, or prompt of the global agent for the organisation only.

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
resource "quant_ai_agent_overlay" "example" {
  agent_id             = "example-global-agent"
  system_prompt_append = "Always answer in plain English."
  disabled_tools       = ["example_tool"]
  max_tokens           = 4096
}
```

## Argument Reference

### Required

None.

### Optional

* `additional_skills` (List of String) - (Optional) Org-owned skill IDs to add.
* `additional_tools` (List of String) - (Optional) Tool names to add.
* `agent_id` (String) - (Optional) Global agent identifier (e.g., 'quantgov-code').
* `allowed_collections` (List of String) - (Optional) Vector DB collections.
* `disabled_skills` (List of String) - (Optional) Global skill IDs to exclude.
* `disabled_tools` (List of String) - (Optional) Tool names to remove.
* `guardrail_preset` (String) - (Optional) Guardrail preset. Valid values: `official`, `official-sensitive`, `protected`.
* `max_tokens` (Number) - (Optional) Override max tokens. Minimum: 1.
* `model_id` (String) - (Optional) Override the base agent's model.
* `organisation` (String) - (Optional) The organisation ID.
* `system_prompt_append` (String) - (Optional) Text appended to base system prompt.
* `temperature` (Number) - (Optional) Override temperature.
* `version` (Number) - (Optional) Current version for optimistic concurrency.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `base` (Object) - (Read-Only) Base global agent metadata.
* `overlay` (Object) - (Read-Only) Overlay as returned by the API.

## Import

Import an overlay with the ID `organisation/agent_id`.

```bash
terraform import quant_ai_agent_overlay.example example-org/example-global-agent
```
