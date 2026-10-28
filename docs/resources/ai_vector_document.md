---
page_title: "Quant: quant_ai_vector_document"
subcategory: "AI"
description: |-
  Manages documents in a Quant AI vector collection. The platform splits each document into chunks and creates embeddings.
---

# AI Vector Document Resource

Manages documents in a Quant AI vector collection. The platform splits each document into chunks and creates embeddings.

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
resource "quant_ai_vector_document" "faq" {
  collection_id = quant_ai_vector_collection.docs.collection_id

  documents = [
    {
      key     = "faq-returns"
      content = "You can return items within 30 days."
      metadata = {
        title      = "Returns FAQ"
        source_url = "https://example.com/faq/returns"
        tags       = ["faq"]
      }
    }
  ]
}
```

## Argument Reference

### Required

* `documents` (List of Object) - (Required) Documents to store in the collection.
  * `content` (String) - (Required) Document text content.
  * `key` (String) - (Optional) Stable document key for upsert.
  * `metadata` (Object) - (Optional) Metadata for the document.
    * `section` (String) - (Optional) Section label for the document.
    * `source_url` (String) - (Optional) Source URL of the document.
    * `tags` (List of String) - (Optional) List of tags.
    * `title` (String) - (Optional) Title of the document.

### Optional

* `collection_id` (String) - (Optional) The vector collection ID.
* `key` (String) - (Optional) Filter by document key.
* `limit` (Number) - (Optional) Maximum number of documents returned when reading. Maximum: 100. Defaults to `50`.
* `offset` (Number) - (Optional) Offset used when reading documents. Defaults to `0`.
* `organisation` (String) - (Optional) The organisation machine name. Defaults to the provider `organization` setting.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `chunks_created` (Number) - (Read-Only) Number of chunks created from the submitted documents.
* `document_ids` (List of String) - (Read-Only) IDs of the stored documents.
* `message` (String) - (Read-Only) Status message returned by the last API operation.
* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.

## Import

Import with the collection ID, or with `collection_id/key` to filter on one document key.

```bash
terraform import quant_ai_vector_document.faq example-collection-id/faq-returns
```
