# Domain Resource

The `quant_domain` resource manages domains within a Quant project.

## Example Usage

```hcl
provider "quant" {
  organization = "test-organization"
  bearer       = "testtoken"
}

resource "quant_domain" "test" {
  domain  = "example.com"
  project = "default"
}
```

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The domain name to manage.
* `project` - (Optional) The project machine name to create the domain in. Defaults to "default".

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the domain.
* `created_at` - The timestamp when the domain was created.
* `updated_at` - The timestamp when the domain was last updated.
* `deleted_at` - The timestamp when the domain was deleted, if applicable.
* `dns_engaged` - Whether DNS is engaged for this domain.
* `in_section` - Whether the domain is in a section.
* `project_id` - The ID of the project the domain belongs to.
* `section_message` - Any section-related message for the domain.
* `organization` - The organization the domain belongs to.

## Import

Domains can be imported using the project and domain ID in the format `project/domain_id`:

```bash
$ terraform import quant_domain.test default/9555
``` 