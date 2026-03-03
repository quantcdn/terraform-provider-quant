package provider_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var kvStoreResponse = map[string]interface{}{
	"id":   "store-123",
	"name": "my-store",
}

func mockKVStoreServer(t *testing.T, org string, project string, storeId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	storeDeleted := false

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create KV store
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			storeDeleted = false
			return httpmock.NewJsonResponse(200, kvStoreResponse)
		})

	// GET KV store by ID
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s", baseUrl, org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			if storeDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, kvStoreResponse)
		})

	// DELETE KV store
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s", baseUrl, org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			if storeDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			storeDeleted = true
			return httpmock.NewStringResponse(200, ""), nil
		})
}

func TestAccKVStoreResource(t *testing.T) {
	org := "test-org"
	project := "test-project"
	storeId := "store-123"
	mockKVStoreServer(t, org, project, storeId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKVStoreResourceConfig(org, project),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_kv_store.test", "name", "my-store"),
					resource.TestCheckResourceAttr("quant_kv_store.test", "store_id", "store-123"),
					resource.TestCheckResourceAttr("quant_kv_store.test", "project", "test-project"),
				),
			},
			// Import testing
			{
				ResourceName:  "quant_kv_store.test",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s/%s", project, storeId),
				ImportStateVerifyIgnore: []string{
					"organization",
				},
			},
		},
	})
}

func testAccKVStoreResourceConfig(org string, project string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_kv_store" "test" {
  organization = %[1]q
  project      = %[2]q
  name         = "my-store"
}
`, org, project)
}
