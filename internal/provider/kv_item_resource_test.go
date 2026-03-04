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

func mockKVItemServer(t *testing.T, org string, project string, storeId string, key string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	itemDeleted := false
	currentValue := "myval"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// POST create KV item
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s/items", baseUrl, org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			itemDeleted = false

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			var requestBody map[string]interface{}
			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			if v, ok := requestBody["value"]; ok {
				currentValue = fmt.Sprintf("%v", v)
			}

			resp := map[string]interface{}{
				"success": true,
				"key":     key,
				"value":   currentValue,
			}
			return httpmock.NewJsonResponse(200, resp)
		})

	// GET KV item
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s/items/%s", baseUrl, org, project, storeId, key),
		func(req *http.Request) (*http.Response, error) {
			if itemDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			resp := map[string]interface{}{
				"key":   key,
				"value": currentValue,
			}
			return httpmock.NewJsonResponse(200, resp)
		})

	// PUT update KV item
	httpmock.RegisterResponder("PUT",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s/items/%s", baseUrl, org, project, storeId, key),
		func(req *http.Request) (*http.Response, error) {
			if itemDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			var requestBody map[string]interface{}
			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			if v, ok := requestBody["value"]; ok {
				currentValue = fmt.Sprintf("%v", v)
			}

			resp := map[string]interface{}{
				"success": true,
				"key":     key,
				"value":   currentValue,
			}
			return httpmock.NewJsonResponse(200, resp)
		})

	// DELETE KV item
	httpmock.RegisterResponder("DELETE",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s/items/%s", baseUrl, org, project, storeId, key),
		func(req *http.Request) (*http.Response, error) {
			if itemDeleted {
				return httpmock.NewStringResponse(404, "Not Found"), nil
			}
			itemDeleted = true
			resp := map[string]interface{}{
				"success": true,
			}
			return httpmock.NewJsonResponse(200, resp)
		})
}

func TestAccKVItemResource(t *testing.T) {
	org := "test-org"
	project := "test-project"
	storeId := "store-123"
	key := "mykey"
	mockKVItemServer(t, org, project, storeId, key)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccKVItemResourceConfig(org, project, storeId, key, "myval"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_kv_item.test", "key", "mykey"),
					resource.TestCheckResourceAttr("quant_kv_item.test", "value", "myval"),
					resource.TestCheckResourceAttr("quant_kv_item.test", "store_id", "store-123"),
					resource.TestCheckResourceAttr("quant_kv_item.test", "project", "test-project"),
					resource.TestCheckResourceAttr("quant_kv_item.test", "secret", "false"),
				),
			},
			// Import testing
			{
				ResourceName:  "quant_kv_item.test",
				ImportState:   true,
				ImportStateId: fmt.Sprintf("%s/%s/%s", project, storeId, key),
				ImportStateVerifyIgnore: []string{
					"organization",
					"value",
				},
			},
			// Update and Read testing
			{
				Config: testAccKVItemResourceConfig(org, project, storeId, key, "updated-val"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_kv_item.test", "key", "mykey"),
					resource.TestCheckResourceAttr("quant_kv_item.test", "value", "updated-val"),
				),
			},
		},
	})
}

func testAccKVItemResourceConfig(org string, project string, storeId string, key string, value string) string {
	return fmt.Sprintf(`
provider "quant" {
  organization = %[1]q
  bearer = "testtoken"
}

resource "quant_kv_item" "test" {
  organization = %[1]q
  project      = %[2]q
  store_id     = %[3]q
  key          = %[4]q
  value        = %[5]q
}
`, org, project, storeId, key, value)
}

// ---------- KV Item HTTP error-path tests ----------

func setupKVItemErrorResponder(t *testing.T, org string, project string, storeId string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/kv/%s/items", baseUrl, org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testKVItemErrorConfig() string {
	return `
provider "quant" {
  organization = "test-org"
  bearer = "testtoken"
}

resource "quant_kv_item" "test" {
  organization = "test-org"
  project      = "test-project"
  store_id     = "store-123"
  key          = "error-key"
  value        = "error-val"
}
`
}

func TestAccKVItemResource_CreateAuth401(t *testing.T) {
	setupKVItemErrorResponder(t, "test-org", "test-project", "store-123", 401, map[string]interface{}{
		"error":   true,
		"message": "Invalid API token",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVItemErrorConfig(),
				ExpectError: regexp.MustCompile(`Authentication Failed`),
			},
		},
	})
}

func TestAccKVItemResource_CreateForbidden403(t *testing.T) {
	setupKVItemErrorResponder(t, "test-org", "test-project", "store-123", 403, map[string]interface{}{
		"error":   true,
		"message": "Insufficient permissions",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVItemErrorConfig(),
				ExpectError: regexp.MustCompile(`Authorization Failed`),
			},
		},
	})
}

func TestAccKVItemResource_CreateBadRequest400(t *testing.T) {
	setupKVItemErrorResponder(t, "test-org", "test-project", "store-123", 400, map[string]interface{}{
		"error":   true,
		"message": "Invalid item configuration",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVItemErrorConfig(),
				ExpectError: regexp.MustCompile(`Invalid KV Item Configuration`),
			},
		},
	})
}

func TestAccKVItemResource_CreateServerError500(t *testing.T) {
	setupKVItemErrorResponder(t, "test-org", "test-project", "store-123", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testKVItemErrorConfig(),
				ExpectError: regexp.MustCompile(`Unable to Create KV Item`),
			},
		},
	})
}
