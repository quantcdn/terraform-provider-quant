package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"terraform-provider-quant/internal/provider"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jarcoal/httpmock"
)

var domainResponse = map[string]interface{}{
	"id":              9555,
	"domain":          "example.com",
	"dns_engaged":     0,
	"in_section":      0,
	"project_id":      1,
	"section_message": "",
	"created_at":      "2024-01-01T00:00:00Z",
	"updated_at":      "2024-01-01T00:00:00Z",
	"deleted_at":      "",
}

func mockDomainServer(t *testing.T, organizationID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	// List domains
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/default/domains", baseUrl, organizationID),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(200, []interface{}{domainResponse})
		})

	// Get domain by ID
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/default/domains/%d", baseUrl, organizationID, domainResponse["id"]),
		func(req *http.Request) (*http.Response, error) {
			// Extract domain ID from URL
			parts := strings.Split(req.URL.Path, "/")
			domainID := parts[len(parts)-1]
			if domainID == "9555" {
				return httpmock.NewJsonResponse(200, domainResponse)
			}
			return httpmock.NewStringResponse(404, "Not Found"), nil
		})

	// Create domain
	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/default/domains", baseUrl, organizationID),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(201, domainResponse)
		})

	// Update domain
	httpmock.RegisterResponder("PATCH", fmt.Sprintf("%s/organizations/%s/projects/default/domains/%d", baseUrl, organizationID, domainResponse["id"]),
		func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return httpmock.NewStringResponse(400, "Failed to read request body"), nil
			}

			var requestBody struct {
				Name string `json:"name"`
			}

			if err := json.Unmarshal(body, &requestBody); err != nil {
				return httpmock.NewStringResponse(400, "Invalid JSON"), nil
			}

			response := make(map[string]interface{})
			for k, v := range domainResponse {
				response[k] = v
			}
			return httpmock.NewJsonResponse(200, response)
		})

	// Delete domain
	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/default/domains/%d", baseUrl, organizationID, domainResponse["id"]),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(204, ""), nil
		})
}

func testDomainResourceFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"quant": providerserver.NewProtocol6WithError(provider.New()()),
	}
}

func TestDomainResource(t *testing.T) {
	organizationID := "test-organization"
	mockDomainServer(t, organizationID)
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testDomainResourceFactories(t),
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testDomainResourceConfig(organizationID, "test-domain", "example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_domain.test", "domain", "example.com"),
					resource.TestCheckResourceAttr("quant_domain.test", "dns_engaged", "0"),
				),
			},
			// Import testing
			{
				ResourceName:      "quant_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "default/9555",
			},
			// Update and Read testing
			{
				Config: testDomainResourceConfig(organizationID, "test-domain-updated", "example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("quant_domain.test", "domain", "example.com"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testDomainResourceConfig(organization string, name string, domain string) string {
	return fmt.Sprintf(`
provider "quant" {
	organization = %[1]q
	bearer = "testtoken"
}

resource "quant_domain" "test" {
  domain = %[3]q
  project = "default"
}
`, organization, name, domain)
}

// ---------- Domain HTTP error-path tests ----------

func setupDomainErrorResponder(t *testing.T, org string, project string, statusCode int, body map[string]interface{}) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Unhandled Request: %s %s", req.Method, req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("%s/organizations/%s/projects/%s/domains", baseUrl, org, project),
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(statusCode, body)
		})
}

func testDomainErrorConfig() string {
	return `
provider "quant" {
  organization = "test-organization"
  bearer = "testtoken"
}

resource "quant_domain" "test" {
  domain  = "error-test.example.com"
  project = "default"
}
`
}

func TestDomainResource_CreateError(t *testing.T) {
	setupDomainErrorResponder(t, "test-organization", "default", 500, map[string]interface{}{
		"error":   true,
		"message": "Internal server error",
	})
	defer httpmock.DeactivateAndReset()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testDomainResourceFactories(t),
		Steps: []resource.TestStep{
			{
				Config:      testDomainErrorConfig(),
				ExpectError: regexp.MustCompile(`Error creating domain`),
			},
		},
	})
}
