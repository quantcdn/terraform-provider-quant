package provider_test

import (
	"context"
	"os"
	"testing"

	quantadmingo "github.com/quantcdn/quant-admin-go"
	"github.com/stretchr/testify/assert"
)

func TestListDomains(t *testing.T) {
	bearer := os.Getenv("QUANT_BEARER")
	cfg := quantadmingo.NewConfiguration()
	client := quantadmingo.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), quantadmingo.ContextAccessToken, bearer)

	domains, _, err := client.DomainsAPI.DomainsList(ctx, "quant", "api-test").Execute()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	for _, domain := range domains {
		assert.Contains(t, domain.GetDomain(), "quantcdn")
	}
}
