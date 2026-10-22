---
page_title: "Quant: quant_ai_vector_collection"
subcategory: "AI"
description: |-
  Manages a vector collection for AI semantic search and retrieval.
---

# AI Vector Collection Resource

Manages a vector collection for AI semantic search and retrieval.

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
resource "quant_ai_vector_collection" "docs" {
  name            = "example-docs"
  description     = "Product documentation"
  embedding_model = "amazon.titan-embed-text-v2:0"
  dimensions      = 1024
}
```

## Argument Reference

### Required

* `embedding_model` (String) - (Required) Embedding model to use. Supported: amazon.titan-embed-text-v2:0, cohere.embed-english-v3, cohere.embed-multilingual-v3.
* `name` (String) - (Required) Collection name (used for reference).

### Optional

* `collection_id` (String) - (Optional) The collection ID.
* `description` (String) - (Optional) Free-text description.
* `dimensions` (Number) - (Optional) Embedding dimensions (default: 1024).
* `organisation` (String) - (Optional) The organisation ID.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `collection` (Object) - (Read-Only) Collection details as returned by the API.
* `message` (String) - (Read-Only) Status message returned by the last API operation.
* `success` (Boolean) - (Read-Only) Whether the last API operation succeeded.

## Import

Import a collection with the collection ID, or with `organisation/collection_id`.

```bash
terraform import quant_ai_vector_collection.docs example-org/example-collection-id
```
