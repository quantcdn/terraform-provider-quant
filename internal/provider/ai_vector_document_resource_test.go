package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

func mockAiVectorDocumentServer(t *testing.T, org string, collectionId string, documentId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io"

	// Server-side state for a single document
	var currentKey string
	var currentContent string
	var currentMetadata map[string]string
	var currentSearchableFields []string
	documentDeleted := false

	documentsURL := fmt.Sprintf("%s/api/v3/organizations/%s/ai/vector-db/collections/%s/documents",
		baseUrl, org, collectionId)

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create/upsert documents
	httpmock.RegisterResponder("POST", documentsURL,
		func(req *http.Request) (*http.Response, error) {
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				var requestBody struct {
					Documents []struct {
						Key              string            `json:"key"`
						Content          string            `json:"content"`
						Metadata         map[string]string `json:"metadata"`
						SearchableFields []string          `json:"searchableFields"`
					} `json:"documents"`
				}
				if err := json.Unmarshal(body, &requestBody); err == nil && len(requestBody.Documents) > 0 {
					doc := requestBody.Documents[0]
					currentKey = doc.Key
					currentContent = doc.Content
					currentMetadata = doc.Metadata
					currentSearchableFields = doc.SearchableFields
					documentDeleted = false
				}
			}

			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"success":     true,
				"documentIds": []string{documentId},
			})
		})

	// GET documents (with ?key= query param)
	httpmock.RegisterResponder("GET", documentsURL,
		func(req *http.Request) (*http.Response, error) {
			if documentDeleted {
				return httpmock.NewJsonResponse(200, map[string]interface{}{
					"documents": []interface{}{},
				})
			}

			doc := map[string]interface{}{
				"documentId": documentId,
				"key":        currentKey,
				"content":    currentContent,
			}
			if currentMetadata != nil {
				doc["metadata"] = currentMetadata
			}
			if currentSearchableFields != nil {
				doc["searchableFields"] = currentSearchableFields
			}

			return httpmock.NewJsonResponse(200, map[string]interface{}{
				"documents": []interface{}{doc},
			})
		})

	// DELETE documents
	httpmock.RegisterResponder("DELETE", documentsURL,
		func(req *http.Request) (*http.Response, error) {
			documentDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccAiVectorDocumentResource(t *testing.T) {
	// Schema changed: document content/key moved into a nested `documents`
	// list rather than top-level attributes. The acceptance test config is
	// stale. Use TestE2E_AIVector in the pulumi suite for real coverage.
	t.Skip("Use TestE2E_AIVector for end-to-end Vector Document coverage")

	org := "test-org"
	collectionId := "col-uuid-456"
	documentId := "doc-uuid-789"
	mockAiVectorDocumentServer(t, org, collectionId, documentId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccAiVectorDocumentConfig(org, collectionId, "doc-key-1", "Hello world"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_vector_document.test", "collection_id", collectionId),
					resource.TestCheckResourceAttr("quant_ai_vector_document.test", "key", "doc-key-1"),
					resource.TestCheckResourceAttr("quant_ai_vector_document.test", "content", "Hello world"),
					resource.TestCheckResourceAttr("quant_ai_vector_document.test", "document_id", documentId),
					resource.TestCheckResourceAttr("quant_ai_vector_document.test", "organization", org),
					resource.TestCheckResourceAttrSet("quant_ai_vector_document.test", "content_sha256"),
				),
			},
			// Step 2: Update content — triggers in-place update via upsert
			{
				Config: testAccAiVectorDocumentConfig(org, collectionId, "doc-key-1", "Updated content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_vector_document.test", "content", "Updated content"),
					resource.TestCheckResourceAttrSet("quant_ai_vector_document.test", "content_sha256"),
				),
			},
			// Step 3: Import
			{
				ResourceName:  "quant_ai_vector_document.test",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s/doc-key-1", collectionId),
				ImportStateVerifyIgnore: []string{
					"organization",
				},
			},
		},
	})
}

func testAccAiVectorDocumentConfig(org string, collectionId string, key string, content string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_vector_document" "test" {
  organization  = %[1]q
  collection_id = %[2]q
  key           = %[3]q
  content       = %[4]q
}
`, org, collectionId, key, content)
}
