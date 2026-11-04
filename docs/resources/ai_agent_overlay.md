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
* `filter_policies` (List of String) - (Optional) Filter policy IDs applied to this agent's requests for the organisation, with any organisation-wide mandatory policies.
* `guardrail_preset` (String) - (Optional) Guardrail preset. Valid values: `official`, `official-sensitive`, `protected`.
* `long_context` (Boolean) - (Optional) Force the 1M context window on (`true`) or off (`false`) for the organisation.
* `max_tokens` (Number) - (Optional) Override max tokens. Minimum: 1.
* `mcp_servers` (List of Object) - (Optional) Registered MCP servers to attach. The agent gets the tools of each attached server in dashboard chat, agent chat and QuantCode. Each object has:
  * `server` (String) - (Required) The registered server name (lowercase letters, digits and hyphens).
  * `tools` (List of String) - (Optional) Only these tools of the server. Omit for every tool the server allows. An empty list allows none.
* `model_id` (String) - (Optional) Override the base agent's model.
* `organisation` (String) - (Optional) The organisation ID.
* `reasoning_effort` (String) - (Optional) Override the reasoning effort. Valid values: `low`, `medium`, `high`, `xhigh`, `max`, limited to the levels the effective model supports.
* `system_prompt_append` (String) - (Optional) Text appended to base system prompt.
* `temperature` (Number) - (Optional) Override temperature.
* `version` (Number) - (Optional) Current version for optimistic concurrency.

## Values you do not set

The API replaces the whole overlay on each change. To keep settings that someone made in the dashboard or the API, the provider reads the stored overlay first and sends back every value that your configuration does not set. So an argument you leave out keeps its stored value; it is not cleared.

To clear a value, set it explicitly: `[]` for a list, `""` for `model_id`, `system_prompt_append` or `guardrail_preset`. To remove the whole overlay, destroy the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `base` (Object) - (Read-Only) Base global agent metadata.
* `overlay` (Object) - (Read-Only) Overlay as returned by the API.

## Import

Import an overlay with the ID `organisation/agent_id`.

```bash
terraform import quant_ai_agent_overlay.example example-org/example-global-agent
```
