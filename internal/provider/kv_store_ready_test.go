package provider_test

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/provider"
)

// shortKVStoreReadyWait shrinks the post-create wait so tests do not sleep.
func shortKVStoreReadyWait(t *testing.T, interval, timeout time.Duration) {
	t.Helper()
	prevInterval, prevTimeout := provider.KVStoreReadyInterval, provider.KVStoreReadyTimeout
	provider.KVStoreReadyInterval, provider.KVStoreReadyTimeout = interval, timeout
	t.Cleanup(func() { provider.KVStoreReadyInterval, provider.KVStoreReadyTimeout = prevInterval, prevTimeout })
}

func itemsListURL(org, project, storeId string) string {
	return fmt.Sprintf("https://dashboard.quantcdn.io/api/v2/organizations/%s/projects/%s/kv/%s/items", org, project, storeId)
}

// The API accepts a create before the DynamoDB table is active. Item writes
// fail until then, so Create must not return before the store answers.
func TestAccKVStoreResource_WaitsUntilTheStoreAnswers(t *testing.T) {
	org, project, storeId := "test-org", "test-project", "store-123"
	mockKVStoreServer(t, org, project, storeId)
	defer httpmock.DeactivateAndReset()
	shortKVStoreReadyWait(t, 10*time.Millisecond, 5*time.Second)

	calls := 0
	httpmock.RegisterResponder("GET", itemsListURL(org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			calls++
			if calls < 3 {
				// What the portal answers while the table is still creating.
				return httpmock.NewJsonResponse(400, map[string]interface{}{"error": true, "message": "Failed to fetch KV store items"})
			}
			return httpmock.NewJsonResponse(200, map[string]interface{}{"data": []interface{}{}, "next_cursor": nil})
		})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccKVStoreResourceConfig(org, project),
			Check:  resource.TestCheckResourceAttr("quant_kv_store.test", "store_id", storeId),
		}},
	})

	if calls < 3 {
		t.Fatalf("expected Create to poll the store until it answered, got %d list calls", calls)
	}
}

func TestAccKVStoreResource_ReadyTimeoutIsAnError(t *testing.T) {
	org, project, storeId := "test-org", "test-project", "store-123"
	mockKVStoreServer(t, org, project, storeId)
	defer httpmock.DeactivateAndReset()
	shortKVStoreReadyWait(t, 5*time.Millisecond, 40*time.Millisecond)

	httpmock.RegisterResponder("GET", itemsListURL(org, project, storeId),
		httpmock.NewJsonResponderOrPanic(400, map[string]interface{}{"error": true, "message": "Failed to fetch KV store items"}))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      testAccKVStoreResourceConfig(org, project),
			ExpectError: regexp.MustCompile(`KV Store Not Ready`),
		}},
	})
}

func TestAccKVStoreResource_ReadyCheckAuthFailureIsImmediate(t *testing.T) {
	org, project, storeId := "test-org", "test-project", "store-123"
	mockKVStoreServer(t, org, project, storeId)
	defer httpmock.DeactivateAndReset()
	shortKVStoreReadyWait(t, 10*time.Millisecond, 5*time.Second)

	calls := 0
	httpmock.RegisterResponder("GET", itemsListURL(org, project, storeId),
		func(req *http.Request) (*http.Response, error) {
			calls++
			return httpmock.NewJsonResponse(403, map[string]interface{}{"error": true, "message": "Forbidden"})
		})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      testAccKVStoreResourceConfig(org, project),
			ExpectError: regexp.MustCompile(`Unable to Check KV Store`),
		}},
	})
	if calls != 1 {
		t.Fatalf("expected one list call before giving up on 403, got %d", calls)
	}
}
