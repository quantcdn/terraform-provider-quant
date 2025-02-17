package provider_test

import (
	"github.com/jarcoal/httpmock"
	"fmt"
	"net/http"
	"testing"
)

var domainResponse = map[string]interface{}{
	"id": 1,
	"domain": "test-domain.com",
	"dns_engaged": 1,
}

func mockDomainServer(t *testing.T, organizationID string, projectID string, domainID string) {
	httpmock.Activate()
	baseUrl := "https://dashboard.quantcdn.io/api/v2"

	httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
		t.Logf("Request: %s", req.URL)
		return httpmock.NewStringResponse(404, "Not Found"), nil
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/domains", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, domainResponse)
	})

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/organizations/%s/projects/%s/domains/%s", baseUrl, organizationID, projectID, domainID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, domainResponse)
	})

	httpmock.RegisterResponder("POST", fmt.Sprintf("%s/organizations/%s/projects/%s/domains", baseUrl, organizationID, projectID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, domainResponse)
	})

	httpmock.RegisterResponder("DELETE", fmt.Sprintf("%s/organizations/%s/projects/%s/domains/%s", baseUrl, organizationID, projectID, domainID), func(req *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, domainResponse)
	})
}

func TestDomainResource(t *testing.T) {
	organizationID := "test-organization"
	projectID := "test-project"
	domainID := "test-domain"
	mockDomainServer(t, organizationID, projectID, domainID)
	defer httpmock.DeactivateAndReset()

	// resource.Test(t, resource.TestCase{
	// 	ProtoV6ProviderFactories: testDomainResourceFactories(t),
	// 	Steps: []resource.TestStep{
	// 		// Create and Read testing
	// 		{
	// 			Config: testDomainResourceConfig(organizationID, projectID, "test-domain.com"),
	// 			Check: resource.ComposeAggregateTestCheckFunc(
	// 				resource.TestCheckResourceAttr("quant_domain.test", "domain", "test-domain.com"),
	// 			),
	// 		},
	// 		// Update and Read testing
	// 		{
	// 			Config: testDomainResourceConfig(organizationID, projectID, "test-domain-updated.com"),
	// 			Check: resource.ComposeAggregateTestCheckFunc(
	// 				resource.TestCheckResourceAttr("quant_domain.test", "domain", "test-domain-updated.com"),
	// 			),
	// 		},
	// 		// Delete testing
	// 		{
	// 			Config: testDomainResourceConfig(organizationID, projectID, "test-domain-updated.com"),
	// 			Check: resource.ComposeAggregateTestCheckFunc(
	// 				resource.TestCheckResourceAttr("quant_domain.test", "domain", "test-domain-updated.com"),
	// 			),
	// 		},
	// 	},
	// })
}

// func testDomainResourceConfig(organizationID string, projectID string, domain string) string {
// 	return fmt.Sprintf(`
// 	provider "quant" {
// 		bearer = "testtoken"
// 		organization = "%s"
// 	}
// 	resource "quant_domain" "test" {
// 		project = "%s"
// 		domain = "%s"
// 	}
// 	`, organizationID, projectID, domain)
// }

// func testDomainResourceConfigUpdate(organizationID string, projectID string, domain string) string {
// 	return fmt.Sprintf(`
// 	provider "quant" {
// 		bearer = "testtoken"
// 		organization = "%s"
// 	}
// 	resource "quant_domain" "test" {
// 		project = "%s"
// 		domain = "%s"
// 	}
// 	`, organizationID, projectID, domain)
// }