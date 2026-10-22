---
page_title: "Quant: quant_slack_bot"
subcategory: "AI"
description: |-
  Manages a Slack bot that is backed by a Quant AI agent.
---

# Slack Bot Resource

Manages a Slack bot that is backed by a Quant AI agent.

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
resource "quant_slack_bot" "support" {
  name          = "Support bot"
  model_id      = "example-model-id"
  system_prompt = "You answer questions from the support team."
  setup_type    = "quant"

  allowed_channels = ["C0123456789"]
  max_tokens       = 2048
}
```

## Argument Reference

### Required

* `model_id` (String) - (Required) AI model identifier.
* `name` (String) - (Required) Display name for the bot.
* `setup_type` (String) - (Required) Whether to use Quant-managed or customer-provided Slack app. Valid values: `quant`, `byo`.
* `system_prompt` (String) - (Required) System prompt for the backing AI agent.

### Optional

* `agent_access_control` (Object) - (Optional) Agent-level access control settings. The provider schema defines no sub-attributes.
* `allow_guests` (Boolean) - (Optional) Whether guest users may interact with the bot.
* `allowed_channels` (List of String) - (Optional) Slack channel IDs the bot may respond in.
* `allowed_collections` (List of String) - (Optional) Vector DB collections the agent may query.
* `allowed_sub_agents` (List of String) - (Optional) Sub-agents the agent may call.
* `allowed_tools` (List of String) - (Optional) Tools the agent may use.
* `allowed_users` (List of String) - (Optional) Slack user IDs allowed to interact with the bot.
* `assigned_skills` (List of String) - (Optional) Skills assigned to the agent.
* `bot_id` (String) - (Optional) The Slack bot ID.
* `denied_users` (List of String) - (Optional) Slack user IDs denied from interacting with the bot.
* `filter_policies` (List of String) - (Optional) Content filter policies.
* `guardrail_preset` (String) - (Optional) Guardrail preset name.
* `home_tab_content` (String) - (Optional) Content shown on the bot's Home tab in Slack.
* `keywords` (List of String) - (Optional) Keywords that trigger the bot.
* `keywords_enabled` (Boolean) - (Optional) Whether keyword triggers are enabled.
* `long_context` (Boolean) - (Optional) Enable long context mode.
* `max_tokens` (Number) - (Optional) Maximum response tokens. Minimum: 1.
* `organisation` (String) - (Optional) The organisation ID.
* `session_ttl_days` (Number) - (Optional) Session TTL in days. Range: 1 to 90.
* `temperature` (Number) - (Optional) Sampling temperature.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `bot` (Object) - (Read-Only) Bot details as returned by the API.

## Import

Import a bot with the bot ID.

```bash
terraform import quant_slack_bot.support example-bot-id
```
