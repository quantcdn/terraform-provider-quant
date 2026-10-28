---
page_title: "Quant: quant_volume"
subcategory: "Quant Cloud"
description: |-
  Manages a persistent storage volume for a Quant Cloud environment.
---

# Volume Resource

Manages a persistent storage volume for a Quant Cloud environment.

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
resource "quant_volume" "uploads" {
  application = "example-app"
  environment = "production"
  volume_name = "uploads"
  description = "User uploads"
}
```

## Argument Reference

### Required

* `volume_name` (String) - (Required) Volume name.

### Optional

* `application` (String) - (Optional) The application ID.
* `description` (String) - (Optional) Volume description.
* `environment` (String) - (Optional) The environment ID.
* `organisation` (String) - (Optional) The organisation ID.
* `root_directory` (String) - (Optional) Root directory path.
* `volume` (String) - (Optional) The volume ID.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `access_point_arn` (String) - (Read-Only) The access point resource name of the volume.
* `access_point_id` (String) - (Read-Only) The access point ID of the volume.
* `created_at` (String) - (Read-Only) Creation timestamp.
* `environment_efs_id` (String) - (Read-Only) The ID of the shared filesystem that holds the volume.
* `volume_id` (String) - (Read-Only) The ID of the volume.

## Import

Import a volume with the ID `application/environment/volume_id`.

```bash
terraform import quant_volume.uploads example-app/production/example-volume-id
```
