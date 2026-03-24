package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

func mockKVStoreServer(t *testing.T, org string, project string, storeId string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	storeDeleted := false

	// Dynamic closure state for mock responses
	var currentName = "my-store"

	createResponse := func() map[string]interface{} {
		return map[string]interface{}{
			"id":   storeId,
			"name": currentName,
		}
	}

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create KV store
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			storeDeleted = false
			var requestBody map[string]interface{}
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
				}
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// GET KV store by ID
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s", baseUrl, org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			if storeDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			return httpmock.NewJsonResponse(200, createResponse())
		})

	// PUT KV store (update)
	httpmock.RegisterResponder("PUT",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s", baseUrl, org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			var requestBody map[string]interface{}
			if req.Body != nil {
				body, _ := io.ReadAll(req.Body)
				if err := json.Unmarshal(body, &requestBody); err == nil {
					if name, ok := requestBody["name"].(string); ok {
						currentName = name
					}
				}
			}
			return httpmock.NewJsonResponse(200, createResponse())
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
			// Note: KV store Update is not supported — the resource returns an error on Update.
			// The 'name' field lacks RequiresReplace, so changing it would attempt an in-place
			// update and fail. No Update test step is added here.
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

// ---------- KV Store Update error test ----------
// The 'name' field has no RequiresReplace, so changing it triggers Update,
// which returns an error because the KV API has no update endpoint.

func TestAccKVStoreResource_UpdateError(t *testing.T) {
	org := "test-org"
	project := "test-project"
	storeId := "store-123"
	mockKVStoreServer(t, org, project, storeId)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccKVStoreResourceConfig(org, project),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_kv_store.test", "name", "my-store"),
				),
			},
			// Step 2: Change name to trigger Update -> expect error
			{
				Config:      testAccKVStoreResourceConfigWithName(org, project, "renamed-store"),
				ExpectError: regexp.MustCompile(`Update Not Supported`),
			},
		},
	})
}

func testAccKVStoreResourceConfigWithName(org string, project string, name string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_kv_store" "test" {
  organization = %[1]q
  project      = %[2]q
  name         = %[3]q
}
`, org, project, name)
}

// ---------- KV Store HTTP error-path tests ----------

func setupKVStoreErrorResponder(t *testing.T, org string, project string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testKVStoreErrorConfig() string {
	return `
provider "quant" {
  organization = "test-org"
  bearer = "testtoken"
}

resource "quant_kv_store" "test" {
  organization = "test-org"
  project      = "test-project"
  name         = "error-test"
}
`
}

func TestAccKVStoreResource_CreateAuth401(t *testing.T) {
	setupKVStoreErrorResponder(t, "test-org", "test-project", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVStoreErrorConfig(),
				ExpectError: regexp.MustCompile(`Authentication Failed`),
			},
		},
	})
}

func TestAccKVStoreResource_CreateForbidden403(t *testing.T) {
	setupKVStoreErrorResponder(t, "test-org", "test-project", 403, map[string]interface{}{
		"error":   true,
		"message": "Insufficient permissions",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVStoreErrorConfig(),
				ExpectError: regexp.MustCompile(`Authorization Failed`),
			},
		},
	})
}

func TestAccKVStoreResource_CreateBadRequest400(t *testing.T) {
	setupKVStoreErrorResponder(t, "test-org", "test-project", 400, map[string]interface{}{
		"error":   true,
		"message": "Invalid store configuration",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVStoreErrorConfig(),
				ExpectError: regexp.MustCompile(`Invalid KV Store Configuration`),
			},
		},
	})
}

func TestAccKVStoreResource_CreateServerError500(t *testing.T) {
	setupKVStoreErrorResponder(t, "test-org", "test-project", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVStoreErrorConfig(),
				ExpectError: regexp.MustCompile(`Unable to Create KV Store`),
			},
		},
	})
}
