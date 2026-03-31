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

func mockAiVectorCollectionServer(t *testing.T, org string, collectionId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io"

	collectionDeleted := false
	currentName := ""
	currentDescription := ""

	collectionsURL := fmt.Sprintf("%s/api/v3/organisations/%s/ai/vector-db/collections", baseUrl, org)
	collectionURL := fmt.Sprintf("%s/%s", collectionsURL, collectionId)

	createResponse := func() map[string]interface{} {
		col := map[string]interface{}{
			"collectionId": collectionId,
			"name":         currentName,
			"createdAt":    "2026-03-30T10:00:00Z",
		}
		if currentDescription != "" {
			col["description"] = currentDescription
		}
		return map[string]interface{}{
			"collection": col,
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create collection
	httpmock.RegisterResponder("POST", collectionsURL,
		func(req *http.Request) (*http.Response, error) {
			collectionDeleted = false
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				var requestBody map[string]interface{}
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
					if desc, ok := requestBody["description"].(string); ok {
						currentDescription = desc
					}
				}
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// GET collection by ID
	httpmock.RegisterResponder("GET", collectionURL,
		func(req *http.Request) (*http.Response, error) {
			if collectionDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// DELETE collection
	httpmock.RegisterResponder("DELETE", collectionURL,
		func(req *http.Request) (*http.Response, error) {
			if collectionDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			collectionDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccAiVectorCollectionResource(t *testing.T) {
	org := "test-org"
	collectionId := "col-uuid-123"
	mockAiVectorCollectionServer(t, org, collectionId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccAiVectorCollectionConfig(org, "my-collection", "A test collection"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_ai_vector_collection.test", "name", "my-collection"),
					resource.TestCheckResourceAttr("quant_ai_vector_collection.test", "description", "A test collection"),
					resource.TestCheckResourceAttr("quant_ai_vector_collection.test", "id", collectionId),
					resource.TestCheckResourceAttr("quant_ai_vector_collection.test", "organization", org),
					resource.TestCheckResourceAttrSet("quant_ai_vector_collection.test", "created_at"),
				),
			},
			// Step 2: Import
			{
				ResourceName:  "quant_ai_vector_collection.test",
				ImportState:   true,
				ImportStateId: collectionId,
				ImportStateVerifyIgnore: []string{
					"organization",
				},
			},
		},
	})
}

func testAccAiVectorCollectionConfig(org string, name string, description string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer       = "testtoken"
}

resource "quant_ai_vector_collection" "test" {
  organization = %[1]q
  name         = %[2]q
  description  = %[3]q
}
`, org, name, description)
}
