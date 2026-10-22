---
page_title: "Quant: quant_ai_agent"
subcategory: "AI"
description: |-
  Manages a custom AI agent with its own model, system prompt, tools, and skills.
---

# AI Agent Resource

Manages a custom AI agent with its own model, system prompt, tools, and skills.

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
resource "quant_ai_agent" "support" {
  name          = "Support agent"
  description   = "Answers customer questions."
  model_id      = "example-model-id"
  system_prompt = "You are a helpful support agent."

  allowed_tools       = ["example_tool"]
  allowed_collections = ["example-collection-id"]
  max_tokens          = 4096
}
```

## Argument Reference

### Required

* `description` (String) - (Required) Free-text description.
* `model_id` (String) - (Required) The model identifier.
* `name` (String) - (Required) Name.
* `system_prompt` (String) - (Required) System prompt for the agent.

### Optional

* `agent_id` (String) - (Optional) The agent ID.
* `allowed_collections` (List of String) - (Optional) Vector collections the agent may query.
* `allowed_tools` (List of String) - (Optional) Tool names the agent may use.
* `assigned_skills` (List of String) - (Optional) Skill IDs to assign to this agent.
* `created_by` (String) - (Optional) User identifier who created the agent.
* `filter_policies` (List of String) - (Optional) Filter policy IDs to apply to this agent's inference requests.
* `group` (String) - (Optional) Group label for the agent.
* `guardrail_preset` (String) - (Optional) Guardrail preset name. Valid values: `official`, `official-sensitive`, `protected`.
* `long_context` (Boolean) - (Optional) Enable 1M context window support.
* `max_tokens` (Number) - (Optional) Maximum number of response tokens. Range: 1 to 200000.
* `organisation` (String) - (Optional) The organisation ID.
* `temperature` (Number) - (Optional) Sampling temperature.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `agent` (Object) - (Read-Only) Agent details as returned by the API.
* `message` (String) - (Read-Only) Status message returned by the last API operation.
* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.

## Import

Import an agent with the agent ID.

```bash
terraform import quant_ai_agent.support example-agent-id
```
