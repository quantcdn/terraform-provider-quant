---
page_title: "Quant: quant_ai_skill"
subcategory: "AI"
description: |-
  Manages an AI skill. A skill is a set of instructions that an AI agent can use.
---

# AI Skill Resource

Manages an AI skill. A skill is a set of instructions that an AI agent can use.

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
resource "quant_ai_skill" "example" {
  name              = "example-skill"
  description       = "Explains the returns policy."
  trigger_condition = "The user asks about returns."
  content           = <<-EOT
    # Returns policy
    Customers can return items within 30 days.
  EOT
  tags = ["support"]
}
```

## Argument Reference

### Required

* `content` (String) - (Required) Text content.
* `name` (String) - (Required) Name.
* `trigger_condition` (String) - (Required) Condition that tells the agent when to use the skill.

### Optional

* `allowed_tools` (List of String) - (Optional) Tool names the agent may use.
* `description` (String) - (Optional) Free-text description.
* `disable_model_invocation` (Boolean) - (Optional) When `true`, the model cannot invoke the skill by itself.
* `files` (Object) - (Optional) Supporting files for the skill. The provider schema defines no sub-attributes.
* `installed_by` (String) - (Optional) Identifier of the user or tool that installed the skill.
* `namespace` (String) - (Optional) Namespace of the skill.
* `organisation` (String) - (Optional) The organisation ID.
* `required_tools` (List of String) - (Optional) Tool names the skill requires.
* `skill_id` (String) - (Optional) The skill ID.
* `tags` (List of String) - (Optional) List of tags.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `message` (String) - (Read-Only) Status message returned by the last API operation.
* `skill` (Object) - (Read-Only) Skill details as returned by the API.
* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.

## Import

Import a skill with the skill ID.

```bash
terraform import quant_ai_skill.example example-skill-id
```
